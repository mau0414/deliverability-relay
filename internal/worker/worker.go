package worker

import ("log"
		"context"
		"time"

		"github.com/mau0414/deliverability-relay/internal/queue"
		"github.com/mau0414/deliverability-relay/internal/smtp"
		"github.com/mau0414/deliverability-relay/internal/dns"
)

type Worker struct {

	queue queue.Queue
	mxResolver *dns.MXResolver
	mtaDomain string
}

func NewWorker(q queue.Queue, mtaDomain string) *Worker {

	return &Worker{
		queue: q,
		mxResolver: dns.NewMXResolver(),
		mtaDomain: mtaDomain,
	}	

}

// func obtainAddress()

func (w *Worker) Start(ctx context.Context) {

	for {

		email, err := w.queue.Dequeue()

		if err != nil {
			log.Printf("error popping out of the queue: %v", err)
			continue
		}

		log.Printf("processing email: %+v", email)

		for _, to := range email.To {

			toDomain, err := dns.ParseReceiverDomain(to)

			if err != nil {
				log.Printf("invalid recipient address %s: %v", to, err)
				continue
			}	

			resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			cancel()

			rcpt, err := w.mxResolver.ResolveMXRecord(resolveCtx, toDomain)

			if err != nil {

				log.Printf("failed to resolve MX for %s: %v", to, err)
				continue

			}

			addr := rcpt + ":25"
			// addr := "localhost:1025" - uncomment to test at mailpit

			connection, err := smtp.NewConnection(addr)

			if err != nil {

				log.Printf("failed to connect to %s: %v", addr, err)
				continue
			
			}

			if err := connection.Deliver(w.mtaDomain, email); err != nil {

				log.Printf("failed to deliver to %s: %v", to, err)
				continue

			}
			

		}
		

	}

}