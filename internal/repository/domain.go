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
		INSERT INTO domains (id, name, has_spf, spf_record, has_dkim, dkim_record, has_dmarc, dmarc_record, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, created_at = now(), updated_at = now())
	`

	_, err := r.pool.Exec(ctx, query,
		domain.ID,
		domain.Name,
		domain.HasSPF,
		domain.SPFRecord,
		domain.HasDKIM,
		domain.HasDMARC,
		domain.HasDMARC,
	)

	return err
}
