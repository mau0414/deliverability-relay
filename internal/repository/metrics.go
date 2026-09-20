package repository

import (
	"context"
	"time"

	"github.com/mau0414/deliverability-relay/internal/domain"
)


type StatusCount struct {
	Status domain.Status
	Count  int
}

type DailyVolume struct {
	Date  time.Time
	Count int
}

type Metrics struct {
	ByStatus []StatusCount
	Daily    []DailyVolume
}

func (r *EmailRepository) GetMetrics(ctx context.Context) (Metrics, error) {
	var metrics Metrics

	statusRows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*)
		FROM emails
		GROUP BY status
	`)
	if err != nil {
		return metrics, err
	}
	defer statusRows.Close()

	for statusRows.Next() {
		var sc StatusCount
		if err := statusRows.Scan(&sc.Status, &sc.Count); err != nil {
			return metrics, err
		}
		metrics.ByStatus = append(metrics.ByStatus, sc)
	}
	if err := statusRows.Err(); err != nil {
		return metrics, err
	}

	dailyRows, err := r.pool.Query(ctx, `
		SELECT date_trunc('day', created_at) AS day, COUNT(*)
		FROM emails
		GROUP BY day
		ORDER BY day
	`)
	if err != nil {
		return metrics, err
	}
	defer dailyRows.Close()

	for dailyRows.Next() {
		var dv DailyVolume
		if err := dailyRows.Scan(&dv.Date, &dv.Count); err != nil {
			return metrics, err
		}
		metrics.Daily = append(metrics.Daily, dv)
	}
	if err := dailyRows.Err(); err != nil {
		return metrics, err
	}

	return metrics, nil
}