package smtp

import ("net/smtp"
		"fmt"
		"crypto/tls"
		"strings"
		"log"
		"net"
	
		"github.com/mau0414/deliverability-relay/internal/domain")

type Connection struct {

	addr string
	client *smtp.Client

}

func NewConnection(addr string) (*Connection, error) {

	smtpClient, err := smtp.Dial(addr) 
	
	if err != nil {
		return nil, fmt.Errorf("SMTP Connection to address %s failed: %w", addr, err)
	}
	
	return &Connection{
		addr: addr,
		client: smtpClient,
	}, nil

}

func protocolFormat(email domain.Email) string {
	toHeader := strings.Join(email.To, ", ")

	return fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		email.From,
		toHeader,
		email.Subject,
		email.HTML,
	)
}

func (c *Connection) Deliver(mtaDomain string, email domain.Email) error {

	// to guarantee connection close afterwards
	defer c.client.Close()

	// Hello to server
	if err := c.client.Hello(mtaDomain); err != nil {
		return fmt.Errorf("HELO command failed: %w", err)
	}

	host, _, err := net.SplitHostPort(c.addr)
	if err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}
	
	// StartTLS to make content encrypted 
	if ok, _ := c.client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{ServerName: host}
		if err := c.client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS command failed: %w", err)
    }
}

	// inform from to server
	if err := c.client.Mail(email.From); err != nil {
		return fmt.Errorf("FROM command failed: %w", err)
	} 

	// Inform each recipient to server
	for _, to := range email.To {
		if err := c.client.Rcpt(to); err != nil {
			return fmt.Errorf("TO command failed: %w", err)
		}
	}

	// DATA command to get writer
	formattedEmail := protocolFormat(email)
	writer, err := c.client.Data()

	if err != nil {
		return fmt.Errorf("DATA command failed: %w", err)
	}

	// writing email at correct protocol format
	if _, err := writer.Write([]byte(formattedEmail)); err != nil {
			return fmt.Errorf("failed to write email body: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close data writer: %w", err)
	}

	// quit connection to server
	if err := c.client.Quit(); err != nil {
		log.Printf("warning: QUIT failed (email likely already delivered): %v", err)
	}

	return nil

}