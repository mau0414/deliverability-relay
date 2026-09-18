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

func parseSPFRecord(records []string) (bool, string) {

	for _, record := range records {

		if strings.HasPrefix(record, "v=spf1") {
			return true, record
		}
	}

	return false, ""
}

func (v *Validator) ValidateDomain(ctx context.Context, domainName string) (*domain.ValidationResult, error) {
	// 3 second timeout so network consult doesn't stall application
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// get all txt registers of given domain
	txtRecords, err := v.resolver.LookupTXT(ctx, domainName)
	if err != nil {

		return nil, fmt.Errorf("failed consulting TXT register for the domain %s: %w", domainName, err)
	}

	result := &domain.ValidationResult{
		Domain: domainName,
	}

	result.HasSPF, result.SPFRecord = validateSPF(result, txtRecords)

	if !result.HasSPF {
		return result, domain.ErrInvalidDomain
	}

	return result, nil
}