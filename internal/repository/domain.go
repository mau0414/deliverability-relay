package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mau0414/deliverability-relay/internal/domain"
)

type DomainRepository struct {
	pool *pgxpool.Pool
}

func NewDomainRepository(pool *pgxpool.Pool) *DomainRepository {
	return &DomainRepository{pool: pool}
}

func (r *DomainRepository) Save(ctx context.Context, domain domain.Domain) error {
	query := `
		INSERT INTO domains (id, name, has_spf, spf_record, has_dkim, has_dmarc, dmarc_record)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		domain.ID,
		domain.Name,
		domain.HasSPF,
		domain.SPFRecord,
		domain.HasDKIM,
		domain.HasDMARC,
		domain.DMARCRecord,
	)

	return err
}

func (r *DomainRepository) FindDomain(ctx context.Context, name string) (*domain.Domain, error) {
	query := `
		SELECT id, name, has_spf, spf_record, has_dkim, has_dmarc, dmarc_record
		FROM domains
		WHERE name = $1
	`

	var d domain.Domain

	err := r.pool.QueryRow(ctx, query, name).Scan(
		&d.ID,
		&d.Name,
		&d.HasSPF,
		&d.SPFRecord,
		&d.HasDKIM,
		&d.HasDMARC,
		&d.DMARCRecord,
	)
	if err != nil {
		return nil, err
	}

	return &d, nil
}