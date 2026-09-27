package domain

import ("errors"
		"fmt"
		"strings")

const MaxDeliveryAttempts = 5

var (
    ErrInvalidDomain  = errors.New("domain validation failed: missing or invalid SPF record")
    ErrInvalidDKIM    = errors.New("dkim validation failed")
	ErrDomainNotFound    = errors.New("given domain Not Found")
)
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

func ParseDomain(emailAddr string) (string, error) {

	parts := strings.Split(emailAddr, "@")

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {

		return "", errors.New("invalid email format")

	}

	return parts[1], nil

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
	if _, err := ParseDomain(e.From); err != nil {
		return errors.New("sender email has an invalid format")
	} 
	for _, to := range e.To {

		if a, err := ParseDomain(to); err != nil {
			fmt.Println(a)
			return errors.New("receiver email has an invalid format")
		} 
		
	}
	
	return nil
}