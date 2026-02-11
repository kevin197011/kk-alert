package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Config from rule JiraConfig JSON. For Jira Cloud use Email + Token (API token) as basic auth.
type Config struct {
	BaseURL   string `json:"base_url"`
	Email     string `json:"email"`
	Token     string `json:"token"`
	Project   string `json:"project"`
	IssueType string `json:"issue_type"`
}

// CreateIssue creates a Jira issue and returns the issue key (e.g. PROJ-123).
func CreateIssue(cfg *Config, summary, description string) (string, error) {
	if cfg == nil || cfg.BaseURL == "" || cfg.Project == "" {
		return "", fmt.Errorf("jira config missing base_url or project")
	}
	if cfg.IssueType == "" {
		cfg.IssueType = "Task"
	}
	url := strings.TrimSuffix(cfg.BaseURL, "/") + "/rest/api/3/issue"
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	// Jira Cloud REST v3: description is ADF (Atlassian Document Format).
	descDoc := map[string]interface{}{
		"type": "doc", "version": 1,
		"content": []map[string]interface{}{
			{"type": "paragraph", "content": []map[string]interface{}{{"type": "text", "text": description}}},
		},
	}
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project":     map[string]string{"key": cfg.Project},
			"summary":     summary,
			"description": descDoc,
			"issuetype":   map[string]string{"name": cfg.IssueType},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if cfg.Token != "" {
		if cfg.Email != "" {
			req.SetBasicAuth(cfg.Email, cfg.Token)
		} else {
			req.Header.Set("Authorization", "Bearer "+cfg.Token)
		}
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		var errBody bytes.Buffer
		_, _ = errBody.ReadFrom(resp.Body)
		return "", fmt.Errorf("jira api %d: %s", resp.StatusCode, errBody.String())
	}
	// Read response body for key (we haven't consumed it yet in success path)
	var result struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Key, nil
}
