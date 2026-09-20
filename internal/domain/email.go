package domain

import "errors"

const MaxDeliveryAttempts = 5

var (
    ErrInvalidDomain  = errors.New("domain validation failed: missing or invalid SPF record")
    ErrInvalidDKIM    = errors.New("dkim validation failed: missing or empty public key (p= tag)")
)
// ValidationResult - keeps status of dns check
type ValidationResult struct {
	Domain      string
	HasSPF      bool
	HasDKIM     bool
	SPFRecord   string
	HasDMARC    bool
	DMARCRecord string
}

type Status string

const (
	StatusQueued  Status = "queued"
	StatusSending Status = "sending"
	StatusSent    Status = "sent"
	StatusBounced Status = "bounced"
	StatusFailed  Status = "failed"
)
type Email struct {
	ID string
	From string
	To []string
	Subject string
	HTML string
	Status Status
	Attempts int
}

func (e Email) Validate() error {
	
	if len(e.To) == 0 {
		return errors.New("missing required field: to")
	}
	if e.From == "" {
		return errors.New("missing required field: from")
	}
	if e.Subject == ""{
		return errors.New("missing required field: subject")
	}
	
	return nil
}