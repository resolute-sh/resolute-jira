// Package jira provides Jira integration activities for resolute workflows.
package jira

import (
	"github.com/resolute-sh/resolute/core"
	"go.temporal.io/sdk/worker"
)

const (
	ProviderName    = "resolute-jira"
	ProviderVersion = "1.0.0"
)

// Provider returns the Jira provider for registration.
func Provider() core.Provider {
	return core.NewProvider(ProviderName, ProviderVersion).
		AddActivity("jira.FetchIssues", FetchIssuesActivity).
		AddActivity("jira.FetchIssue", FetchIssueActivity).
		AddActivity("jira.SearchJQL", SearchJQLActivity)
}

// RegisterActivities registers all Jira activities with a Temporal worker.
func RegisterActivities(w worker.Worker) {
	core.RegisterProviderActivities(w, Provider())
}
