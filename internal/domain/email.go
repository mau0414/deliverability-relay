package domain

import "errors"

var ErrInvalidDomain = errors.New("not valid authentication registers for domain")

// ValidationResult - keeps status of dns check
type ValidationResult struct {
	Domain   string
	HasSPF   bool
	HasDKIM  bool
	SPFRecord string
}