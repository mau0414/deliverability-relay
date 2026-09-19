package queue

import "github.com/mau0414/deliverability-relay/internal/domain"

type Queue interface {
	Enqueue(email domain.Email) error
	Dequeue() (domain.Email, error)
}