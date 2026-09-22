package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/mau0414/deliverability-relay/internal/domain"
	"github.com/redis/go-redis/v9"
)

func domainKey(name string) string {
	return "domain:" + name
}

func (r *Redis) SetDomain(
	ctx context.Context,
	d domain.Domain,
) error {
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}

	return r.client.Set(
		ctx,
		domainKey(d.Name),
		data,
		10*time.Minute,
	).Err()
}

func (r *Redis) GetDomain(
	ctx context.Context,
	name string,
) (*domain.Domain, error) {

	data, err := r.client.Get(
		ctx,
		domainKey(name),
	).Bytes()

	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}

		return nil, err
	}

	var d domain.Domain

	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	return &d, nil
}