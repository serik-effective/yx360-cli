package yx360mcp

import (
	"context"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/effective-dev-os/yx360-cli/internal/calendar"
	"github.com/effective-dev-os/yx360-cli/internal/telemost"
)

type calendarListInput struct {
	From string `json:"from,omitempty" jsonschema:"start time filter (RFC3339 or YYYY-MM-DD)"`
	To   string `json:"to,omitempty"   jsonschema:"end time filter (RFC3339 or YYYY-MM-DD)"`
}

type calendarCreateInput struct {
	Title        string   `json:"title"                   jsonschema:"event title"`
	StartsAt     string   `json:"starts_at"               jsonschema:"start time (RFC3339)"`
	EndsAt       string   `json:"ends_at"                 jsonschema:"end time (RFC3339)"`
	Description  string   `json:"description,omitempty"   jsonschema:"event description"`
	Location     string   `json:"location,omitempty"      jsonschema:"event location"`
	Attendees    []string `json:"attendees,omitempty"     jsonschema:"attendee email addresses"`
	WithTelemost bool     `json:"with_telemost,omitempty" jsonschema:"create and attach a Telemost conference link"`
	Confirmed    bool     `json:"confirmed"               jsonschema:"set true to execute; omit for dry-run preview"`
}

type calendarUpdateInput struct {
	Href        string   `json:"href"                  jsonschema:"event href from calendar_list"`
	Title       string   `json:"title,omitempty"       jsonschema:"new event title"`
	StartsAt    string   `json:"starts_at,omitempty"   jsonschema:"new start time (RFC3339)"`
	EndsAt      string   `json:"ends_at,omitempty"     jsonschema:"new end time (RFC3339)"`
	Description string   `json:"description,omitempty" jsonschema:"new event description"`
	Location    string   `json:"location,omitempty"    jsonschema:"new event location"`
	Attendees   []string `json:"attendees,omitempty"   jsonschema:"replacement attendee list"`
	Confirmed   bool     `json:"confirmed"             jsonschema:"set true to execute; omit for dry-run preview"`
}

type calendarDeleteInput struct {
	Href      string `json:"href"      jsonschema:"event href from calendar_list"`
	Confirmed bool   `json:"confirmed" jsonschema:"set true to execute; omit for dry-run preview"`
}

type telemostCreateInput struct {
	Confirmed bool `json:"confirmed" jsonschema:"set true to execute; omit for dry-run preview"`
}

func registerCalendarTools(
	srv *sdkmcp.Server,
	calSvcFn func(context.Context) (*calendar.Service, error),
	tmSvcFn func(context.Context) (*telemost.Service, error),
) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "calendar_list",
		Description: "List calendar events within an optional time window.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in calendarListInput) (*sdkmcp.CallToolResult, any, error) {
		q, err := buildCalQuery(in.From, in.To)
		if err != nil {
			return nil, nil, err
		}
		svc, err := calSvcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		events, err := svc.List(ctx, q)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(events)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "calendar_create",
		Description: "Create a calendar event. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in calendarCreateInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult(fmt.Sprintf("would create event %q at %s", in.Title, in.StartsAt))
		}
		event, err := buildCalEvent(in.Title, in.StartsAt, in.EndsAt, in.Description, in.Location, in.Attendees)
		if err != nil {
			return nil, nil, err
		}
		if in.WithTelemost && tmSvcFn != nil {
			tm, err := tmSvcFn(ctx)
			if err != nil {
				return nil, nil, toolErr(err)
			}
			conf, err := tm.Create(ctx, telemost.CreateOptions{})
			if err != nil {
				return nil, nil, toolErr(err)
			}
			event.URL = conf.JoinURL
			if event.Location == "" {
				event.Location = conf.JoinURL
			}
			if event.Description != "" {
				event.Description += "\n\n"
			}
			event.Description += "Telemost: " + conf.JoinURL
		}
		svc, err := calSvcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		created, err := svc.Create(ctx, event)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(created)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "calendar_update",
		Description: "Update a calendar event. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in calendarUpdateInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult("would update event " + in.Href)
		}
		patch, err := buildCalEvent(in.Title, in.StartsAt, in.EndsAt, in.Description, in.Location, in.Attendees)
		if err != nil {
			return nil, nil, err
		}
		svc, err := calSvcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		updated, err := svc.Update(ctx, in.Href, patch)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(updated)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "calendar_delete",
		Description: "Delete a calendar event. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in calendarDeleteInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult("would delete event " + in.Href)
		}
		svc, err := calSvcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		deleted, err := svc.Delete(ctx, in.Href)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(deleted)
	})

	if tmSvcFn != nil {
		sdkmcp.AddTool(srv, &sdkmcp.Tool{
			Name:        "telemost_create",
			Description: "Create a Yandex Telemost conference link. Pass confirmed=true to execute.",
		}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in telemostCreateInput) (*sdkmcp.CallToolResult, any, error) {
			if !in.Confirmed {
				return dryRunResult("would create Telemost conference")
			}
			svc, err := tmSvcFn(ctx)
			if err != nil {
				return nil, nil, toolErr(err)
			}
			conf, err := svc.Create(ctx, telemost.CreateOptions{})
			if err != nil {
				return nil, nil, toolErr(err)
			}
			return textResult(conf)
		})
	}
}

func buildCalQuery(from, to string) (calendar.Query, error) {
	var q calendar.Query
	if from != "" {
		t, err := parseCalTime(from)
		if err != nil {
			return q, fmt.Errorf("calendar: invalid from time: %w", err)
		}
		q.From = t
	}
	if to != "" {
		t, err := parseCalTime(to)
		if err != nil {
			return q, fmt.Errorf("calendar: invalid to time: %w", err)
		}
		q.To = t
	}
	return q, nil
}

func buildCalEvent(title, startsAt, endsAt, description, location string, attendees []string) (calendar.Event, error) {
	event := calendar.Event{
		Title:       title,
		Description: description,
		Location:    location,
		Attendees:   attendees,
	}
	if startsAt != "" {
		t, err := parseCalTime(startsAt)
		if err != nil {
			return event, fmt.Errorf("calendar: invalid starts_at: %w", err)
		}
		event.StartsAt = t
	}
	if endsAt != "" {
		t, err := parseCalTime(endsAt)
		if err != nil {
			return event, fmt.Errorf("calendar: invalid ends_at: %w", err)
		}
		event.EndsAt = t
	}
	return event, nil
}

func parseCalTime(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", value)
}
