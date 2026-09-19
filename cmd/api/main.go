package main

import (
	"context"
	// "errors"
	"fmt"
	"log"

	// "github.com/mau0414/deliverability-relay/internal/dns"
	// "github.com/mau0414/deliverability-relay/internal/domain"
	"github.com/mau0414/deliverability-relay/internal/api"
	"github.com/mau0414/deliverability-relay/internal/queue"
	"github.com/mau0414/deliverability-relay/internal/worker"
	"github.com/mau0414/deliverability-relay/internal/dns"
	
)

func main() {
	// validator := dns.NewValidator()
	// ctx := context.Background()

	// domainToTest := "google.com"

	// fmt.Printf("validating: %s...\n", domainToTest)
	// result, err := validator.ValidateDomain(ctx, "stripe.com", "google")

	// if err != nil {
	// 	if errors.Is(err, domain.ErrInvalidDomain) {
	// 		fmt.Printf("Domain %s does not have a valid SPF record!\n", domainToTest)
	// 	} else {
	// 		log.Fatalf("Network error: %v\n", err)
	// 	}
	// 	return
	// }

	// fmt.Println("Valid domain")
	// fmt.Printf("   • Domain: %s\n", result.Domain)
	// fmt.Printf("   • Does it have SPF?: %t\n", result.HasSPF)
	// fmt.Printf("   • Record content: %s\n", result.SPFRecord)
	// fmt.Printf("   • Does it have DMARC?: %t\n", result.HasDMARC)

	// TODO remove later - test of mx record resolver
	r := dns.NewMXResolver()
	host, err := r.ResolveMXRecord(context.Background(), "gmail.com")
	fmt.Println(host, err)

	q := queue.NewMemoryQueue(100)

	// workers creating and start
	w := worker.NewWorker(q, "deliverability-relay.local")
	go w.Start(context.Background())

	// server creation and start
	server := api.NewServer(q)

	addr := ":8080"
	log.Printf("server listening in %s", addr)

	if err := server.ListenAndServe(addr); err != nil {
		log.Printf("error in initializing server")
		log.Fatal(err)
	}

}