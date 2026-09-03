package calendar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
)

func TestVerifyUsesBasicAuthForAppPassword(t *testing.T) {
	var authHeaders []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || user != "user@example.com" || password != "secret" {
			t.Errorf("basic auth = (%q, %q, %v)", user, password, ok)
		}
		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusMultiStatus)
		switch r.URL.Path {
		case "/":
			w.Write([]byte(`<multistatus xmlns="DAV:"><response><href>/</href><propstat><prop><current-user-principal><href>/principals/user/</href></current-user-principal></prop></propstat></response></multistatus>`))
		case "/principals/user/":
			w.Write([]byte(`<multistatus xmlns="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav"><response><href>/principals/user/</href><propstat><prop><C:calendar-home-set><href>/calendars/user/</href></C:calendar-home-set></prop></propstat></response></multistatus>`))
		default:
			w.Write([]byte(`<multistatus xmlns="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav"><response><href>/calendars/user/events/</href><propstat><prop><resourcetype><collection/><C:calendar/></resourcetype></prop></propstat></response></multistatus>`))
		}
	}))
	defer srv.Close()

	cred, err := auth.NewAppPasswordCredential("user@example.com", "secret")
	if err != nil {
		t.Fatalf("NewAppPasswordCredential: %v", err)
	}
	svc := NewService(config.Calendar{BaseURL: srv.URL, Scope: config.CalendarScope}, cred)
	if err := svc.Verify(context.Background()); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(authHeaders) != 3 {
		t.Fatalf("requests = %d, want 3", len(authHeaders))
	}
}

func TestRequestRejectsUnusableCredential(t *testing.T) {
	svc := NewService(config.Calendar{BaseURL: "http://127.0.0.1:0", Scope: config.CalendarScope}, &auth.Credential{AccessToken: "x", Scope: "login:info"})
	if err := svc.Verify(context.Background()); err != ErrReauthRequired {
		t.Fatalf("Verify error = %v, want ErrReauthRequired", err)
	}
}
