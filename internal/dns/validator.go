package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

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

func (v* Validator) findSPFRecord(ctx context.Context, domainName string) (bool, string, error) {

	// get all txt registers of given domain
	txtRecords, err := v.resolver.LookupTXT(ctx, domainName)
	if err != nil {

		return false, "", fmt.Errorf("failed consulting TXT register for the domain %s: %w", domainName, err)
	}

	for _, record := range txtRecords {

		if strings.HasPrefix(record, "v=spf1") {
			return true, record, nil
		}
	}

	return false, "", domain.ErrInvalidDomain
}

func (v* Validator) getTXTRecords(ctx context.Context, domainName string) ([]string, error) {

	return v.resolver.LookupTXT(ctx, domainName)

}

func parseRecord(record string) map[string] string {


	// TODO estudar o que é esse make
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

func (v* Validator) validateDKIM(ctx context.Context, domainName string, selector string) (bool, error) {

	dkimDomain := buildDKIMDomain(domainName, selector)

	txtRecords, err := v.getTXTRecords(ctx, dkimDomain)

	// fmt.Printf("DKIM domain: %s\n", dkimDomain)
	// fmt.Printf("TXT records: %v\n", txtRecords)
	// fmt.Printf("Error: %v\n", err)

	if err != nil || len(txtRecords) == 0 {

		return false, domain.ErrInvalidDKIM // TODO separate error from empty dkim records
	
	}

	for _, record := range txtRecords {

		dkimMap := parseRecord(record)

		if dkimMap["p"] != "" {
			return true, nil
		}
	}

	return false, domain.ErrInvalidDKIM

}


// devo retornar erro quando p != none, reject, quarantine?
func(v *Validator) findDMARC(ctx context.Context, domainName string) (bool, string) {

	dmarcDomain :=  "_dmarc." + domainName
	txtRecords, err := v.resolver.LookupTXT(ctx, dmarcDomain)
	
	fmt.Println(txtRecords)

	if err != nil || len(txtRecords) == 0 {

		return false, ""
	}

	for _, record := range txtRecords {

		dmarcMap := parseRecord(record)

		if dmarcMap["v"] == "DMARC1" {
			return true, dmarcMap["p"]
		}

	}

	return false, ""


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

	if !result.HasSPF {
		return result, err
	}

	result.HasDKIM, err = v.validateDKIM(ctx, domainName, selector)

	if !result.HasDKIM {
		return result, err
	}

	result.HasDMARC, result.DMARCRecord = v.findDMARC(ctx, domainName)

	return result, nil
}