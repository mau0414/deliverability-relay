package main

import (
	"context"
	"log"

	"github.com/mau0414/deliverability-relay/internal/api"
	"github.com/mau0414/deliverability-relay/internal/queue"
	"github.com/mau0414/deliverability-relay/internal/worker"
	"github.com/mau0414/deliverability-relay/internal/dns"
	"github.com/mau0414/deliverability-relay/internal/repository"
	"github.com/mau0414/deliverability-relay/internal/cache"
	"github.com/jackc/pgx/v5/pgxpool"
	
)

func main() {

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://mta:mta_dev_password@localhost:5433/mta") // TODO colocar isso num .env?
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()
	
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("postgres ping failed: %v", err)
	}

	emailRepository := repository.NewEmailRepository(pool)
	domainRepository := repository.NewDomainRepository(pool)

	redisClient := cache.NewRedis()

	if err := redisClient.Ping(context.Background()); err != nil {
		log.Fatalf("failed to connect to Redis: %v", err)
	}

	q := queue.NewMemoryQueue(100)

	dnsValidator := dns.NewValidator()

	// workers creating and start
	w := worker.NewWorker(q, "deliverability-relay.local", emailRepository)
	go w.Start(ctx)

	// server creation and start
	server := api.NewServer(q, emailRepository, domainRepository, dnsValidator, redisClient)

	addr := ":8080"
	log.Printf("server listening in %s", addr)

	if err := server.ListenAndServe(addr); err != nil {
		log.Printf("error in initializing server")
		log.Fatal(err)
	}

}