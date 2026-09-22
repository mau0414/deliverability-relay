package api

import ("github.com/mau0414/deliverability-relay/internal/domain"
		"github.com/google/uuid")

type SendEmailRequest struct {
	From string `json:"from"`
	To []string `json:"to"`
	Subject string `json:"subject"`
	HTML string `json:"html"`
}

type SendEmailResponse struct {
    ID string `json:"id"`
}

func (req SendEmailRequest) ToEmailDomain() domain.Email {

	return domain.Email{
		ID: uuid.New().String(),
		From: req.From,
		To: req.To,
		Subject: req.Subject,
		HTML: req.HTML,
		Status: domain.StatusQueued,
		Attempts: 0,
	}

}

type CreateDomainRequest struct {
	Name string
}

type CreateDomainResponse struct {
	ID string
}

func (req CreateDomainRequest) ToDomainDomain(HasSPF bool, SPFRecord string, HasDKIM bool, HasDMARC bool, DMARCRecord string) domain.Domain {

	return domain.Domain{
		ID: uuid.New().String(),
		Name: req.Name,
		HasSPF: HasSPF,
		SPFRecord: SPFRecord,
		HasDKIM: HasDKIM,
		HasDMARC: HasDMARC,
		DMARCRecord: DMARCRecord,
	}
}
