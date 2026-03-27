package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/grafana/loki/v3/pkg/loghttp"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestFormatLokiResults_TimestampParsing tests that timestamps from Loki are correctly formatted
// With the official client, timestamps are already parsed as time.Time, so this test
// verifies that formatting works correctly.
func TestFormatLokiResults_TimestampParsing(t *testing.T) {
	// Test case with known timestamp: 2024-01-15T10:30:45Z
	timestamp := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	// Create a sample Loki result with the test timestamp
	result := &loghttp.QueryResponse{
		Status: "success",
		Data: loghttp.QueryResponseData{
			ResultType: loghttp.ResultTypeStream,
			Result: loghttp.Streams{
				{
					Labels: loghttp.LabelSet{
						"job":   "test-job",
						"level": "info",
					},
					Entries: []loghttp.Entry{
						{Timestamp: timestamp, Line: "Test log message"},
					},
				},
			},
		},
	}

	// Format the results
	output, err := formatLokiResults(result, "text")
	if err != nil {
		t.Fatalf("formatLokiResults failed: %v", err)
	}

	// Verify the output contains the correct timestamp
	if !strings.Contains(output, "2024-01-15T") {
		t.Errorf("Expected output to contain date '2024-01-15T', but got:\n%s", output)
	}

	// Verify it doesn't contain the year 2262 (the bug we fixed)
	if strings.Contains(output, "2262") {
		t.Errorf("Output contains year 2262, indicating timestamp parsing bug is present:\n%s", output)
	}

	// Verify it contains the expected log message
	if !strings.Contains(output, "Test log message") {
		t.Errorf("Expected output to contain 'Test log message', but got:\n%s", output)
	}

	// Verify it contains the stream information
	if !strings.Contains(output, "job") && !strings.Contains(output, "test-job") {
		t.Errorf("Expected output to contain stream info 'job' or 'test-job', but got:\n%s", output)
	}
}

// TestFormatLokiResults_MultipleTimestamps tests formatting of multiple log entries with different timestamps
func TestFormatLokiResults_MultipleTimestamps(t *testing.T) {
	result := &loghttp.QueryResponse{
		Status: "success",
		Data: loghttp.QueryResponseData{
			ResultType: loghttp.ResultTypeStream,
			Result: loghttp.Streams{
				{
					Labels: loghttp.LabelSet{
						"job": "test-job",
					},
					Entries: []loghttp.Entry{
						{Timestamp: time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC), Line: "First log message"},
						{Timestamp: time.Date(2024, 1, 15, 10, 31, 0, 0, time.UTC), Line: "Second log message"},
						{Timestamp: time.Date(2024, 1, 15, 10, 31, 15, 0, time.UTC), Line: "Third log message"},
					},
				},
			},
		},
	}

	output, err := formatLokiResults(result, "text")
	if err != nil {
		t.Fatalf("formatLokiResults failed: %v", err)
	}

	// Check that all timestamps are in 2024, not 2262
	if strings.Contains(output, "2262") {
		t.Errorf("Output contains year 2262, indicating timestamp parsing bug:\n%s", output)
	}

	// All timestamps should be in 2024
	occurrences := strings.Count(output, "2024")
	if occurrences < 3 {
		t.Errorf("Expected at least 3 occurrences of '2024' in output, but found %d:\n%s", occurrences, output)
	}
}

// TestFormatLokiResults_EmptyResult tests handling of empty results
func TestFormatLokiResults_EmptyResult(t *testing.T) {
	result := &loghttp.QueryResponse{
		Status: "success",
		Data: loghttp.QueryResponseData{
			ResultType: loghttp.ResultTypeStream,
			Result:     loghttp.Streams{},
		},
	}

	output, err := formatLokiResults(result, "text")
	if err != nil {
		t.Fatalf("formatLokiResults failed: %v", err)
	}

	expected := "No logs found matching the query"
	if output != expected {
		t.Errorf("Expected output '%s', but got '%s'", expected, output)
	}
}

// TestFormatLokiResults_RecentTimestamp tests with a very recent timestamp to ensure current dates work
func TestFormatLokiResults_RecentTimestamp(t *testing.T) {
	// Use current time
	now := time.Now().UTC()

	result := &loghttp.QueryResponse{
		Status: "success",
		Data: loghttp.QueryResponseData{
			ResultType: loghttp.ResultTypeStream,
			Result: loghttp.Streams{
				{
					Labels: loghttp.LabelSet{
						"job": "recent-test",
					},
					Entries: []loghttp.Entry{
						{Timestamp: now, Line: "Recent log message"},
					},
				},
			},
		},
	}

	output, err := formatLokiResults(result, "text")
	if err != nil {
		t.Fatalf("formatLokiResults failed: %v", err)
	}

	// Should contain current year, not 2262
	currentYear := now.Format("2006")
	if !strings.Contains(output, currentYear) {
		t.Errorf("Expected output to contain current year %s, but got:\n%s", currentYear, output)
	}

	if strings.Contains(output, "2262") {
		t.Errorf("Output contains year 2262, indicating timestamp parsing bug:\n%s", output)
	}
}

// TestFormatLokiResults_NoYear2262Bug is a regression test for the specific bug reported in issue #3
// This test ensures that timestamps never show year 2262 due to incorrect nanosecond conversion
func TestFormatLokiResults_NoYear2262Bug(t *testing.T) {
	// This test uses a variety of realistic timestamps
	testCases := []struct {
		name         string
		timestamp    time.Time
		expectedYear string
	}{
		{
			name:         "Current timestamp",
			timestamp:    time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC),
			expectedYear: "2024",
		},
		{
			name:         "Recent timestamp",
			timestamp:    time.Date(2023, 11, 14, 22, 13, 20, 0, time.UTC),
			expectedYear: "2023",
		},
		{
			name:         "Future timestamp",
			timestamp:    time.Date(2027, 1, 11, 2, 13, 20, 0, time.UTC),
			expectedYear: "2027",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := &loghttp.QueryResponse{
				Status: "success",
				Data: loghttp.QueryResponseData{
					ResultType: loghttp.ResultTypeStream,
					Result: loghttp.Streams{
						{
							Labels: loghttp.LabelSet{
								"job": "regression-test",
							},
							Entries: []loghttp.Entry{
								{Timestamp: tc.timestamp, Line: "Test log message"},
							},
						},
					},
				},
			}

			output, err := formatLokiResults(result, "text")
			if err != nil {
				t.Fatalf("formatLokiResults failed: %v", err)
			}

			// The main regression check: ensure we never see year 2262
			if strings.Contains(output, "2262") {
				t.Errorf("REGRESSION: Output contains year 2262, the original bug is present:\n%s", output)
			}

			// Verify we see the expected year instead
			if !strings.Contains(output, tc.expectedYear) {
				t.Errorf("Expected output to contain year %s, but got:\n%s", tc.expectedYear, output)
			}
		})
	}
}

// TestFormatLokiResults_RawFormat tests the raw output format
func TestFormatLokiResults_RawFormat(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	result := &loghttp.QueryResponse{
		Status: "success",
		Data: loghttp.QueryResponseData{
			ResultType: loghttp.ResultTypeStream,
			Result: loghttp.Streams{
				{
					Labels: loghttp.LabelSet{
						"job": "test-job",
					},
					Entries: []loghttp.Entry{
						{Timestamp: timestamp, Line: "Test log message"},
					},
				},
			},
		},
	}

	output, err := formatLokiResults(result, "raw")
	if err != nil {
		t.Fatalf("formatLokiResults failed: %v", err)
	}

	// Raw format should have timestamp, labels, and message
	if !strings.Contains(output, "2024-01-15T10:30:45Z") {
		t.Errorf("Expected output to contain timestamp '2024-01-15T10:30:45Z', but got:\n%s", output)
	}

	if !strings.Contains(output, "Test log message") {
		t.Errorf("Expected output to contain 'Test log message', but got:\n%s", output)
	}
}

// TestFormatLokiResults_JSONFormat tests the JSON output format
func TestFormatLokiResults_JSONFormat(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)

	result := &loghttp.QueryResponse{
		Status: "success",
		Data: loghttp.QueryResponseData{
			ResultType: loghttp.ResultTypeStream,
			Result: loghttp.Streams{
				{
					Labels: loghttp.LabelSet{
						"job": "test-job",
					},
					Entries: []loghttp.Entry{
						{Timestamp: timestamp, Line: "Test log message"},
					},
				},
			},
		},
	}

	output, err := formatLokiResults(result, "json")
	if err != nil {
		t.Fatalf("formatLokiResults failed: %v", err)
	}

	// JSON format should be valid JSON with status
	if !strings.Contains(output, `"status"`) {
		t.Errorf("Expected output to contain '\"status\"', but got:\n%s", output)
	}
}

// TestFormatLokiLabelsResults tests the labels formatting
func TestFormatLokiLabelsResults(t *testing.T) {
	result := &loghttp.LabelResponse{
		Status: "success",
		Data:   []string{"job", "level", "namespace"},
	}

	// Test raw format
	output, err := formatLokiLabelsResults(result, "raw")
	if err != nil {
		t.Fatalf("formatLokiLabelsResults failed: %v", err)
	}

	if !strings.Contains(output, "job\n") {
		t.Errorf("Expected output to contain 'job', but got:\n%s", output)
	}

	// Test text format
	output, err = formatLokiLabelsResults(result, "text")
	if err != nil {
		t.Fatalf("formatLokiLabelsResults failed: %v", err)
	}

	if !strings.Contains(output, "Found 3 labels") {
		t.Errorf("Expected output to contain 'Found 3 labels', but got:\n%s", output)
	}
}

// TestFormatLokiLabelValuesResults tests the label values formatting
func TestFormatLokiLabelValuesResults(t *testing.T) {
	result := &loghttp.LabelResponse{
		Status: "success",
		Data:   []string{"value1", "value2", "value3"},
	}

	// Test raw format
	output, err := formatLokiLabelValuesResults("testlabel", result, "raw")
	if err != nil {
		t.Fatalf("formatLokiLabelValuesResults failed: %v", err)
	}

	if !strings.Contains(output, "value1\n") {
		t.Errorf("Expected output to contain 'value1', but got:\n%s", output)
	}

	// Test text format
	output, err = formatLokiLabelValuesResults("testlabel", result, "text")
	if err != nil {
		t.Fatalf("formatLokiLabelValuesResults failed: %v", err)
	}

	if !strings.Contains(output, "Found 3 values for label 'testlabel'") {
		t.Errorf("Expected output to contain label count and name, but got:\n%s", output)
	}
}

// TestFilterEntries_CaseInsensitive tests case-insensitive filtering
func TestFilterEntries_CaseInsensitive(t *testing.T) {
	entries := []logEntry{
		{Timestamp: time.Now(), Line: "ERROR: something failed", Labels: loghttp.LabelSet{"job": "test"}},
		{Timestamp: time.Now(), Line: "info: everything is fine", Labels: loghttp.LabelSet{"job": "test"}},
		{Timestamp: time.Now(), Line: "Warning: potential issue", Labels: loghttp.LabelSet{"job": "test"}},
		{Timestamp: time.Now(), Line: "error in lowercase", Labels: loghttp.LabelSet{"job": "test"}},
	}

	// Case-insensitive search for "error"
	filtered := filterEntries(entries, "error", false)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 entries matching 'error' (case-insensitive), got %d", len(filtered))
	}

	// Verify both ERROR and error are matched
	for _, entry := range filtered {
		lowerLine := strings.ToLower(entry.Line)
		if !strings.Contains(lowerLine, "error") {
			t.Errorf("Entry '%s' should not have matched 'error'", entry.Line)
		}
	}
}

// TestFilterEntries_CaseSensitive tests case-sensitive filtering
func TestFilterEntries_CaseSensitive(t *testing.T) {
	entries := []logEntry{
		{Timestamp: time.Now(), Line: "ERROR: something failed", Labels: loghttp.LabelSet{"job": "test"}},
		{Timestamp: time.Now(), Line: "info: everything is fine", Labels: loghttp.LabelSet{"job": "test"}},
		{Timestamp: time.Now(), Line: "Warning: potential issue", Labels: loghttp.LabelSet{"job": "test"}},
		{Timestamp: time.Now(), Line: "error in lowercase", Labels: loghttp.LabelSet{"job": "test"}},
	}

	// Case-sensitive search for "ERROR"
	filtered := filterEntries(entries, "ERROR", true)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 entry matching 'ERROR' (case-sensitive), got %d", len(filtered))
	}

	if len(filtered) > 0 && !strings.Contains(filtered[0].Line, "ERROR") {
		t.Errorf("Expected entry to contain 'ERROR', got '%s'", filtered[0].Line)
	}
}

// TestApplyHeadTail_Head tests head functionality
func TestApplyHeadTail_Head(t *testing.T) {
	entries := []logEntry{
		{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Line: "first"},
		{Timestamp: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Line: "second"},
		{Timestamp: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Line: "third"},
		{Timestamp: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Line: "fourth"},
		{Timestamp: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Line: "fifth"},
	}

	result := applyHeadTail(entries, 3, 0)

	if len(result) != 3 {
		t.Errorf("Expected 3 entries with head=3, got %d", len(result))
	}

	expected := []string{"first", "second", "third"}
	for i, entry := range result {
		if entry.Line != expected[i] {
			t.Errorf("Entry %d: expected '%s', got '%s'", i, expected[i], entry.Line)
		}
	}
}

// TestApplyHeadTail_Tail tests tail functionality
func TestApplyHeadTail_Tail(t *testing.T) {
	entries := []logEntry{
		{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Line: "first"},
		{Timestamp: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Line: "second"},
		{Timestamp: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Line: "third"},
		{Timestamp: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Line: "fourth"},
		{Timestamp: time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), Line: "fifth"},
	}

	result := applyHeadTail(entries, 0, 2)

	if len(result) != 2 {
		t.Errorf("Expected 2 entries with tail=2, got %d", len(result))
	}

	expected := []string{"fourth", "fifth"}
	for i, entry := range result {
		if entry.Line != expected[i] {
			t.Errorf("Entry %d: expected '%s', got '%s'", i, expected[i], entry.Line)
		}
	}
}

// TestApplyHeadTail_LargerThanCount tests head/tail when value exceeds entry count
func TestApplyHeadTail_LargerThanCount(t *testing.T) {
	entries := []logEntry{
		{Timestamp: time.Now(), Line: "first"},
		{Timestamp: time.Now(), Line: "second"},
	}

	// Head larger than count
	result := applyHeadTail(entries, 10, 0)
	if len(result) != 2 {
		t.Errorf("Expected all 2 entries when head > count, got %d", len(result))
	}

	// Tail larger than count
	result = applyHeadTail(entries, 0, 10)
	if len(result) != 2 {
		t.Errorf("Expected all 2 entries when tail > count, got %d", len(result))
	}
}

// TestCollectAndSortEntries_MultipleStreams tests merging and sorting across streams
func TestCollectAndSortEntries_MultipleStreams(t *testing.T) {
	streams := loghttp.Streams{
		{
			Labels: loghttp.LabelSet{"job": "job1"},
			Entries: []loghttp.Entry{
				{Timestamp: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Line: "job1-third"},
				{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Line: "job1-first"},
			},
		},
		{
			Labels: loghttp.LabelSet{"job": "job2"},
			Entries: []loghttp.Entry{
				{Timestamp: time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), Line: "job2-fourth"},
				{Timestamp: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Line: "job2-second"},
			},
		},
	}

	entries := collectAndSortEntries(streams)

	if len(entries) != 4 {
		t.Fatalf("Expected 4 entries, got %d", len(entries))
	}

	// Verify entries are sorted by timestamp (oldest first)
	expected := []string{"job1-first", "job2-second", "job1-third", "job2-fourth"}
	for i, entry := range entries {
		if entry.Line != expected[i] {
			t.Errorf("Entry %d: expected '%s', got '%s'", i, expected[i], entry.Line)
		}
	}
}

// TestEntriesToStreams tests converting entries back to streams format
func TestEntriesToStreams(t *testing.T) {
	entries := []logEntry{
		{Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), Line: "first", Labels: loghttp.LabelSet{"job": "test"}},
		{Timestamp: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), Line: "second", Labels: loghttp.LabelSet{"job": "test"}},
		{Timestamp: time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), Line: "third", Labels: loghttp.LabelSet{"job": "other"}},
	}

	streams := entriesToStreams(entries)

	// Should have 2 streams (grouped by labels)
	if len(streams) != 2 {
		t.Errorf("Expected 2 streams, got %d", len(streams))
	}

	// Count total entries across streams
	totalEntries := 0
	for _, stream := range streams {
		totalEntries += len(stream.Entries)
	}
	if totalEntries != 3 {
		t.Errorf("Expected 3 total entries across streams, got %d", totalEntries)
	}
}

// TestEntriesToStreams_Empty tests converting empty entries
func TestEntriesToStreams_Empty(t *testing.T) {
	streams := entriesToStreams([]logEntry{})

	if len(streams) != 0 {
		t.Errorf("Expected 0 streams for empty entries, got %d", len(streams))
	}
}

// TestHandleLokiQuery_HeadTailMutuallyExclusive tests that head and tail cannot be used together
func TestHandleLokiQuery_HeadTailMutuallyExclusive(t *testing.T) {
	// Create a request with both head and tail set
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "loki_query",
			Arguments: map[string]any{
				"query": `{job="test"}`,
				"head":  float64(5),
				"tail":  float64(5),
			},
		},
	}

	_, err := HandleLokiQuery(context.Background(), request)

	if err == nil {
		t.Error("Expected error when both head and tail are specified")
	}

	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("Expected error about mutual exclusivity, got: %s", err.Error())
	}
}
