package dns

import (
	"context"
	"net"
	"errors"

)

type MXResolver struct {
	resolver *net.Resolver
}

func NewMXResolver() *MXResolver {

	return &MXResolver{
		resolver: net.DefaultResolver,
	}

}

func (r *MXResolver) ResolveMXRecord(ctx context.Context, domainName string) (string, error) {

	mxRecords, err := r.resolver.LookupMX(ctx, domainName)

	if err != nil {
		return "", err
	}

	if len(mxRecords) == 0 {
		return "", errors.New("no MX records found for receiver domainName")
	}

	return mxRecords[0].Host, nil

}