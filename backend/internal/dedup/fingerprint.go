package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
)

// Key returns a deterministic fingerprint for alert deduplication:
// same (source_id, title, labels) => same key => same logical alert.
// Used as ExternalID so we reuse one alert record until it is resolved.
func Key(sourceID uint, title string, labels map[string]string) string {
	return keyWithDisambiguator(sourceID, title, labels, -1)
}

// KeyForSeries is used by the scheduler when iterating over query results.
// If labels do not contain "instance" or "job", resultIndex is appended so each
// series gets a distinct key (one alert per result). Otherwise same as Key.
// When resultIndex >= 0 and labels lack instance/job, key stability across
// runs depends on result order; reordering may cause resolve+recreate.
func KeyForSeries(sourceID uint, title string, labels map[string]string, resultIndex int) string {
	return keyWithDisambiguator(sourceID, title, labels, resultIndex)
}

// KeyForSeriesWithRule is used by the scheduler so that different rules produce
// different alerts for the same instance (same title+labels). Without ruleID,
// multiple rules with the same name would share one alert row and only the last
// would appear (e.g. 3 rules x 7 instances = 21 triggers but only 7 alerts).
func KeyForSeriesWithRule(sourceID uint, ruleID uint, title string, labels map[string]string, resultIndex int) string {
	if labels == nil {
		labels = make(map[string]string)
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	labelsSorted := make(map[string]string, len(labels))
	for _, k := range keys {
		labelsSorted[k] = labels[k]
	}
	labelsJSON, _ := json.Marshal(labelsSorted)
	if labelsJSON == nil {
		labelsJSON = []byte("{}")
	}
	payload := strconv.FormatUint(uint64(sourceID), 10) + "|" + strconv.FormatUint(uint64(ruleID), 10) + "|" + title + "|" + string(labelsJSON)
	if resultIndex >= 0 && labels["instance"] == "" && labels["job"] == "" {
		payload += "|" + strconv.Itoa(resultIndex)
	}
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

func keyWithDisambiguator(sourceID uint, title string, labels map[string]string, resultIndex int) string {
	if labels == nil {
		labels = make(map[string]string)
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	labelsSorted := make(map[string]string, len(labels))
	for _, k := range keys {
		labelsSorted[k] = labels[k]
	}
	labelsJSON, _ := json.Marshal(labelsSorted)
	if labelsJSON == nil {
		labelsJSON = []byte("{}")
	}
	payload := strconv.FormatUint(uint64(sourceID), 10) + "|" + title + "|" + string(labelsJSON)
	// When labels lack instance/job, use result index so multiple series become multiple alerts
	if resultIndex >= 0 && labels["instance"] == "" && labels["job"] == "" {
		payload += "|" + strconv.Itoa(resultIndex)
	}
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}
