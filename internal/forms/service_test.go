package forms

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
)

func TestListResponsesRequiresReadScope(t *testing.T) {
	t.Parallel()

	cfg := config.Forms{BaseURL: "http://example.invalid", OrgID: "org1", ReadScope: config.FormsReadScope, WriteScope: config.FormsWriteScope}
	cred := &auth.Credential{AccessToken: "token", Scope: "", Expiry: time.Now().Add(time.Hour)}
	svc := NewService(cfg, cred)

	_, err := svc.ListResponses(context.Background(), "survey-1", 0, "")
	if !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("ListResponses() error = %v, want ErrReauthRequired", err)
	}
}

func TestListResponsesDecodesTolerantAnswers(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Org-Id"); got != "7023313" {
			t.Errorf("X-Org-Id header = %q, want 7023313", got)
		}
		if got := r.Header.Get("Authorization"); got != "OAuth token" {
			t.Errorf("Authorization header = %q, want OAuth token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"answers":[{"id":"a1","respondent_id":"r1","submitted_at":"2026-06-01T00:00:00Z","extra_field":"unparsed"}],"next":"cursor-2"}`))
	}))
	t.Cleanup(server.Close)

	cfg := config.Forms{BaseURL: server.URL, OrgID: "7023313", ReadScope: config.FormsReadScope, WriteScope: config.FormsWriteScope}
	cred := &auth.Credential{AccessToken: "token", Scope: config.FormsReadScope, Expiry: time.Now().Add(time.Hour)}
	svc := NewService(cfg, cred)

	result, err := svc.ListResponses(context.Background(), "survey-1", 10, "")
	if err != nil {
		t.Fatalf("ListResponses() error = %v", err)
	}
	if len(result.Answers) != 1 {
		t.Fatalf("Answers = %v, want 1", result.Answers)
	}
	answer := result.Answers[0]
	if answer.ID != "a1" || answer.RespondentID != "r1" || answer.SubmittedAt != "2026-06-01T00:00:00Z" {
		t.Fatalf("answer = %+v, unexpected fields", answer)
	}
	if answer.Raw["extra_field"] != "unparsed" {
		t.Fatalf("Raw fallback missing extra_field: %+v", answer.Raw)
	}
	if result.NextPageToken != "cursor-2" {
		t.Fatalf("NextPageToken = %q, want cursor-2", result.NextPageToken)
	}
}

func TestDoReturnsReauthOn401(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	cfg := config.Forms{BaseURL: server.URL, OrgID: "7023313", ReadScope: config.FormsReadScope, WriteScope: config.FormsWriteScope}
	cred := &auth.Credential{AccessToken: "token", Scope: config.FormsReadScope, Expiry: time.Now().Add(time.Hour)}
	svc := NewService(cfg, cred)

	_, err := svc.ListResponses(context.Background(), "survey-1", 0, "")
	if !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("ListResponses() error = %v, want ErrReauthRequired", err)
	}
}

func TestCreateSurveyRequiresWriteScope(t *testing.T) {
	t.Parallel()

	cfg := config.Forms{BaseURL: "http://example.invalid", OrgID: "org1", ReadScope: config.FormsReadScope, WriteScope: config.FormsWriteScope}
	cred := &auth.Credential{AccessToken: "token", Scope: config.FormsReadScope, Expiry: time.Now().Add(time.Hour)}
	svc := NewService(cfg, cred)

	_, err := svc.CreateSurvey(context.Background(), "My Survey")
	if !errors.Is(err, ErrReauthRequired) {
		t.Fatalf("CreateSurvey() error = %v, want ErrReauthRequired", err)
	}
}

func TestPublishRequiresOrgID(t *testing.T) {
	t.Parallel()

	cfg := config.Forms{BaseURL: "http://example.invalid", OrgID: "", ReadScope: config.FormsReadScope, WriteScope: config.FormsWriteScope}
	cred := &auth.Credential{AccessToken: "token", Scope: config.FormsWriteScope, Expiry: time.Now().Add(time.Hour)}
	svc := NewService(cfg, cred)

	_, err := svc.Publish(context.Background(), "survey-1")
	if err == nil {
		t.Fatal("Publish() error = nil, want org id error")
	}
}

func TestCloudOrgIDUsesCloudHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Cloud-Org-Id"); got != "c80b932aa5e94c01b6bd368df0bc3e36" {
			t.Errorf("X-Cloud-Org-Id header = %q, want the cloud org id", got)
		}
		if got := r.Header.Get("X-Org-Id"); got != "" {
			t.Errorf("X-Org-Id header = %q, want empty for a cloud org id", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"answers":[],"next":""}`))
	}))
	t.Cleanup(server.Close)

	cfg := config.Forms{BaseURL: server.URL, OrgID: "c80b932aa5e94c01b6bd368df0bc3e36", ReadScope: config.FormsReadScope, WriteScope: config.FormsWriteScope}
	cred := &auth.Credential{AccessToken: "token", Scope: config.FormsReadScope, Expiry: time.Now().Add(time.Hour)}
	svc := NewService(cfg, cred)

	if _, err := svc.ListResponses(context.Background(), "survey-1", 0, ""); err != nil {
		t.Fatalf("ListResponses() error = %v", err)
	}
}
