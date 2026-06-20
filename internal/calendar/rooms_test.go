package calendar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoomRegistryAddResolveAndOverride(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "rooms.json")
	registry := NewRoomRegistryAt(path)
	if err := registry.Add(RoomMapping{Name: "Sun", Email: "sun@example.com"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := registry.Add(RoomMapping{Name: "sun", Email: "sun-room@example.com"}); err != nil {
		t.Fatalf("Add override: %v", err)
	}

	room, err := registry.Resolve("SUN")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if room.Name != "sun" || room.Email != "sun-room@example.com" || room.URI != "mailto:sun-room@example.com" {
		t.Fatalf("room = %+v", room)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
}

func TestRoomRegistryMergeDiscovered(t *testing.T) {
	t.Parallel()

	registry := NewRoomRegistryAt(filepath.Join(t.TempDir(), "rooms.json"))
	rooms, err := registry.MergeDiscovered([]Event{{
		Rooms: []Room{{
			Name:  "Sun",
			Email: "sun@example.com",
			URI:   "mailto:sun@example.com",
		}},
		Resources: []Room{{
			Name:  "Projector",
			Email: "projector@example.com",
		}},
	}})
	if err != nil {
		t.Fatalf("MergeDiscovered: %v", err)
	}
	if len(rooms) != 2 {
		t.Fatalf("rooms = %+v", rooms)
	}
	if _, err := registry.Resolve("projector"); err != nil {
		t.Fatalf("Resolve discovered resource: %v", err)
	}
}
