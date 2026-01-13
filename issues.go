package jira

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/resolute-sh/resolute/core"
	transform "github.com/resolute-sh/resolute-transform"
)

// FetchIssuesInput is the input for FetchIssuesActivity.
type FetchIssuesInput struct {
	BaseURL    string
	Email      string
	APIToken   string
	Project    string
	Since      *time.Time
	MaxResults int
}

// FetchIssuesOutput is the output of FetchIssuesActivity.
type FetchIssuesOutput struct {
	Ref   core.DataRef
	Count int
	Total int
}

// FetchIssuesActivity fetches issues from a Jira project and stores them.
func FetchIssuesActivity(ctx context.Context, input FetchIssuesInput) (FetchIssuesOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL:  input.BaseURL,
		Email:    input.Email,
		APIToken: input.APIToken,
	})

	jql := fmt.Sprintf("project = %s ORDER BY updated DESC", input.Project)
	if input.Since != nil {
		jql = fmt.Sprintf("project = %s AND updated >= '%s' ORDER BY updated DESC",
			input.Project, input.Since.Format("2006-01-02 15:04"))
	}

	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 100
	}

	result, err := client.SearchJQL(ctx, jql, maxResults)
	if err != nil {
		return FetchIssuesOutput{}, fmt.Errorf("search jql: %w", err)
	}

	docs := make([]transform.Document, 0, len(result.Issues))
	for _, issue := range result.Issues {
		doc := issueToDocument(issue)
		docs = append(docs, doc)
	}

	ref, err := transform.StoreDocuments(ctx, docs)
	if err != nil {
		return FetchIssuesOutput{}, fmt.Errorf("store documents: %w", err)
	}

	return FetchIssuesOutput{
		Ref:   ref,
		Count: len(docs),
		Total: result.Total,
	}, nil
}

// FetchIssueInput is the input for FetchIssueActivity.
type FetchIssueInput struct {
	BaseURL  string
	Email    string
	APIToken string
	IssueKey string
}

// FetchIssueOutput is the output of FetchIssueActivity.
type FetchIssueOutput struct {
	Document transform.Document
	Found    bool
}

// FetchIssueActivity fetches a single issue by key.
func FetchIssueActivity(ctx context.Context, input FetchIssueInput) (FetchIssueOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL:  input.BaseURL,
		Email:    input.Email,
		APIToken: input.APIToken,
	})

	issue, err := client.GetIssue(ctx, input.IssueKey)
	if err != nil {
		return FetchIssueOutput{}, fmt.Errorf("get issue: %w", err)
	}

	return FetchIssueOutput{
		Document: issueToDocument(*issue),
		Found:    true,
	}, nil
}

// SearchJQLInput is the input for SearchJQLActivity.
type SearchJQLInput struct {
	BaseURL    string
	Email      string
	APIToken   string
	JQL        string
	MaxResults int
}

// SearchJQLOutput is the output of SearchJQLActivity.
type SearchJQLOutput struct {
	Ref   core.DataRef
	Count int
	Total int
}

// SearchJQLActivity searches for issues using JQL and stores them.
func SearchJQLActivity(ctx context.Context, input SearchJQLInput) (SearchJQLOutput, error) {
	client := NewClient(ClientConfig{
		BaseURL:  input.BaseURL,
		Email:    input.Email,
		APIToken: input.APIToken,
	})

	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 100
	}

	result, err := client.SearchJQL(ctx, input.JQL, maxResults)
	if err != nil {
		return SearchJQLOutput{}, fmt.Errorf("search jql: %w", err)
	}

	docs := make([]transform.Document, 0, len(result.Issues))
	for _, issue := range result.Issues {
		doc := issueToDocument(issue)
		docs = append(docs, doc)
	}

	ref, err := transform.StoreDocuments(ctx, docs)
	if err != nil {
		return SearchJQLOutput{}, fmt.Errorf("store documents: %w", err)
	}

	return SearchJQLOutput{
		Ref:   ref,
		Count: len(docs),
		Total: result.Total,
	}, nil
}

// issueToDocument converts a Jira issue to a transform.Document.
func issueToDocument(issue Issue) transform.Document {
	content := issue.Fields.Summary
	if issue.Fields.Description != "" {
		content += "\n\n" + issue.Fields.Description
	}

	if issue.Fields.Comments != nil {
		for _, comment := range issue.Fields.Comments.Comments {
			content += fmt.Sprintf("\n\n[Comment by %s]: %s",
				comment.Author.DisplayName, comment.Body)
		}
	}

	var updatedAt time.Time
	if issue.Fields.Updated != "" {
		updatedAt, _ = time.Parse("2006-01-02T15:04:05.000-0700", issue.Fields.Updated)
	}

	metadata := map[string]string{
		"issue_key":   issue.Key,
		"project":     issue.Fields.Project.Key,
		"status":      issue.Fields.Status.Name,
		"issue_type":  issue.Fields.IssueType.Name,
	}

	if issue.Fields.Priority != nil {
		metadata["priority"] = issue.Fields.Priority.Name
	}

	if issue.Fields.Assignee != nil {
		metadata["assignee"] = issue.Fields.Assignee.DisplayName
	}

	return transform.Document{
		ID:        issue.Key,
		Content:   content,
		Title:     issue.Fields.Summary,
		Source:    "jira",
		URL:       issue.Self,
		Metadata:  metadata,
		UpdatedAt: updatedAt,
	}
}

// FetchIssues creates a node for fetching Jira issues.
func FetchIssues(input FetchIssuesInput) *core.Node[FetchIssuesInput, FetchIssuesOutput] {
	return core.NewNode("jira.FetchIssues", FetchIssuesActivity, input)
}

// FetchIssue creates a node for fetching a single Jira issue.
func FetchIssue(input FetchIssueInput) *core.Node[FetchIssueInput, FetchIssueOutput] {
	return core.NewNode("jira.FetchIssue", FetchIssueActivity, input)
}

// SearchJQL creates a node for searching Jira with JQL.
func SearchJQL(input SearchJQLInput) *core.Node[SearchJQLInput, SearchJQLOutput] {
	return core.NewNode("jira.SearchJQL", SearchJQLActivity, input)
}

// FetchAllIssuesConfig contains configuration for fetching all issues.
type FetchAllIssuesConfig struct {
	BaseURL    string
	Email      string
	APIToken   string
	Project    string
	Since      *time.Time
	MaxResults int // per page, default 100
}

// FetchAllIssuesOutput is the output of FetchAllIssuesActivity.
type FetchAllIssuesOutput struct {
	Ref        core.DataRef
	Count      int
	PageCount  int
	FinalCursor string
}

// FetchAllIssues creates a node that fetches ALL issues using pagination.
// Unlike FetchIssues which fetches a single page, this fetches all pages.
func FetchAllIssues(config FetchAllIssuesConfig) *core.Node[core.PaginateWithInputParams[FetchAllIssuesConfig], core.PaginateWithInputOutput[Issue, FetchAllIssuesConfig]] {
	fetcher := func(ctx context.Context, cfg FetchAllIssuesConfig, cursor string) (core.PageResult[Issue], error) {
		client := NewClient(ClientConfig{
			BaseURL:  cfg.BaseURL,
			Email:    cfg.Email,
			APIToken: cfg.APIToken,
		})

		jql := fmt.Sprintf("project = %s ORDER BY updated DESC", cfg.Project)
		if cfg.Since != nil {
			jql = fmt.Sprintf("project = %s AND updated >= '%s' ORDER BY updated DESC",
				cfg.Project, cfg.Since.Format("2006-01-02 15:04"))
		}

		startAt := 0
		if cursor != "" {
			var err error
			startAt, err = strconv.Atoi(cursor)
			if err != nil {
				return core.PageResult[Issue]{}, fmt.Errorf("parse cursor: %w", err)
			}
		}

		maxResults := cfg.MaxResults
		if maxResults <= 0 {
			maxResults = 100
		}

		result, err := client.SearchJQLWithParams(ctx, SearchJQLParams{
			JQL:        jql,
			StartAt:    startAt,
			MaxResults: maxResults,
		})
		if err != nil {
			return core.PageResult[Issue]{}, fmt.Errorf("search jql: %w", err)
		}

		nextStartAt := startAt + len(result.Issues)
		hasMore := nextStartAt < result.Total
		nextCursor := ""
		if hasMore {
			nextCursor = strconv.Itoa(nextStartAt)
		}

		return core.PageResult[Issue]{
			Items:      result.Issues,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		}, nil
	}

	return core.PaginateWithConfig[Issue, FetchAllIssuesConfig]("jira.FetchAllIssues", fetcher).
		WithTimeout(30 * time.Minute) // Long timeout for large datasets
}

// SearchAllJQL creates a node that fetches ALL issues matching a JQL query using pagination.
type SearchAllJQLConfig struct {
	BaseURL    string
	Email      string
	APIToken   string
	JQL        string
	MaxResults int // per page, default 100
}

// SearchAllJQL creates a node that searches with JQL and fetches all results.
func SearchAllJQL(config SearchAllJQLConfig) *core.Node[core.PaginateWithInputParams[SearchAllJQLConfig], core.PaginateWithInputOutput[Issue, SearchAllJQLConfig]] {
	fetcher := func(ctx context.Context, cfg SearchAllJQLConfig, cursor string) (core.PageResult[Issue], error) {
		client := NewClient(ClientConfig{
			BaseURL:  cfg.BaseURL,
			Email:    cfg.Email,
			APIToken: cfg.APIToken,
		})

		startAt := 0
		if cursor != "" {
			var err error
			startAt, err = strconv.Atoi(cursor)
			if err != nil {
				return core.PageResult[Issue]{}, fmt.Errorf("parse cursor: %w", err)
			}
		}

		maxResults := cfg.MaxResults
		if maxResults <= 0 {
			maxResults = 100
		}

		result, err := client.SearchJQLWithParams(ctx, SearchJQLParams{
			JQL:        cfg.JQL,
			StartAt:    startAt,
			MaxResults: maxResults,
		})
		if err != nil {
			return core.PageResult[Issue]{}, fmt.Errorf("search jql: %w", err)
		}

		nextStartAt := startAt + len(result.Issues)
		hasMore := nextStartAt < result.Total
		nextCursor := ""
		if hasMore {
			nextCursor = strconv.Itoa(nextStartAt)
		}

		return core.PageResult[Issue]{
			Items:      result.Issues,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		}, nil
	}

	return core.PaginateWithConfig[Issue, SearchAllJQLConfig]("jira.SearchAllJQL", fetcher).
		WithTimeout(30 * time.Minute)
}
