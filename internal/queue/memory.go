package queue

import ("errors"

		"github.com/mau0414/deliverability-relay/internal/domain")

type MemoryQueue struct {

	queue chan domain.Email

}

func NewMemoryQueue(bufferSize int) *MemoryQueue {

	newQueue := &MemoryQueue{
		queue: make(chan domain.Email, bufferSize),
	}

	return newQueue	

}

func (m *MemoryQueue) Enqueue(email domain.Email) error {

	select {
	case m.queue <- email:
		return nil
	default:
		return errors.New("queue is full")
	}

}

// todo aqui precisa de algum tratamento de erro?
func (m *MemoryQueue) Dequeue() (domain.Email, error) {

	nextEmail := <-m.queue

	return nextEmail, nil

}