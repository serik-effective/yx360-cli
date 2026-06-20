package calendar

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
)

var ErrReauthRequired = errors.New("calendar: stored credential is missing, expired, or does not include calendar:all; run yx360 login --calendar")

type Service struct {
	cfg    config.Calendar
	cred   *auth.Credential
	client *http.Client
}

type Query struct {
	From time.Time
	To   time.Time
}

type MutateOptions struct {
	Event       Event
	TelemostURL string
}

func NewService(cfg config.Calendar, cred *auth.Credential) *Service {
	return &Service{cfg: cfg, cred: cred, client: ipv4Client()}
}

func (s *Service) List(ctx context.Context, q Query) ([]Event, error) {
	calendarURL, err := s.primaryCalendarURL(ctx)
	if err != nil {
		return nil, err
	}
	if q.From.IsZero() {
		q.From = time.Now().Add(-24 * time.Hour)
	}
	if q.To.IsZero() {
		q.To = q.From.Add(30 * 24 * time.Hour)
	}
	body := fmt.Sprintf(`<?xml version="1.0" encoding="utf-8" ?>
<C:calendar-query xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop><D:getetag/><C:calendar-data/></D:prop>
  <C:filter><C:comp-filter name="VCALENDAR"><C:comp-filter name="VEVENT"><C:time-range start="%s" end="%s"/></C:comp-filter></C:comp-filter></C:filter>
</C:calendar-query>`, q.From.UTC().Format("20060102T150405Z"), q.To.UTC().Format("20060102T150405Z"))
	resp, err := s.request(ctx, "REPORT", calendarURL, "1", "application/xml; charset=utf-8", strings.NewReader(body), "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("calendar: list failed: HTTP %d", resp.StatusCode)
	}
	events, err := decodeEvents(resp.Body)
	if err != nil {
		return nil, err
	}
	for i := range events {
		events[i] = events[i].occurrenceIn(q.From, q.To)
	}
	return events, nil
}

func (s *Service) Read(ctx context.Context, href string) (*Event, error) {
	event, err := s.getEvent(ctx, href)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *Service) Create(ctx context.Context, event Event) (*Event, error) {
	if event.UID == "" {
		event.UID = newUID()
	}
	calendarURL, err := s.primaryCalendarURL(ctx)
	if err != nil {
		return nil, err
	}
	href := eventHref(calendarURL, event.UID)
	resp, err := s.request(ctx, http.MethodPut, href, "", "text/calendar; charset=utf-8", strings.NewReader(buildICS(event)), "*")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("calendar: create failed: HTTP %d", resp.StatusCode)
	}
	return s.getEvent(ctx, href)
}

func (s *Service) Update(ctx context.Context, href string, patch Event) (*Event, error) {
	current, err := s.getEvent(ctx, href)
	if err != nil {
		return nil, err
	}
	updated := *current
	if patch.Title != "" {
		updated.Title = patch.Title
	}
	if patch.Description != "" {
		updated.Description = patch.Description
	}
	if patch.Location != "" {
		updated.Location = patch.Location
	}
	if patch.URL != "" {
		updated.URL = patch.URL
	}
	if !patch.StartsAt.IsZero() {
		updated.StartsAt = patch.StartsAt
	}
	if !patch.EndsAt.IsZero() {
		updated.EndsAt = patch.EndsAt
	}
	if patch.Attendees != nil {
		updated.Attendees = patch.Attendees
	}
	resp, err := s.request(ctx, http.MethodPut, absoluteURL(s.cfg.BaseURL, current.Href), "", "text/calendar; charset=utf-8", strings.NewReader(buildICS(updated)), current.ETag)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("calendar: update failed: HTTP %d", resp.StatusCode)
	}
	return s.getEvent(ctx, current.Href)
}

func (s *Service) Delete(ctx context.Context, href string) (*Event, error) {
	event, err := s.getEvent(ctx, href)
	if err != nil {
		return nil, err
	}
	resp, err := s.request(ctx, http.MethodDelete, absoluteURL(s.cfg.BaseURL, event.Href), "", "", nil, event.ETag)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar: delete failed: HTTP %d", resp.StatusCode)
	}
	return event, nil
}

func (s *Service) primaryCalendarURL(ctx context.Context) (string, error) {
	resp, err := s.propfind(ctx, strings.TrimRight(s.cfg.BaseURL, "/")+"/", "0", []string{"D:current-user-principal"})
	if err != nil {
		return "", err
	}
	principal := firstPropHref(resp, "current-user-principal")
	if principal == "" {
		return "", errors.New("calendar: CalDAV principal not found")
	}
	resp, err = s.propfind(ctx, absoluteURL(s.cfg.BaseURL, principal), "0", []string{"C:calendar-home-set"})
	if err != nil {
		return "", err
	}
	home := firstPropHref(resp, "calendar-home-set")
	if home == "" {
		return "", errors.New("calendar: CalDAV calendar-home-set not found")
	}
	resp, err = s.propfind(ctx, absoluteURL(s.cfg.BaseURL, home), "1", []string{"D:resourcetype", "D:displayname"})
	if err != nil {
		return "", err
	}
	for _, r := range resp.Responses {
		if r.isCalendar() {
			return absoluteURL(s.cfg.BaseURL, r.Href), nil
		}
	}
	return "", errors.New("calendar: no calendar collection found")
}

func (s *Service) propfind(ctx context.Context, endpoint, depth string, props []string) (*multistatus, error) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8" ?><D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav"><D:prop>`)
	for _, prop := range props {
		b.WriteString("<")
		b.WriteString(prop)
		b.WriteString("/>")
	}
	b.WriteString(`</D:prop></D:propfind>`)
	resp, err := s.request(ctx, "PROPFIND", endpoint, depth, "application/xml; charset=utf-8", strings.NewReader(b.String()), "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("calendar: propfind failed: HTTP %d", resp.StatusCode)
	}
	var ms multistatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return nil, err
	}
	return &ms, nil
}

func (s *Service) getEvent(ctx context.Context, href string) (*Event, error) {
	endpoint := absoluteURL(s.cfg.BaseURL, href)
	resp, err := s.request(ctx, http.MethodGet, endpoint, "", "", nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("calendar: read failed: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	event := parseEvent(href, resp.Header.Get("ETag"), string(body))
	return &event, nil
}

func (s *Service) request(ctx context.Context, method, endpoint, depth, contentType string, body io.Reader, ifMatch string) (*http.Response, error) {
	if s.cred == nil || !s.cred.Valid() || !s.cred.HasScopes(s.cfg.Scope) {
		return nil, ErrReauthRequired
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+s.cred.AccessToken)
	if depth != "" {
		req.Header.Set("Depth", depth)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if ifMatch != "" {
		if ifMatch == "*" {
			req.Header.Set("If-None-Match", "*")
		} else {
			req.Header.Set("If-Match", ifMatch)
		}
	}
	return s.client.Do(req)
}

type multistatus struct {
	Responses []response `xml:"response"`
}

type response struct {
	Href     string     `xml:"href"`
	Propstat []propstat `xml:"propstat"`
}

type propstat struct {
	Prop prop `xml:"prop"`
}

type prop struct {
	GetETag              string       `xml:"getetag"`
	CalendarData         string       `xml:"calendar-data"`
	DisplayName          string       `xml:"displayname"`
	CurrentUserPrincipal hrefHolder   `xml:"current-user-principal"`
	CalendarHomeSet      hrefHolder   `xml:"calendar-home-set"`
	PrincipalURL         hrefHolder   `xml:"principal-URL"`
	ResourceType         resourceType `xml:"resourcetype"`
}

type hrefHolder struct {
	Href string `xml:"href"`
}

type resourceType struct {
	Calendar *struct{} `xml:"calendar"`
}

func (r response) isCalendar() bool {
	for _, ps := range r.Propstat {
		if ps.Prop.ResourceType.Calendar != nil {
			return true
		}
	}
	return false
}

func firstPropHref(ms *multistatus, name string) string {
	for _, r := range ms.Responses {
		for _, ps := range r.Propstat {
			switch name {
			case "current-user-principal":
				if ps.Prop.CurrentUserPrincipal.Href != "" {
					return ps.Prop.CurrentUserPrincipal.Href
				}
			case "calendar-home-set":
				if ps.Prop.CalendarHomeSet.Href != "" {
					return ps.Prop.CalendarHomeSet.Href
				}
			}
		}
	}
	return ""
}

func decodeEvents(body io.Reader) ([]Event, error) {
	var ms multistatus
	if err := xml.NewDecoder(body).Decode(&ms); err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(ms.Responses))
	for _, r := range ms.Responses {
		var etag, data string
		for _, ps := range r.Propstat {
			if ps.Prop.GetETag != "" {
				etag = ps.Prop.GetETag
			}
			if ps.Prop.CalendarData != "" {
				data = ps.Prop.CalendarData
			}
		}
		if data != "" {
			events = append(events, parseEvent(r.Href, etag, data))
		}
	}
	return events, nil
}

func absoluteURL(base, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(href, "/")
	}
	ref, err := url.Parse(href)
	if err != nil {
		return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(href, "/")
	}
	return u.ResolveReference(ref).String()
}

func ipv4Client() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}).DialContext(ctx, "tcp4", address)
	}
	return &http.Client{Timeout: 30 * time.Second, Transport: transport}
}

func cloneBody(data string) io.Reader {
	return bytes.NewBufferString(data)
}
