package main

import (
	"context"
	"log"
	"os"
	"flag"

	"github.com/mau0414/deliverability-relay/internal/api"
	"github.com/mau0414/deliverability-relay/internal/queue"
	"github.com/mau0414/deliverability-relay/internal/worker"
	"github.com/mau0414/deliverability-relay/internal/dns"
	"github.com/mau0414/deliverability-relay/internal/repository"
	"github.com/mau0414/deliverability-relay/internal/cache"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on real environment variables")
	}

	envTest := flag.Bool("env-test", false, "skip domain SPF/DKIM preflight validation (local testing only)")
	flag.Parse()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
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
	server := api.NewServer(q, emailRepository, domainRepository, dnsValidator, redisClient, *envTest)

	addr := ":8080"
	log.Printf("server listening in %s", addr)

	if err := server.ListenAndServe(addr); err != nil {
		log.Printf("error in initializing server")
		log.Fatal(err)
	}

}