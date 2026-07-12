package yx360mcp

import (
	"context"
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/effective-dev-os/yx360-cli/internal/forms"
)

type formsResponsesInput struct {
	SurveyID  string `json:"survey_id"            jsonschema:"survey ID to list responses for"`
	PageSize  int    `json:"page_size,omitempty"  jsonschema:"max responses per page (0 = server default)"`
	PageToken string `json:"page_token,omitempty" jsonschema:"pagination cursor from a previous response"`
}

type formsCreateInput struct {
	Title     string `json:"title"     jsonschema:"survey title"`
	Confirmed bool   `json:"confirmed" jsonschema:"set true to execute; omit for dry-run preview"`
}

type formsPublishInput struct {
	SurveyID  string `json:"survey_id" jsonschema:"survey ID to publish"`
	Confirmed bool   `json:"confirmed" jsonschema:"set true to execute; omit for dry-run preview"`
}

type formsUnpublishInput struct {
	SurveyID  string `json:"survey_id" jsonschema:"survey ID to unpublish"`
	Confirmed bool   `json:"confirmed" jsonschema:"set true to execute; omit for dry-run preview"`
}

func registerFormsTools(srv *sdkmcp.Server, svcFn func(context.Context) (*forms.Service, error)) {
	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "forms_responses",
		Description: "List responses for a Yandex Forms survey.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in formsResponsesInput) (*sdkmcp.CallToolResult, any, error) {
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		result, err := svc.ListResponses(ctx, in.SurveyID, in.PageSize, in.PageToken)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(result)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "forms_create",
		Description: "Create a new Yandex Forms survey. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in formsCreateInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult(fmt.Sprintf("would create survey %q", in.Title))
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		survey, err := svc.CreateSurvey(ctx, in.Title)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(survey)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "forms_publish",
		Description: "Publish a survey, making it publicly reachable. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in formsPublishInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult("would publish survey " + in.SurveyID)
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		result, err := svc.Publish(ctx, in.SurveyID)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(result)
	})

	sdkmcp.AddTool(srv, &sdkmcp.Tool{
		Name:        "forms_unpublish",
		Description: "Unpublish a survey, removing public access. Pass confirmed=true to execute.",
	}, func(ctx context.Context, _ *sdkmcp.CallToolRequest, in formsUnpublishInput) (*sdkmcp.CallToolResult, any, error) {
		if !in.Confirmed {
			return dryRunResult("would unpublish survey " + in.SurveyID)
		}
		svc, err := svcFn(ctx)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		result, err := svc.Unpublish(ctx, in.SurveyID)
		if err != nil {
			return nil, nil, toolErr(err)
		}
		return textResult(result)
	})
}
