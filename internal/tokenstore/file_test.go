package tokenstore

import (
	"strings"
	"testing"
)

func TestFileStoreProfilePath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	defaultStore, err := NewFileStore()
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	profileStore, err := NewFileStoreFor("calendar-telemost")
	if err != nil {
		t.Fatalf("NewFileStoreFor: %v", err)
	}
	if defaultStore.path == profileStore.path {
		t.Fatalf("profile store reused default path %q", defaultStore.path)
	}
	if !strings.HasSuffix(profileStore.path, "credential.calendar-telemost.json") {
		t.Fatalf("profile path = %q", profileStore.path)
	}
}
