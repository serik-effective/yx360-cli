package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/effective-dev-os/yx360-cli/internal/config"
	"github.com/effective-dev-os/yx360-cli/internal/forms"
	"github.com/effective-dev-os/yx360-cli/internal/tokenstore"
)

func newFormsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forms",
		Short: "Read responses and manage Yandex Forms surveys",
	}
	cmd.AddCommand(newFormsResponsesCmd())
	cmd.AddCommand(newFormsCreateCmd())
	cmd.AddCommand(newFormsQuestionsCmd())
	cmd.AddCommand(newFormsPublishCmd())
	cmd.AddCommand(newFormsUnpublishCmd())
	return cmd
}

func newFormsQuestionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "questions",
		Short: "Manage survey questions",
	}
	cmd.AddCommand(newFormsQuestionsAddCmd())
	return cmd
}

func newFormsQuestionsAddCmd() *cobra.Command {
	var (
		label  string
		qType  string
		rating int
		yes    bool
	)
	cmd := &cobra.Command{
		Use:   "add <survey-id>",
		Short: "Add a question (rating|text|integer) to a survey",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if label == "" {
				return errors.New("forms: --label is required")
			}
			if !yes {
				cmd.Println("Forms add-question preview:")
				if qType == "rating" {
					cmd.Printf("  Survey: %s\n  Question: %s (rating 1..%d)\n", args[0], label, rating)
				} else {
					cmd.Printf("  Survey: %s\n  Question: %s (%s)\n", args[0], label, qType)
				}
				if err := confirmPrompt(cmd, "Add question?"); err != nil {
					return err
				}
			}
			svc, err := formsService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.AddQuestion(cmd.Context(), args[0], qType, label, rating)
			if err != nil {
				return friendlyFormsError(err)
			}
			return emit(cmd, "Added question to survey "+args[0], result)
		},
	}
	cmd.Flags().StringVar(&label, "label", "", "question text")
	cmd.Flags().StringVar(&qType, "type", "rating", "question type: rating|text|integer")
	cmd.Flags().IntVar(&rating, "rating", 5, "rating scale size (1..N), for --type rating")
	cmd.Flags().BoolVar(&yes, "yes", false, "add without interactive confirmation")
	return cmd
}

func newFormsResponsesCmd() *cobra.Command {
	var (
		pageSize  int
		pageToken string
	)
	cmd := &cobra.Command{
		Use:   "responses <survey-id>",
		Short: "List responses for a survey",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := formsService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.ListResponses(cmd.Context(), args[0], pageSize, pageToken)
			if err != nil {
				return friendlyFormsError(err)
			}
			return emit(cmd, humanFormsResponses(result), result)
		},
	}
	cmd.Flags().IntVar(&pageSize, "page-size", 0, "maximum responses to return per page")
	cmd.Flags().StringVar(&pageToken, "page-token", "", "pagination cursor from a previous response")
	return cmd
}

func newFormsCreateCmd() *cobra.Command {
	var (
		title string
		yes   bool
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new survey",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if title == "" {
				return errors.New("forms: --title is required")
			}
			if !yes {
				if err := confirmFormsCreate(cmd, title); err != nil {
					return err
				}
			}
			svc, err := formsService(cmd)
			if err != nil {
				return err
			}
			survey, err := svc.CreateSurvey(cmd.Context(), title)
			if err != nil {
				return friendlyFormsError(err)
			}
			human := fmt.Sprintf("Created survey %s\n  Public: %s\n  Answers: %s", survey.ID, formsPublicURL(survey.ID), formsAnswersURL(survey.ID))
			return emit(cmd, human, surveyPayload{ID: survey.ID, Title: survey.Title, PublicURL: formsPublicURL(survey.ID), AnswersURL: formsAnswersURL(survey.ID)})
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "survey title")
	cmd.Flags().BoolVar(&yes, "yes", false, "create without interactive confirmation")
	return cmd
}

func newFormsPublishCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "publish <survey-id>",
		Short: "Publish a survey, making it publicly reachable",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if err := confirmFormsPublish(cmd, "Publish survey?", args[0]); err != nil {
					return err
				}
			}
			svc, err := formsService(cmd)
			if err != nil {
				return err
			}
			_, err = svc.Publish(cmd.Context(), args[0])
			if err != nil {
				return friendlyFormsError(err)
			}
			human := fmt.Sprintf("Published survey %s\n  Public: %s\n  Answers: %s", args[0], formsPublicURL(args[0]), formsAnswersURL(args[0]))
			return emit(cmd, human, surveyStatusPayload{SurveyID: args[0], Status: "published", PublicURL: formsPublicURL(args[0]), AnswersURL: formsAnswersURL(args[0])})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "publish without interactive confirmation")
	return cmd
}

func newFormsUnpublishCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "unpublish <survey-id>",
		Short: "Unpublish a survey, removing public access",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if err := confirmFormsPublish(cmd, "Unpublish survey?", args[0]); err != nil {
					return err
				}
			}
			svc, err := formsService(cmd)
			if err != nil {
				return err
			}
			result, err := svc.Unpublish(cmd.Context(), args[0])
			if err != nil {
				return friendlyFormsError(err)
			}
			return emit(cmd, "Unpublished survey "+args[0], surveyStatusPayload{SurveyID: args[0], Status: "unpublished", Result: result})
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "unpublish without interactive confirmation")
	return cmd
}

type surveyPayload struct {
	ID         string `json:"id"`
	Title      string `json:"title,omitempty"`
	PublicURL  string `json:"public_url"`
	AnswersURL string `json:"answers_url"`
}

type surveyStatusPayload struct {
	SurveyID   string         `json:"survey_id"`
	Status     string         `json:"status"`
	PublicURL  string         `json:"public_url,omitempty"`
	AnswersURL string         `json:"answers_url,omitempty"`
	Result     map[string]any `json:"result,omitempty"`
}

// Yandex Forms public respondent and admin-answers URLs are derived from the
// survey id; the API does not return them. Format verified live 2026-06-20.
func formsPublicURL(surveyID string) string {
	return "https://forms.yandex.ru/cloud/" + surveyID
}

func formsAnswersURL(surveyID string) string {
	return "https://forms.yandex.ru/cloud/admin/" + surveyID + "/answers?view=stats"
}

func formsService(cmd *cobra.Command) (*forms.Service, error) {
	if config.FormsClientID() == "" {
		return nil, errors.New("forms: no Forms OAuth client_id: set YX360_FORMS_CLIENT_ID")
	}
	if config.FormsOrgID() == "" {
		return nil, errors.New("forms: no Forms org id: set YX360_FORMS_ORG_ID")
	}
	store, err := selectStoreFor(formsProfile)
	if err != nil {
		return nil, err
	}
	cred, err := store.Load(cmd.Context())
	if err != nil {
		if errors.Is(err, tokenstore.ErrNoCredential) {
			return nil, forms.ErrReauthRequired
		}
		return nil, err
	}
	return forms.NewService(config.DefaultForms(), cred), nil
}

func friendlyFormsError(err error) error { return err }

func humanFormsResponses(result *forms.ListResponsesResult) string {
	if result == nil || len(result.Answers) == 0 {
		return "No responses"
	}
	var b strings.Builder
	for _, answer := range result.Answers {
		b.WriteString(fmt.Sprintf("%s respondent=%s submitted_at=%s\n", answer.ID, answer.RespondentID, answer.SubmittedAt))
	}
	if result.NextPageToken != "" {
		b.WriteString("next_page_token: " + result.NextPageToken + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func confirmFormsCreate(cmd *cobra.Command, title string) error {
	cmd.Println("Forms create preview:")
	cmd.Println("  Title: " + title)
	return confirmPrompt(cmd, "Create survey?")
}

func confirmFormsPublish(cmd *cobra.Command, prompt, surveyID string) error {
	cmd.Println("Forms publish preview:")
	cmd.Println("  Survey: " + surveyID)
	cmd.Println("  Note: a published form is publicly reachable by anyone with the link")
	return confirmPrompt(cmd, prompt)
}
