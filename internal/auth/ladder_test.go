package auth

import (
	"context"
	"errors"
	"testing"
)

type fakeProvider struct {
	cred   *Credential
	err    error
	called bool
}

func (f *fakeProvider) Authenticate(_ context.Context, _ AuthOptions) (*Credential, error) {
	f.called = true
	return f.cred, f.err
}

func TestLadderAdvancesOnRungUnavailable(t *testing.T) {
	loopback := &fakeProvider{err: errRungUnavailable}
	device := &fakeProvider{cred: &Credential{AccessToken: "tok", ObtainedVia: GrantDevice}}
	ladder := NewLadder(loopback, device)

	cred, err := ladder.Authenticate(context.Background(), AuthOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !loopback.called || !device.called {
		t.Fatalf("expected both rungs tried, loopback=%v device=%v", loopback.called, device.called)
	}
	if cred.ObtainedVia != GrantDevice {
		t.Fatalf("expected device credential, got %s", cred.ObtainedVia)
	}
}

func TestLadderAbortsOnRealError(t *testing.T) {
	realErr := errors.New("invalid_grant: consent denied")
	loopback := &fakeProvider{err: realErr}
	device := &fakeProvider{cred: &Credential{AccessToken: "tok"}}
	ladder := NewLadder(loopback, device)

	_, err := ladder.Authenticate(context.Background(), AuthOptions{})
	if !errors.Is(err, realErr) {
		t.Fatalf("expected real OAuth error to abort, got %v", err)
	}
	if device.called {
		t.Fatal("device rung must NOT run after a real OAuth error")
	}
}

func TestLadderPreferDeviceSkipsLoopback(t *testing.T) {
	loopback := &fakeProvider{err: errRungUnavailable}
	device := &fakeProvider{cred: &Credential{AccessToken: "tok", ObtainedVia: GrantDevice}}
	ladder := NewLadder(loopback, device)

	cred, err := ladder.Authenticate(context.Background(), AuthOptions{PreferDevice: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loopback.called {
		t.Fatal("loopback must be skipped when PreferDevice is set")
	}
	if cred.ObtainedVia != GrantDevice {
		t.Fatalf("expected device credential, got %s", cred.ObtainedVia)
	}
}
