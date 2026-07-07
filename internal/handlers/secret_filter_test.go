package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/grafana/loki/v3/pkg/loghttp"
)

const testGenericAPIKey = "aZ9xY8wV7uT6sR5qP4nM3kL2jH1gF0dS" //gitleaks:allow — fake fixture for filter tests

func TestFilterSecretEntries_OmitsMatchingLines(t *testing.T) {
	if err := InitializeSecretFilterConfig(true); err != nil {
		t.Fatalf("InitializeSecretFilterConfig failed: %v", err)
	}
	defer ResetSecretFilterConfig()

	entries := []logEntry{
		{
			Timestamp: time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC),
			Line:      "normal application log",
			Labels:    loghttp.LabelSet{"job": "api"},
		},
		{
			Timestamp: time.Date(2026, 7, 7, 10, 0, 1, 0, time.UTC),
			Line:      `api_key = "` + testGenericAPIKey + `"`,
			Labels:    loghttp.LabelSet{"job": "api"},
		},
	}

	filtered, summary := filterSecretEntries(context.Background(), entries)

	if summary.LinesOmitted != 1 {
		t.Fatalf("expected 1 omitted line, got %d", summary.LinesOmitted)
	}
	if summary.Findings == 0 {
		t.Fatal("expected at least one secret finding")
	}
	if len(filtered) != 1 {
		t.Fatalf("expected 1 filtered entry, got %d", len(filtered))
	}
	if filtered[0].Line != "normal application log" {
		t.Fatalf("unexpected remaining line: %q", filtered[0].Line)
	}
}

func TestFilterSecretEntries_DisabledKeepsLines(t *testing.T) {
	if err := InitializeSecretFilterConfig(false); err != nil {
		t.Fatalf("InitializeSecretFilterConfig failed: %v", err)
	}
	defer ResetSecretFilterConfig()

	entries := []logEntry{{Line: `api_key = "` + testGenericAPIKey + `"`}}

	filtered, summary := filterSecretEntries(context.Background(), entries)

	if summary.LinesOmitted != 0 || summary.Findings != 0 {
		t.Fatalf("expected empty summary, got %+v", summary)
	}
	if len(filtered) != 1 {
		t.Fatalf("expected disabled filter to keep the entry, got %d entries", len(filtered))
	}
}

func TestFormatSecretOmissionNote(t *testing.T) {
	if got := formatSecretOmissionNote(1); got != "1 line omitted due to detected secrets" {
		t.Fatalf("unexpected singular note: %q", got)
	}
	if got := formatSecretOmissionNote(3); got != "3 lines omitted due to detected secrets" {
		t.Fatalf("unexpected plural note: %q", got)
	}
}
