package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a Jira REST API client.
type Client struct {
	baseURL    string
	email      string
	apiToken   string
	httpClient *http.Client
}

// ClientConfig contains configuration for creating a Jira client.
type ClientConfig struct {
	BaseURL  string
	Email    string
	APIToken string
	Timeout  time.Duration
}

// NewClient creates a new Jira client.
func NewClient(cfg ClientConfig) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL:  cfg.BaseURL,
		email:    cfg.Email,
		apiToken: cfg.APIToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Issue represents a Jira issue.
type Issue struct {
	ID     string      `json:"id"`
	Key    string      `json:"key"`
	Self   string      `json:"self"`
	Fields IssueFields `json:"fields"`
}

// IssueFields contains the fields of a Jira issue.
type IssueFields struct {
	Summary     string      `json:"summary"`
	Description string      `json:"description"`
	Status      Status      `json:"status"`
	IssueType   IssueType   `json:"issuetype"`
	Project     Project     `json:"project"`
	Created     string      `json:"created"`
	Updated     string      `json:"updated"`
	Labels      []string    `json:"labels"`
	Priority    *Priority   `json:"priority"`
	Assignee    *User       `json:"assignee"`
	Reporter    *User       `json:"reporter"`
	Comments    *Comments   `json:"comment"`
}

// Status represents an issue status.
type Status struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// IssueType represents an issue type.
type IssueType struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Project represents a Jira project.
type Project struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Priority represents issue priority.
type Priority struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// User represents a Jira user.
type User struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
	AccountID    string `json:"accountId"`
}

// Comments represents issue comments.
type Comments struct {
	Total    int       `json:"total"`
	Comments []Comment `json:"comments"`
}

// Comment represents a single comment.
type Comment struct {
	ID      string `json:"id"`
	Body    string `json:"body"`
	Author  User   `json:"author"`
	Created string `json:"created"`
	Updated string `json:"updated"`
}

// SearchResult represents a JQL search result.
type SearchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
}

// SearchJQLInput contains parameters for JQL search.
type SearchJQLParams struct {
	JQL        string
	StartAt    int
	MaxResults int
}

// SearchJQL searches for issues using JQL.
func (c *Client) SearchJQL(ctx context.Context, jql string, maxResults int) (*SearchResult, error) {
	return c.SearchJQLWithParams(ctx, SearchJQLParams{
		JQL:        jql,
		StartAt:    0,
		MaxResults: maxResults,
	})
}

// SearchJQLWithParams searches for issues using JQL with full pagination control.
func (c *Client) SearchJQLWithParams(ctx context.Context, params SearchJQLParams) (*SearchResult, error) {
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 50
	}

	endpoint := fmt.Sprintf("%s/rest/api/3/search?jql=%s&startAt=%d&maxResults=%d",
		c.baseURL, url.QueryEscape(params.JQL), params.StartAt, maxResults)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// GetIssue fetches a single issue by key.
func (c *Client) GetIssue(ctx context.Context, issueKey string) (*Issue, error) {
	endpoint := fmt.Sprintf("%s/rest/api/3/issue/%s", c.baseURL, issueKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira API error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var issue Issue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &issue, nil
}

func (c *Client) setAuth(req *http.Request) {
	req.SetBasicAuth(c.email, c.apiToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}
