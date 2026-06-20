package auth

import (
	"testing"
	"time"
)

func TestCredentialValid(t *testing.T) {
	tests := []struct {
		name string
		cred *Credential
		want bool
	}{
		{"nil", nil, false},
		{"empty token", &Credential{}, false},
		{"no expiry treated as valid", &Credential{AccessToken: "x"}, true},
		{"future expiry", &Credential{AccessToken: "x", Expiry: time.Now().Add(time.Hour)}, true},
		{"past expiry", &Credential{AccessToken: "x", Expiry: time.Now().Add(-time.Hour)}, false},
		{"within skew is invalid", &Credential{AccessToken: "x", Expiry: time.Now().Add(30 * time.Second)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cred.Valid(); got != tt.want {
				t.Fatalf("Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
