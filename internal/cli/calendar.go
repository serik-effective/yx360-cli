package cli

import (
	"bufio"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/calendar"
	"github.com/effective-dev-os/yx360-cli/internal/config"
	"github.com/effective-dev-os/yx360-cli/internal/telemost"
	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

func newCalendarCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendar",
		Short: "Read and manage Yandex Calendar events via CalDAV",
	}
	cmd.AddCommand(newCalendarListCmd())
	cmd.AddCommand(newCalendarReadCmd())
	cmd.AddCommand(newCalendarCreateCmd())
	cmd.AddCommand(newCalendarUpdateCmd())
	cmd.AddCommand(newCalendarDeleteCmd())
	return cmd
}

func newCalendarListCmd() *cobra.Command {
	var from, to string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List calendar events",
		RunE: func(cmd *cobra.Command, _ []string) error {
			q, err := parseCalendarQuery(from, to)
			if err != nil {
				return err
			}
			svc, err := calendarService(cmd)
			if err != nil {
				return err
			}
			events, err := svc.List(cmd.Context(), q)
			if err != nil {
				return friendlyCalendarError(err)
			}
			return emit(cmd, humanCalendarEvents(events), events)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "start time (RFC3339 or YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "end time (RFC3339 or YYYY-MM-DD)")
	return cmd
}

func newCalendarReadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read <event-href>",
		Short: "Read one calendar event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := calendarService(cmd)
			if err != nil {
				return err
			}
			event, err := svc.Read(cmd.Context(), args[0])
			if err != nil {
				return friendlyCalendarError(err)
			}
			return emit(cmd, humanCalendarEvent(*event), event)
		},
	}
	return cmd
}

func newCalendarCreateCmd() *cobra.Command {
	var (
		event        calendar.Event
		startValue   string
		endValue     string
		yes          bool
		withTelemost bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a non-recurring calendar event",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := fillEventTimes(&event, startValue, endValue, true); err != nil {
				return err
			}
			if event.Title == "" {
				return errors.New("calendar: --title is required")
			}
			if withTelemost {
				if event.Description != "" {
					event.Description += "\n\n"
				}
				event.Description += "Telemost link will be created during apply."
			}
			if !yes {
				if err := confirmCalendarMutation(cmd, "Create calendar event?", event, withTelemost); err != nil {
					return err
				}
			}
			if withTelemost {
				tm, err := telemostService(cmd)
				if err != nil {
					return err
				}
				conference, err := tm.Create(cmd.Context(), telemost.CreateOptions{})
				if err != nil {
					return friendlyTelemostError(err)
				}
				linkText := "Telemost: " + conference.JoinURL
				event.Location = conference.JoinURL
				event.URL = conference.JoinURL
				event.Description = strings.TrimSpace(strings.ReplaceAll(event.Description, "Telemost link will be created during apply.", linkText))
			}
			svc, err := calendarService(cmd)
			if err != nil {
				return err
			}
			created, err := svc.Create(cmd.Context(), event)
			if err != nil {
				return friendlyCalendarError(err)
			}
			return emit(cmd, "Created calendar event "+created.Href, created)
		},
	}
	addCalendarEventFlags(cmd, &event, &startValue, &endValue)
	cmd.Flags().BoolVar(&withTelemost, "telemost", false, "create and attach a Telemost link")
	cmd.Flags().BoolVar(&yes, "yes", false, "create without interactive confirmation")
	return cmd
}

func newCalendarUpdateCmd() *cobra.Command {
	var (
		patch      calendar.Event
		startValue string
		endValue   string
		yes        bool
	)
	cmd := &cobra.Command{
		Use:   "update <event-href>",
		Short: "Update a non-recurring calendar event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := fillEventTimes(&patch, startValue, endValue, false); err != nil {
				return err
			}
			if !yes {
				if err := confirmCalendarMutation(cmd, "Update calendar event?", patch, false); err != nil {
					return err
				}
			}
			svc, err := calendarService(cmd)
			if err != nil {
				return err
			}
			updated, err := svc.Update(cmd.Context(), args[0], patch)
			if err != nil {
				return friendlyCalendarError(err)
			}
			return emit(cmd, "Updated calendar event "+updated.Href, updated)
		},
	}
	addCalendarEventFlags(cmd, &patch, &startValue, &endValue)
	cmd.Flags().BoolVar(&yes, "yes", false, "update without interactive confirmation")
	return cmd
}

func newCalendarDeleteCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <event-href>",
		Short: "Delete a calendar event",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := calendarService(cmd)
			if err != nil {
				return err
			}
			event, err := svc.Read(cmd.Context(), args[0])
			if err != nil {
				return friendlyCalendarError(err)
			}
			if !yes {
				if err := confirmCalendarMutation(cmd, "Delete calendar event?", *event, false); err != nil {
					return err
				}
			}
			deleted, err := svc.Delete(cmd.Context(), args[0])
			if err != nil {
				return friendlyCalendarError(err)
			}
			return emit(cmd, "Deleted calendar event "+deleted.Href, deleted)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "delete without interactive confirmation")
	return cmd
}

func newTelemostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telemost",
		Short: "Create Yandex Telemost links",
	}
	cmd.AddCommand(newTelemostCreateCmd())
	return cmd
}

func newTelemostCreateCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Telemost conference",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !yes {
				cmd.Println("Telemost create preview:")
				cmd.Println("  Waiting room: PUBLIC")
				if err := confirmPrompt(cmd, "Create Telemost link?"); err != nil {
					return err
				}
			}
			svc, err := telemostService(cmd)
			if err != nil {
				return err
			}
			conference, err := svc.Create(cmd.Context(), telemost.CreateOptions{})
			if err != nil {
				return friendlyTelemostError(err)
			}
			return emit(cmd, "Created Telemost link "+conference.JoinURL, conference)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "create without interactive confirmation")
	return cmd
}

func addCalendarEventFlags(cmd *cobra.Command, event *calendar.Event, startValue, endValue *string) {
	cmd.Flags().StringVar(&event.Title, "title", "", "event title")
	cmd.Flags().StringVar(startValue, "starts-at", "", "start time (RFC3339)")
	cmd.Flags().StringVar(endValue, "ends-at", "", "end time (RFC3339)")
	cmd.Flags().StringVar(&event.Description, "description", "", "event description")
	cmd.Flags().StringVar(&event.Location, "location", "", "event location")
	cmd.Flags().StringArrayVar(&event.Attendees, "attendee", nil, "attendee email; repeatable")
}

func fillEventTimes(event *calendar.Event, startValue, endValue string, required bool) error {
	if startValue == "" || endValue == "" {
		if required {
			return errors.New("calendar: --starts-at and --ends-at are required")
		}
		return nil
	}
	start, err := parseTimeValue(startValue)
	if err != nil {
		return fmt.Errorf("calendar: --starts-at must be RFC3339 or YYYY-MM-DD: %w", err)
	}
	end, err := parseTimeValue(endValue)
	if err != nil {
		return fmt.Errorf("calendar: --ends-at must be RFC3339 or YYYY-MM-DD: %w", err)
	}
	if !end.After(start) {
		return errors.New("calendar: --ends-at must be after --starts-at")
	}
	event.StartsAt = start
	event.EndsAt = end
	return nil
}

func parseCalendarQuery(from, to string) (calendar.Query, error) {
	var q calendar.Query
	var err error
	if from != "" {
		q.From, err = parseTimeValue(from)
		if err != nil {
			return q, fmt.Errorf("calendar: --from must be RFC3339 or YYYY-MM-DD: %w", err)
		}
	}
	if to != "" {
		q.To, err = parseTimeValue(to)
		if err != nil {
			return q, fmt.Errorf("calendar: --to must be RFC3339 or YYYY-MM-DD: %w", err)
		}
	}
	return q, nil
}

func parseTimeValue(value string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", value)
}

func calendarService(cmd *cobra.Command) (*calendar.Service, error) {
	store, err := selectStoreFor(calendarTelemostProfile)
	if err != nil {
		return nil, err
	}
	cred, err := store.Load(cmd.Context())
	if err != nil {
		if errors.Is(err, tokenstore.ErrNoCredential) {
			return nil, calendar.ErrReauthRequired
		}
		return nil, err
	}
	return calendar.NewService(config.DefaultCalendar(), cred), nil
}

func telemostService(cmd *cobra.Command) (*telemost.Service, error) {
	store, err := selectStoreFor(calendarTelemostProfile)
	if err != nil {
		return nil, err
	}
	cred, err := store.Load(cmd.Context())
	if err != nil {
		if errors.Is(err, tokenstore.ErrNoCredential) {
			return nil, telemost.ErrReauthRequired
		}
		return nil, err
	}
	return telemost.NewService(config.DefaultTelemost(), cred), nil
}

func friendlyCalendarError(err error) error { return err }

func friendlyTelemostError(err error) error { return err }

func humanCalendarEvents(events []calendar.Event) string {
	if len(events) == 0 {
		return "No calendar events"
	}
	var b strings.Builder
	for _, event := range events {
		b.WriteString(humanCalendarEvent(event))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func humanCalendarEvent(event calendar.Event) string {
	title := event.Title
	if title == "" {
		title = "(no title)"
	}
	return fmt.Sprintf("%s %s %s-%s", event.Href, title, event.StartsAt.Format(time.RFC3339), event.EndsAt.Format(time.RFC3339))
}

func confirmCalendarMutation(cmd *cobra.Command, title string, event calendar.Event, telemost bool) error {
	cmd.Println(title)
	if event.Href != "" {
		cmd.Println("  Href: " + event.Href)
	}
	if event.Title != "" {
		cmd.Println("  Title: " + event.Title)
	}
	if !event.StartsAt.IsZero() {
		cmd.Println("  Starts: " + event.StartsAt.Format(time.RFC3339))
	}
	if !event.EndsAt.IsZero() {
		cmd.Println("  Ends: " + event.EndsAt.Format(time.RFC3339))
	}
	if len(event.Attendees) > 0 {
		cmd.Println("  Attendees: " + strings.Join(event.Attendees, ","))
	}
	if telemost {
		cmd.Println("  Telemost: create link")
	}
	return confirmPrompt(cmd, "Continue?")
}

func confirmPrompt(cmd *cobra.Command, prompt string) error {
	cmd.Print(prompt + " [y/N] ")
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil {
		return err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	if answer != "y" && answer != "yes" {
		return errors.New("calendar: action cancelled")
	}
	return nil
}
