package mail

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseUnsubscribeHeadersPreservesOrderAndMethods(t *testing.T) {
	t.Parallel()

	got := ParseUnsubscribeHeaders(
		"<mailto:list@example.com?subject=unsubscribe>, <https://example.com/unsub>, <http://example.net/unsub>",
		"",
	)
	if len(got) != 3 {
		t.Fatalf("options = %v, want 3", got)
	}
	checkOption(t, got[0], UnsubscribeMailto, "mailto:list@example.com?subject=unsubscribe", true, false)
	checkOption(t, got[1], UnsubscribeHTTPSGet, "https://example.com/unsub", false, false)
	checkOption(t, got[2], UnsubscribeHTTPSGet, "http://example.net/unsub", false, false)
}

func TestParseUnsubscribeHeadersOneClickOnlyForHTTPS(t *testing.T) {
	t.Parallel()

	got := ParseUnsubscribeHeaders(
		"<http://example.net/unsub>, <https://example.com/unsub>",
		" List-Unsubscribe=One-Click ",
	)
	if len(got) != 2 {
		t.Fatalf("options = %v, want 2", got)
	}
	checkOption(t, got[0], UnsubscribeHTTPSGet, "http://example.net/unsub", false, false)
	checkOption(t, got[1], UnsubscribeHTTPSPost, "https://example.com/unsub", false, true)
}

func TestParseUnsubscribeHeadersIgnoresUnsupportedSchemes(t *testing.T) {
	t.Parallel()

	got := ParseUnsubscribeHeaders(
		"<file:///tmp/unsubscribe>, <javascript:alert(1)>, <mailto:list@example.com>",
		"",
	)
	if len(got) != 1 {
		t.Fatalf("options = %v, want only mailto", got)
	}
	checkOption(t, got[0], UnsubscribeMailto, "mailto:list@example.com", true, false)
}

func TestParseMailtoUnsubscribe(t *testing.T) {
	t.Parallel()

	got, err := parseMailtoUnsubscribe("mailto:list@example.com?subject=Remove%20me&body=Please%20unsubscribe")
	if err != nil {
		t.Fatalf("parseMailtoUnsubscribe() error = %v", err)
	}
	if len(got.To) != 1 || got.To[0] != "list@example.com" {
		t.Fatalf("To = %v, want list@example.com", got.To)
	}
	if got.Subject != "Remove me" {
		t.Fatalf("Subject = %q, want Remove me", got.Subject)
	}
	if got.Body != "Please unsubscribe" {
		t.Fatalf("Body = %q, want Please unsubscribe", got.Body)
	}
}

func TestExecuteUnsubscribePOSTBlocksRedirect(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/next", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	status, err := executeUnsubscribePOST(context.Background(), server.URL)
	if err == nil {
		t.Fatalf("executeUnsubscribePOST() error = nil, want redirect error")
	}
	if status != http.StatusFound {
		t.Fatalf("status = %d, want %d", status, http.StatusFound)
	}
}

func checkOption(t *testing.T, got UnsubscribeOption, method UnsubscribeMethod, uri string, requiresSMTP bool, oneClick bool) {
	t.Helper()
	if got.Method != method || got.URI != uri || got.RequiresSMTP != requiresSMTP || got.OneClick != oneClick {
		t.Fatalf("option = %+v, want method=%s uri=%s requiresSMTP=%t oneClick=%t", got, method, uri, requiresSMTP, oneClick)
	}
}
