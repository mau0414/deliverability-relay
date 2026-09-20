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