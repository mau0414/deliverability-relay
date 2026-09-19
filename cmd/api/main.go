package main

import (
	// "context"
	// "errors"
	// "fmt"
	"log"

	// "github.com/mau0414/deliverability-relay/internal/dns"
	// "github.com/mau0414/deliverability-relay/internal/domain"
	"github.com/mau0414/deliverability-relay/internal/api"
	"github.com/mau0414/deliverability-relay/internal/queue"
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


	q := queue.NewMemoryQueue(100)
	server := api.NewServer(q)

	addr := ":8080"
	log.Printf("server listening in %s", addr)

	if err := server.ListenAndServe(addr); err != nil {
		log.Printf("error in initializing server")
		log.Fatal(err)
	}

}