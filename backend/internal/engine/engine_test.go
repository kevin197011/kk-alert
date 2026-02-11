package engine

import (
	"testing"
	"time"

	"github.com/kk-alert/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestMatchRule(t *testing.T) {
	alert := &models.Alert{SourceID: 1, Severity: "warning"}
	labels := map[string]string{"job": "api", "env": "prod"}

	// no severity filter -> match
	r := &models.Rule{}
	if !matchRule(r, alert, labels) {
		t.Error("expected match when no filter")
	}

	// severity match
	r.MatchSeverity = "warning"
	if !matchRule(r, alert, labels) {
		t.Error("expected match severity")
	}
	r.MatchSeverity = "critical"
	if matchRule(r, alert, labels) {
		t.Error("expected no match when severity differs")
	}

	// datasource_ids filter
	r.MatchSeverity = ""
	r.DatasourceIDs = "[1,2]"
	if !matchRule(r, alert, labels) {
		t.Error("expected match when source_id in list")
	}
	r.DatasourceIDs = "[2,3]"
	if matchRule(r, alert, labels) {
		t.Error("expected no match when source_id not in list")
	}

	// match_labels
	r.DatasourceIDs = ""
	r.MatchLabels = `{"job":"api"}`
	if !matchRule(r, alert, labels) {
		t.Error("expected match when labels subset match")
	}
	r.MatchLabels = `{"job":"other"}`
	if matchRule(r, alert, labels) {
		t.Error("expected no match when label value differs")
	}
}

func TestDurationSatisfied(t *testing.T) {
	alert := &models.Alert{FiringAt: time.Now().Add(-10 * time.Minute)}
	r := &models.Rule{}

	if !durationSatisfied(r, alert) {
		t.Error("no duration -> satisfied")
	}
	r.Duration = "0"
	if !durationSatisfied(r, alert) {
		t.Error("duration 0 -> satisfied")
	}
	r.Duration = "5m"
	if !durationSatisfied(r, alert) {
		t.Error("firing 10m ago, duration 5m -> satisfied")
	}
	alert.FiringAt = time.Now().Add(-2 * time.Minute)
	if durationSatisfied(r, alert) {
		t.Error("firing 2m ago, duration 5m -> not satisfied")
	}
}

func TestParseHM(t *testing.T) {
	if parseHM("22:00") != 22*60 {
		t.Error("22:00")
	}
	if parseHM("08:30") != 8*60+30 {
		t.Error("08:30")
	}
	if parseHM("00:00") != 0 {
		t.Error("00:00")
	}
	if parseHM("invalid") != -1 {
		t.Error("invalid")
	}
}

func TestInExcludeWindow(t *testing.T) {
	r := &models.Rule{}
	if inExcludeWindow(r) {
		t.Error("empty -> not in window")
	}
	// window 00:00-23:59 excludes all day; we can't assert current time so test parse only
	r.ExcludeWindows = `[{"start":"00:00","end":"00:00"}]`
	// 00:00-00:00 means no range (start==end) so we don't match
	if inExcludeWindow(r) {
		t.Error("00:00-00:00")
	}
}

func TestLabelsMatch(t *testing.T) {
	labels := map[string]string{"job": "api", "env": "prod"}
	if !labelsMatch(labels, map[string]string{"job": "api"}) {
		t.Error("subset match")
	}
	if !labelsMatch(labels, map[string]string{"job": "api", "env": "prod"}) {
		t.Error("full match")
	}
	if labelsMatch(labels, map[string]string{"job": "other"}) {
		t.Error("value diff")
	}
	if labelsMatch(labels, map[string]string{}) {
		t.Error("empty want should not match")
	}
}

func TestSuppression(t *testing.T) {
	r := &models.Rule{ID: 1, Suppression: `{"source_labels":{"job":"api"},"suppressed_labels":{"env":"prod"},"duration":"30m"}`}
	updateSuppressionWindow(r, map[string]string{"job": "api"})
	if !suppressed(r, map[string]string{"env": "prod"}) {
		t.Error("expected suppressed: window active and labels match suppressed_labels")
	}
	if suppressed(r, map[string]string{"env": "other"}) {
		t.Error("labels do not match suppressed_labels -> not suppressed")
	}
}

func TestProcessAlertNoPanic(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.AutoMigrate(&models.Rule{}, &models.Alert{}, &models.Channel{}, &models.Template{}, &models.AlertSendRecord{})
	alert := &models.Alert{ID: "test-1", SourceID: 1, SourceType: "prometheus", Title: "Test", Severity: "warning", Status: "firing", FiringAt: time.Now(), Labels: "{}", Annotations: "{}"}
	ProcessAlert(db, alert)
	// no rules -> no send; should not panic
}

func TestTypeFingerprintAndAggregationKey(t *testing.T) {
	labels := map[string]string{"job": "api", "hostname": "h1", "env": "prod"}
	fp := typeFingerprint(labels, "hostname")
	if fp == "" {
		t.Error("expected non-empty fingerprint")
	}
	if key := aggregationKey(labels, "hostname"); key != "h1" {
		t.Errorf("aggregationKey hostname: got %q", key)
	}
	labels2 := map[string]string{"job": "api", "hostname": "h2", "env": "prod"}
	if typeFingerprint(labels2, "hostname") != fp {
		t.Error("same type (different hostname) should have same fingerprint")
	}
	if !labelsSameType(labels, labels2, "hostname") {
		t.Error("expected same type for different hostname only")
	}
}
