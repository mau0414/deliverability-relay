package domain

import "testing"

func TestEmailValidate(t *testing.T) {
	tests := []struct {
		name    string
		email   Email
		wantErr bool
	}{
		{
			name: "valid email",
			email: Email{
				From:    "sender@example.com",
				To:      []string{"receiver@example.com"},
				Subject: "hello",
			},
			wantErr: false,
		},
		{
			name: "missing from",
			email: Email{
				To:      []string{"receiver@example.com"},
				Subject: "hello",
			},
			wantErr: true,
		},
		{
			name: "missing to",
			email: Email{
				From:    "sender@example.com",
				Subject: "hello",
			},
			wantErr: true,
		},
		{
			name: "empty to slice",
			email: Email{
				From:    "sender@example.com",
				To:      []string{},
				Subject: "hello",
			},
			wantErr: true,
		},
		{
			name: "missing subject",
			email: Email{
				From: "sender@example.com",
				To:   []string{"receiver@example.com"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.email.Validate()

			if tt.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestParseDomain(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		wantDomain string
		wantErr    bool
	}{
		{
			name:       "valid email",
			email:      "user@example.com",
			wantDomain: "example.com",
			wantErr:    false,
		},
		{
			name:       "valid email with subdomain",
			email:      "user@mail.example.com",
			wantDomain: "mail.example.com",
			wantErr:    false,
		},
		{
			name:    "missing @",
			email:   "userexample.com",
			wantErr: true,
		},
		{
			name:    "empty string",
			email:   "",
			wantErr: true,
		},
		{
			name:    "missing domain",
			email:   "user@",
			wantErr: true,
		},
		{
			name:    "missing local part",
			email:   "@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDomain, err := ParseDomain(tt.email)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if gotDomain != tt.wantDomain {
				t.Errorf("expected domain %q, got %q", tt.wantDomain, gotDomain)
			}
		})
	}
}
