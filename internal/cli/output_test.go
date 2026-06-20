package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
)

func TestJSONNeverLeaksToken(t *testing.T) {
	cred := &auth.Credential{
		AccessToken:  "SECRET-ACCESS-TOKEN",
		RefreshToken: "SECRET-REFRESH-TOKEN",
		TokenType:    "bearer",
		Account:      "user@yandex.ru",
	}
	payload := loginPayload{
		Status:  "logged-in",
		Account: cred.Account,
		Scopes:  []string{"login:info"},
	}

	jsonOutput = true
	t.Cleanup(func() { jsonOutput = false })

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)

	if err := emit(cmd, "human", payload); err != nil {
		t.Fatalf("emit: %v", err)
	}

	got := out.String()
	for _, secret := range []string{cred.AccessToken, cred.RefreshToken} {
		if strings.Contains(got, secret) {
			t.Fatalf("JSON output leaked a token: %q", got)
		}
	}
	if !strings.Contains(got, "user@yandex.ru") {
		t.Fatalf("expected account in payload, got %q", got)
	}
}
