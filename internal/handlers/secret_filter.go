package handlers

import (
	"context"
	"fmt"
	"sync"

	"github.com/betterleaks/betterleaks/detect"
)

// SecretFilterConfig holds the global secret filtering configuration.
type SecretFilterConfig struct {
	Enabled  bool
	detector *detect.Detector
}

// SecretFilterSummary describes content omitted from tool output.
type SecretFilterSummary struct {
	LinesOmitted int
	Findings     int
}

var (
	secretFilterConfig *SecretFilterConfig
	secretFilterMu     sync.RWMutex
)

// InitializeSecretFilterConfig creates the shared detector used to scan returned logs.
func InitializeSecretFilterConfig(enabled bool) error {
	cfg := &SecretFilterConfig{Enabled: enabled}
	if enabled {
		detector, err := detect.NewDetectorDefaultConfig()
		if err != nil {
			return fmt.Errorf("initialize secret detector: %w", err)
		}
		detector.SkipFindingAppend = true
		cfg.detector = detector
	}

	secretFilterMu.Lock()
	defer secretFilterMu.Unlock()
	secretFilterConfig = cfg
	return nil
}

// GetSecretFilterConfig returns the current secret filter configuration.
func GetSecretFilterConfig() *SecretFilterConfig {
	secretFilterMu.RLock()
	defer secretFilterMu.RUnlock()
	return secretFilterConfig
}

// ResetSecretFilterConfig clears the secret filter configuration (for testing).
func ResetSecretFilterConfig() {
	secretFilterMu.Lock()
	defer secretFilterMu.Unlock()
	secretFilterConfig = nil
}

func filterSecretEntries(ctx context.Context, entries []logEntry) ([]logEntry, SecretFilterSummary) {
	cfg := GetSecretFilterConfig()
	if cfg == nil || !cfg.Enabled || cfg.detector == nil || len(entries) == 0 {
		return entries, SecretFilterSummary{}
	}

	filtered := make([]logEntry, 0, len(entries))
	var summary SecretFilterSummary
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return filtered, summary
		default:
		}

		findings := cfg.detector.DetectString(entry.Line)
		if len(findings) > 0 {
			summary.LinesOmitted++
			summary.Findings += len(findings)
			continue
		}
		filtered = append(filtered, entry)
	}

	return filtered, summary
}

func formatSecretOmissionNote(lines int) string {
	if lines == 1 {
		return "1 line omitted due to detected secrets"
	}
	return fmt.Sprintf("%d lines omitted due to detected secrets", lines)
}
