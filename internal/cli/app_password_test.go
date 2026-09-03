package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
)

func TestAppPasswordProfileRouting(t *testing.T) {
	tests := []struct {
		name                                                                          string
		mailScope, mailSendScope, calendarScope, telemostScope, formsScope, diskScope bool
		want                                                                          string
		wantErr                                                                       string
	}{
		{name: "mail", mailScope: true, want: mailAppPasswordProfile},
		{name: "mail send only", mailSendScope: true, want: mailAppPasswordProfile},
		{name: "calendar", calendarScope: true, want: calendarAppPasswordProfile},
		{name: "no surface", wantErr: "requires --mail or --calendar"},
		{name: "both surfaces", mailScope: true, calendarScope: true, wantErr: "one app password per service"},
		{name: "disk", diskScope: true, wantErr: "REST APIs that accept OAuth tokens only"},
		{name: "forms", formsScope: true, wantErr: "REST APIs that accept OAuth tokens only"},
		{name: "telemost", telemostScope: true, wantErr: "REST APIs that accept OAuth tokens only"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := appPasswordProfile(tt.mailScope, tt.mailSendScope, tt.calendarScope, tt.telemostScope, tt.formsScope, tt.diskScope)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("appPasswordProfile: %v", err)
			}
			if got != tt.want {
				t.Fatalf("profile = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadAppPassword(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetIn(strings.NewReader("piped-secret\n"))
	cmd.SetErr(&bytes.Buffer{})
	got, err := readAppPassword(cmd)
	if err != nil {
		t.Fatalf("readAppPassword: %v", err)
	}
	if got != "piped-secret" {
		t.Fatalf("piped password = %q", got)
	}

	t.Setenv("YX360_APP_PASSWORD", "env-secret")
	got, err = readAppPassword(cmd)
	if err != nil {
		t.Fatalf("readAppPassword: %v", err)
	}
	if got != "env-secret" {
		t.Fatalf("env password = %q", got)
	}
}

func TestLoadCredentialPrefersAppPassword(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	insecureFileStore = true
	t.Cleanup(func() { insecureFileStore = false })
	ctx := context.Background()

	oauthStore, err := selectStoreFor(mailProfile)
	if err != nil {
		t.Fatalf("selectStoreFor: %v", err)
	}
	oauth := &auth.Credential{AccessToken: "token", Account: "oauth@example.com", Scope: "mail:imap_full", ObtainedVia: auth.GrantLoopback}
	if err := oauthStore.Save(ctx, oauth); err != nil {
		t.Fatalf("save oauth: %v", err)
	}

	cred, err := loadCredential(ctx, mailAppPasswordProfile, mailProfile)
	if err != nil {
		t.Fatalf("loadCredential: %v", err)
	}
	if cred.Account != "oauth@example.com" {
		t.Fatalf("account = %q, want the OAuth credential", cred.Account)
	}

	appStore, err := selectStoreFor(mailAppPasswordProfile)
	if err != nil {
		t.Fatalf("selectStoreFor: %v", err)
	}
	appCred, err := auth.NewAppPasswordCredential("app@example.com", "secret")
	if err != nil {
		t.Fatalf("NewAppPasswordCredential: %v", err)
	}
	if err := appStore.Save(ctx, appCred); err != nil {
		t.Fatalf("save app password: %v", err)
	}

	cred, err = loadCredential(ctx, mailAppPasswordProfile, mailProfile)
	if err != nil {
		t.Fatalf("loadCredential: %v", err)
	}
	if cred.Account != "app@example.com" || !cred.IsAppPassword() {
		t.Fatalf("credential = %+v, want the stored app password", cred)
	}
}
