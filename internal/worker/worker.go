package worker

import (
	"context"
	"log"
	"time"

	"github.com/mau0414/deliverability-relay/internal/dns"
	"github.com/mau0414/deliverability-relay/internal/domain"
	"github.com/mau0414/deliverability-relay/internal/queue"
	"github.com/mau0414/deliverability-relay/internal/smtp"
	"github.com/mau0414/deliverability-relay/internal/repository"
)

type Worker struct {

	queue queue.Queue
	mxResolver *dns.MXResolver
	mtaDomain string
	repository *repository.EmailRepository
}

func NewWorker(q queue.Queue, mtaDomain string, r *repository.EmailRepository) *Worker {

	return &Worker{
		queue: q,
		mxResolver: dns.NewMXResolver(),
		mtaDomain: mtaDomain,
		repository: r,
	}	

}


func (w *Worker) handleDeliveryFailure(ctx context.Context, email domain.Email, to string, err error) {

	if smtp.IsPermanentFailure(err) {
		log.Printf("permanent failure for delivering to %s: %v", to, err)
		w.updateStatus(ctx, email, domain.StatusBounced)
		return
	}

	email.Attempts++

	if email.Attempts >= domain.MaxDeliveryAttempts {
		log.Printf("giving up on delivering to %s after %d attempts: %v", to, email.Attempts, err)
		w.updateStatus(ctx, email, domain.StatusFailed)
		return
	}

	log.Printf("temporary failure for delivering to %s (attempt %d), requeueing: %v", to, email.Attempts, err)
	if err := w.queue.Enqueue(email); err != nil {
		log.Printf("failed to requeue email %s: %v", email.ID, err)
		w.updateStatus(ctx, email, domain.StatusFailed)
	}
}

func (w *Worker) updateStatus(ctx context.Context, email domain.Email, newStatus domain.Status) {

	updateCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := w.repository.UpdateStatus(updateCtx, email, newStatus); err != nil {
		log.Printf("failed to update status to %s for email %s: %v", newStatus, email.ID, err)
	}
}

func (w *Worker) Start(ctx context.Context) {

	for {

		email, err := w.queue.Dequeue()

		if err != nil {
			log.Printf("error popping out of the queue: %v", err)
			continue
		}

		log.Printf("processing email: %+v", email)

		for _, to := range email.To {

			// toDomain, err := dns.ParseReceiverDomain(to)

			// if err != nil {
			// 	log.Printf("invalid recipient address %s: %v", to, err)
			// 	continue
			// }	

			// resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

			// rcpt, err := w.mxResolver.ResolveMXRecord(resolveCtx, toDomain)
			// cancel()

			// if err != nil {
			// 	log.Printf("failed to resolve MX for %s: %v", to, err)
			// 	w.handleDeliveryFailure(ctx, email, to, err)
			// 	continue
			// }

			// addr := rcpt + ":25"
			addr := "localhost:1025" // - uncomment to test at mailpit

			connection, err := smtp.NewConnection(addr)

			if err != nil {
				log.Printf("failed to connect to %s: %v", addr, err)
				w.handleDeliveryFailure(ctx, email, to, err)
				continue
			}

			if err := connection.Deliver(w.mtaDomain, email); err != nil {
				log.Printf("failed to deliver to %s: %v", to, err)
				w.handleDeliveryFailure(ctx, email, to, err)
				continue
			}

			w.updateStatus(ctx, email, domain.StatusSent)

			log.Printf("delivered to %s successfully", to)

		}
	}
}