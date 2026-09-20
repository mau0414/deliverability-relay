package dns

import "testing"

func TestParseReceiverDomain(t *testing.T) {
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
			gotDomain, err := ParseReceiverDomain(tt.email)

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