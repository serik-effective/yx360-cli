package calendar

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RoomMapping struct {
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	URI   string `json:"uri,omitempty"`
	Kind  string `json:"kind,omitempty"`
}

type RoomRegistry struct {
	path string
}

func NewRoomRegistry() (*RoomRegistry, error) {
	dir := strings.TrimSpace(os.Getenv("YX360_CONFIG_HOME"))
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return nil, err
		}
	}
	return NewRoomRegistryAt(filepath.Join(dir, "yx360", "calendar-rooms.json")), nil
}

func NewRoomRegistryAt(path string) *RoomRegistry {
	return &RoomRegistry{path: path}
}

func (r *RoomRegistry) List() ([]RoomMapping, error) {
	rooms, err := r.load()
	if err != nil {
		return nil, err
	}
	return sortedRoomMappings(rooms), nil
}

func (r *RoomRegistry) Add(room RoomMapping) error {
	if strings.TrimSpace(room.Name) == "" {
		return errors.New("calendar: room name is required")
	}
	room.Name = strings.TrimSpace(room.Name)
	room.Email = strings.TrimSpace(strings.TrimPrefix(room.Email, "mailto:"))
	room.URI = strings.TrimSpace(room.URI)
	room.Kind = strings.ToUpper(strings.TrimSpace(room.Kind))
	if room.Kind == "" {
		room.Kind = "ROOM"
	}
	if room.URI == "" && room.Email != "" {
		room.URI = "mailto:" + room.Email
	}
	if room.Email == "" && strings.HasPrefix(strings.ToLower(room.URI), "mailto:") {
		room.Email = strings.TrimPrefix(room.URI, "mailto:")
	}
	if room.Email == "" && room.URI == "" {
		return errors.New("calendar: room email or URI is required")
	}

	rooms, err := r.load()
	if err != nil {
		return err
	}
	rooms[roomKey(room.Name)] = room
	return r.save(rooms)
}

func (r *RoomRegistry) Resolve(alias string) (Room, error) {
	rooms, err := r.load()
	if err != nil {
		return Room{}, err
	}
	mapping, ok := rooms[roomKey(alias)]
	if !ok {
		return Room{}, fmt.Errorf("calendar: unknown room %q; run `yx360 calendar rooms list` or `yx360 calendar rooms add %s <email>`", alias, alias)
	}
	return Room{
		Alias:  alias,
		Name:   mapping.Name,
		Email:  mapping.Email,
		URI:    mapping.URI,
		Status: "ACCEPTED",
		Role:   "REQ-PARTICIPANT",
		Kind:   firstNonEmpty(mapping.Kind, "ROOM"),
	}, nil
}

func (r *RoomRegistry) MergeDiscovered(events []Event) ([]RoomMapping, error) {
	rooms, err := r.load()
	if err != nil {
		return nil, err
	}
	for _, room := range DiscoverRooms(events) {
		rooms[roomKey(room.Name)] = room
	}
	if err := r.save(rooms); err != nil {
		return nil, err
	}
	return sortedRoomMappings(rooms), nil
}

func DiscoverRooms(events []Event) []RoomMapping {
	rooms := make(map[string]RoomMapping)
	for _, event := range events {
		for _, room := range append(event.Rooms, event.Resources...) {
			name := firstNonEmpty(room.Name, room.Alias, room.Email)
			if name == "" {
				continue
			}
			mapping := RoomMapping{
				Name:  name,
				Email: strings.TrimPrefix(room.Email, "mailto:"),
				URI:   room.URI,
				Kind:  firstNonEmpty(room.Kind, "ROOM"),
			}
			if mapping.URI == "" && mapping.Email != "" {
				mapping.URI = "mailto:" + mapping.Email
			}
			rooms[roomKey(name)] = mapping
		}
	}
	return sortedRoomMappings(rooms)
}

func (r *RoomRegistry) load() (map[string]RoomMapping, error) {
	blob, err := os.ReadFile(r.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]RoomMapping{}, nil
		}
		return nil, err
	}
	var stored struct {
		Rooms []RoomMapping `json:"rooms"`
	}
	if err := json.Unmarshal(blob, &stored); err != nil {
		return nil, err
	}
	rooms := make(map[string]RoomMapping, len(stored.Rooms))
	for _, room := range stored.Rooms {
		if room.Name != "" {
			rooms[roomKey(room.Name)] = room
		}
	}
	return rooms, nil
}

func (r *RoomRegistry) save(rooms map[string]RoomMapping) error {
	if err := os.MkdirAll(filepath.Dir(r.path), 0o700); err != nil {
		return err
	}
	stored := struct {
		Rooms []RoomMapping `json:"rooms"`
	}{Rooms: sortedRoomMappings(rooms)}
	blob, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	blob = append(blob, '\n')
	return os.WriteFile(r.path, blob, 0o600)
}

func sortedRoomMappings(rooms map[string]RoomMapping) []RoomMapping {
	list := make([]RoomMapping, 0, len(rooms))
	for _, room := range rooms {
		list = append(list, room)
	}
	sort.Slice(list, func(i, j int) bool {
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	return list
}

func roomKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
