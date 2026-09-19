package worker

import ("log"

		"github.com/mau0414/deliverability-relay/internal/queue"
)

type Worker struct {

	queue queue.Queue

}

func NewWorker(q queue.Queue) *Worker {

	return &Worker{
		queue: q,
	}	

}

func (w *Worker) Start() {

	for {

		email, err := w.queue.Dequeue()

		if err != nil {
			log.Printf("error popping out of the queue: %v", err)
			continue
		}

		log.Printf("processing email: %+v", email)

	}

}