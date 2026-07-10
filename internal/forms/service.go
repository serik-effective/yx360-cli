package forms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/effective-dev-os/yx360-cli/internal/auth"
	"github.com/effective-dev-os/yx360-cli/internal/config"
	"github.com/effective-dev-os/yx360-cli/internal/netutil"
)

var ErrReauthRequired = errors.New("forms: stored credential is missing, expired, or does not include the required forms scope; run yx360 login --forms")

// Endpoint shapes per Yandex Forms docs, live-unverified (C-1): exact paths and
// response fields may not match what the API returns until checked against a
// real org.
const (
	answersPathFmt   = "/v1/surveys/%s/answers"
	surveysPath      = "/v1/surveys/"
	questionsPathFmt = "/v1/surveys/%s/questions/"
	publishPathFmt   = "/v1/surveys/%s/publish"
	unpublishPathFmt = "/v1/surveys/%s/unpublish"
)

type Service struct {
	cfg    config.Forms
	cred   *auth.Credential
	client *http.Client
}

// Answer is a tolerant decode target: known fields are typed, Raw keeps
// whatever else the API sends so unexpected shapes don't fail decoding (C-1).
type Answer struct {
	ID           string         `json:"id"`
	RespondentID string         `json:"respondent_id,omitempty"`
	SubmittedAt  string         `json:"submitted_at,omitempty"`
	Raw          map[string]any `json:"-"`
}

type ListResponsesResult struct {
	Answers       []Answer `json:"answers"`
	NextPageToken string   `json:"next_page_token,omitempty"`
}

// MarshalJSON emits the full raw answer payload from the API; the documented
// field names vary, so typed fields alone would drop the actual response data.
func (a Answer) MarshalJSON() ([]byte, error) {
	if len(a.Raw) > 0 {
		return json.Marshal(a.Raw)
	}
	type alias Answer
	return json.Marshal(alias(a))
}

type Survey struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

func NewService(cfg config.Forms, cred *auth.Credential) *Service {
	return &Service{cfg: cfg, cred: cred, client: netutil.IPv4Client()}
}

func (s *Service) ListResponses(ctx context.Context, surveyID string, pageSize int, pageToken string) (*ListResponsesResult, error) {
	if surveyID == "" {
		return nil, errors.New("forms: survey id is required")
	}
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + fmt.Sprintf(answersPathFmt, surveyID)
	query := make([]string, 0, 2)
	if pageSize > 0 {
		query = append(query, "page_size="+strconv.Itoa(pageSize))
	}
	if pageToken != "" {
		query = append(query, "page_token="+pageToken)
	}
	if len(query) > 0 {
		endpoint += "?" + strings.Join(query, "&")
	}
	resp, err := s.do(ctx, http.MethodGet, endpoint, s.cfg.ReadScope, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, httpError(resp, "list responses")
	}
	var raw struct {
		Answers []map[string]any `json:"answers"`
		Next    string           `json:"next"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	result := &ListResponsesResult{NextPageToken: raw.Next, Answers: make([]Answer, 0, len(raw.Answers))}
	for _, item := range raw.Answers {
		result.Answers = append(result.Answers, decodeAnswer(item))
	}
	return result, nil
}

func (s *Service) CreateSurvey(ctx context.Context, title string) (*Survey, error) {
	if title == "" {
		return nil, errors.New("forms: title is required")
	}
	body, err := json.Marshal(struct {
		Name string `json:"name"`
	}{Name: title})
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + surveysPath
	resp, err := s.do(ctx, http.MethodPost, endpoint, s.cfg.WriteScope, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, httpError(resp, "create survey")
	}
	var survey Survey
	if err := json.NewDecoder(resp.Body).Decode(&survey); err != nil {
		return nil, err
	}
	if survey.ID == "" {
		return nil, errors.New("forms: create response missing id")
	}
	return &survey, nil
}

// AddQuestion adds one question of the given type (rating|text|integer).
// The rating (enum/radio 1..scale) shape is live-verified; the string/integer
// shapes are Yandex Forms doc-derived.
func (s *Service) AddQuestion(ctx context.Context, surveyID, qType, label string, scale int) (map[string]any, error) {
	if surveyID == "" {
		return nil, errors.New("forms: survey id is required")
	}
	if label == "" {
		return nil, errors.New("forms: question label is required")
	}
	var question map[string]any
	switch qType {
	case "rating":
		if scale < 2 {
			return nil, errors.New("forms: rating scale must be at least 2")
		}
		items := make([]map[string]string, 0, scale)
		for i := 1; i <= scale; i++ {
			items = append(items, map[string]string{"label": strconv.Itoa(i)})
		}
		question = map[string]any{"type": "enum", "label": label, "widget": "radio", "items": items}
	case "text":
		question = map[string]any{"type": "string", "label": label}
	case "integer":
		question = map[string]any{"type": "integer", "label": label}
	default:
		return nil, fmt.Errorf("forms: unknown question type %q (want rating|text|integer)", qType)
	}
	body, err := json.Marshal(question)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + fmt.Sprintf(questionsPathFmt, surveyID)
	resp, err := s.do(ctx, http.MethodPost, endpoint, s.cfg.WriteScope, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, httpError(resp, "add question")
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) Publish(ctx context.Context, surveyID string) (map[string]any, error) {
	return s.setPublishState(ctx, surveyID, publishPathFmt)
}

func (s *Service) Unpublish(ctx context.Context, surveyID string) (map[string]any, error) {
	return s.setPublishState(ctx, surveyID, unpublishPathFmt)
}

func (s *Service) setPublishState(ctx context.Context, surveyID, pathFmt string) (map[string]any, error) {
	if surveyID == "" {
		return nil, errors.New("forms: survey id is required")
	}
	endpoint := strings.TrimRight(s.cfg.BaseURL, "/") + fmt.Sprintf(pathFmt, surveyID)
	resp, err := s.do(ctx, http.MethodPost, endpoint, s.cfg.WriteScope, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, httpError(resp, "request")
	}
	var out map[string]any
	if resp.StatusCode != http.StatusNoContent {
		_ = json.NewDecoder(resp.Body).Decode(&out)
	}
	return out, nil
}

func isNumericOrgID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func httpError(resp *http.Response, action string) error {
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	msg := strings.TrimSpace(string(snippet))
	if msg == "" {
		return fmt.Errorf("forms: %s failed: HTTP %d", action, resp.StatusCode)
	}
	return fmt.Errorf("forms: %s failed: HTTP %d: %s", action, resp.StatusCode, msg)
}

func (s *Service) do(ctx context.Context, method, endpoint, scope string, body io.Reader) (*http.Response, error) {
	if s.cred == nil || !s.cred.Valid() || !s.cred.HasScopes(scope) {
		return nil, ErrReauthRequired
	}
	if s.cfg.OrgID == "" {
		return nil, errors.New("forms: org id is required; set YX360_FORMS_ORG_ID")
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "OAuth "+s.cred.AccessToken)
	// Numeric ids are Yandex 360 organizations (X-Org-Id); non-numeric ids are
	// Yandex Cloud organizations (X-Cloud-Org-Id). The Forms API rejects the
	// wrong header/value pairing with "organization required".
	if isNumericOrgID(s.cfg.OrgID) {
		req.Header.Set("X-Org-Id", s.cfg.OrgID)
	} else {
		req.Header.Set("X-Cloud-Org-Id", s.cfg.OrgID)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, ErrReauthRequired
	}
	return resp, nil
}

func decodeAnswer(item map[string]any) Answer {
	answer := Answer{Raw: item}
	if v, ok := item["id"].(string); ok {
		answer.ID = v
	}
	if v, ok := item["respondent_id"].(string); ok {
		answer.RespondentID = v
	}
	if v, ok := item["submitted_at"].(string); ok {
		answer.SubmittedAt = v
	}
	return answer
}

