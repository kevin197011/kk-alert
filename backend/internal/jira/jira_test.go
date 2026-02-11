package jira

import (
	"testing"
)

func TestCreateIssue_InvalidConfig(t *testing.T) {
	_, err := CreateIssue(nil, "summary", "body")
	if err == nil {
		t.Fatal("expected error for nil config")
	}
	_, err = CreateIssue(&Config{Project: "PROJ"}, "s", "b")
	if err == nil {
		t.Fatal("expected error for missing base_url")
	}
	_, err = CreateIssue(&Config{BaseURL: "https://x.atlassian.net"}, "s", "b")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}
