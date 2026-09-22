package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mau0414/deliverability-relay/internal/domain"
)

type EmailRepository struct {
	pool *pgxpool.Pool
}

func NewEmailRepository(pool *pgxpool.Pool) *EmailRepository {
	return &EmailRepository{pool: pool}
}

func (r *EmailRepository) Save(ctx context.Context, email domain.Email) error {
	query := `
		INSERT INTO emails (id, from_address, to_addresses, subject, html, status, attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.pool.Exec(ctx, query,
		email.ID,
		email.From,
		email.To,
		email.Subject,
		email.HTML,
		email.Status,
		email.Attempts,
	)

	return err
}

func (r *EmailRepository) UpdateStatus(ctx context.Context, email domain.Email, newStatus domain.Status) error {
	query := `
		UPDATE emails
		SET status = $1, attempts = $2, updated_at = now()
		WHERE id = $3
	`

	_, err := r.pool.Exec(ctx, query,
		newStatus,
		email.Attempts,
		email.ID,
	)

	return err
}