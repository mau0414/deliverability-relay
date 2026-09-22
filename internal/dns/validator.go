package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"
	"errors"

	"github.com/mau0414/deliverability-relay/internal/domain"
)

type Validator struct {
	resolver *net.Resolver
}

func NewValidator() *Validator {
	return &Validator{
		resolver: net.DefaultResolver,
	}
}

func buildDKIMDomain(domainName string, selector string) string {
    return selector + "._domainkey." + domainName
} 

func isNXDomain(err error) bool {
	var dnsErr *net.DNSError

    if errors.As(err, &dnsErr) {
        return dnsErr.IsNotFound
    }

    return false
}

func (v* Validator) findSPFRecord(ctx context.Context, domainName string) (bool, string, error) {

	// get all txt registers of given domain
	txtRecords, err := v.resolver.LookupTXT(ctx, domainName)
	if err != nil {

		if isNXDomain(err) {
			return false, "", domain.ErrDomainNotFound
		}

		return false, "", fmt.Errorf("failed consulting TXT register for the domain %s: %w", domainName, err)
	}

	for _, record := range txtRecords {

		if strings.HasPrefix(record, "v=spf1") {
			return true, record, nil
		}
	}

	return false, "", nil
}

func (v* Validator) getTXTRecords(ctx context.Context, domainName string) ([]string, error) {

	return v.resolver.LookupTXT(ctx, domainName)

}

// add test
func parseRecord(record string) map[string]string {

	tags := make(map[string]string)

	parts := strings.Split(record, ";")

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}

		kv := strings.SplitN(trimmed, "=", 2)
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])
			tags[key] = val
		}
	}

	return tags

}

func (v* Validator) findDKIM(ctx context.Context, domainName string, selector string) (bool, error) {

	dkimDomain := buildDKIMDomain(domainName, selector)

	txtRecords, err := v.getTXTRecords(ctx, dkimDomain)

	if err != nil {

		if isNXDomain(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed consulting TXT register for dkim domain %s: %w", dkimDomain, err)
	
	} 

	if len(txtRecords) == 0 {

		return false, nil
	}

	for _, record := range txtRecords {

		dkimMap := parseRecord(record)

		if dkimMap["p"] != "" {
			return true, nil
		}
	}

	return false, nil

}

func(v *Validator) findDMARC(ctx context.Context, domainName string) (bool, string, error) {

	dmarcDomain :=  "_dmarc." + domainName
	txtRecords, err := v.resolver.LookupTXT(ctx, dmarcDomain)

	if err != nil {

		if isNXDomain(err) {
			return false, "", nil
		}

		return false, "", fmt.Errorf("failed consulting TXT register for the dmarc domain %s: %w", dmarcDomain, err)
	}

	if len(txtRecords) == 0 {
		return false, "", nil
	}

	for _, record := range txtRecords {

		dmarcMap := parseRecord(record)

		if dmarcMap["v"] == "DMARC1" {
			return true, dmarcMap["p"], nil
		}

	}

	return false, "", nil


}


func (v *Validator) ValidateDomain(ctx context.Context, domainName string, selector string) (*domain.ValidationResult, error) {
	// 3 second timeout so network consult doesn't stall application
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	result := &domain.ValidationResult{
		Domain: domainName,
	}
	
	var err error

	result.HasSPF, result.SPFRecord, err = v.findSPFRecord(ctx, domainName)
	if err != nil {
		return nil, err
	}
	result.HasDKIM, err = v.findDKIM(ctx, domainName, selector)
	if err != nil {
		return nil, err
	}
	result.HasDMARC, result.DMARCRecord, err = v.findDMARC(ctx, domainName)
	if err != nil {
		return nil, err
	}

	return result, nil
}