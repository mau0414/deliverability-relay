package domain

import "errors"

var (
    ErrInvalidDomain  = errors.New("domain validation failed: missing or invalid SPF record")
    ErrInvalidDKIM    = errors.New("dkim validation failed: missing or empty public key (p= tag)")
)
// ValidationResult - keeps status of dns check
type ValidationResult struct {
	Domain   string
	HasSPF   bool
	HasDKIM  bool
	SPFRecord string
}