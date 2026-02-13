package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bakerweb/wt/internal/connector"
)

// Client implements the connector.Connector interface for Jira.
type Client struct {
	BaseURL  string
	Email    string
	APIToken string
	client   *http.Client
}

// New creates a new Jira client.
func New(baseURL, email, apiToken string) *Client {
	return &Client{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		Email:    email,
		APIToken: apiToken,
		client:   &http.Client{},
	}
}

func (c *Client) Name() string { return "jira" }

func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.Email, c.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.client.Do(req)
}

// jiraIssue represents the JSON structure of a Jira issue.
type jiraIssue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
		Assignee *struct {
			DisplayName  string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
		} `json:"assignee"`
		Labels []string `json:"labels"`
	} `json:"fields"`
}

func issueToTicket(issue jiraIssue, baseURL string) *connector.Ticket {
	t := &connector.Ticket{
		Key:         issue.Key,
		Summary:     issue.Fields.Summary,
		Description: issue.Fields.Description,
		Status:      issue.Fields.Status.Name,
		Labels:      issue.Fields.Labels,
		URL:         baseURL + "/browse/" + issue.Key,
	}
	if issue.Fields.Assignee != nil {
		t.Assignee = issue.Fields.Assignee.DisplayName
	}
	return t
}

func (c *Client) GetTicket(ctx context.Context, key string) (*connector.Ticket, error) {
	resp, err := c.doRequest(ctx, "GET", "/rest/api/3/issue/"+key, nil)
	if err != nil {
		return nil, fmt.Errorf("jira request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira returned %d: %s", resp.StatusCode, string(body))
	}

	var issue jiraIssue
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("failed to decode jira response: %w", err)
	}
	return issueToTicket(issue, c.BaseURL), nil
}

func (c *Client) ListAssigned(ctx context.Context) ([]connector.Ticket, error) {
	jql := "assignee=currentUser() AND statusCategory != Done ORDER BY updated DESC"
	resp, err := c.doRequest(ctx, "GET", "/rest/api/3/search?jql="+jql+"&maxResults=50", nil)
	if err != nil {
		return nil, fmt.Errorf("jira request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Issues []jiraIssue `json:"issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode jira response: %w", err)
	}

	tickets := make([]connector.Ticket, 0, len(result.Issues))
	for _, issue := range result.Issues {
		tickets = append(tickets, *issueToTicket(issue, c.BaseURL))
	}
	return tickets, nil
}

// jiraTransition represents a Jira status transition.
type jiraTransition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	To   struct {
		Name string `json:"name"`
	} `json:"to"`
}

func (c *Client) TransitionTicket(ctx context.Context, key, status string) error {
	// First, get available transitions
	resp, err := c.doRequest(ctx, "GET", "/rest/api/3/issue/"+key+"/transitions", nil)
	if err != nil {
		return fmt.Errorf("failed to get transitions: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Transitions []jiraTransition `json:"transitions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode transitions: %w", err)
	}

	// Find matching transition
	var transitionID string
	statusLower := strings.ToLower(status)
	for _, t := range result.Transitions {
		if strings.ToLower(t.To.Name) == statusLower || strings.ToLower(t.Name) == statusLower {
			transitionID = t.ID
			break
		}
	}
	if transitionID == "" {
		available := make([]string, 0, len(result.Transitions))
		for _, t := range result.Transitions {
			available = append(available, t.To.Name)
		}
		return fmt.Errorf("no transition to %q found (available: %s)", status, strings.Join(available, ", "))
	}

	// Execute transition
	body := fmt.Sprintf(`{"transition":{"id":"%s"}}`, transitionID)
	resp2, err := c.doRequest(ctx, "POST", "/rest/api/3/issue/"+key+"/transitions", strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to transition issue: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusNoContent && resp2.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("jira transition failed with %d: %s", resp2.StatusCode, string(respBody))
	}
	return nil
}

func (c *Client) Validate(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "GET", "/rest/api/3/myself", nil)
	if err != nil {
		return fmt.Errorf("failed to connect to jira: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jira authentication failed (status %d)", resp.StatusCode)
	}
	return nil
}
