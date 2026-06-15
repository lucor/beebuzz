package debugreport

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func mustPtr[T any](v T) *T { return &v }

func validDebugContext() DebugContext {
	return DebugContext{
		AppVersion:                  "0.0.0",
		BuildID:                     "",
		BrowserFamily:               "unknown",
		BrowserVersionMajor:         "0",
		OS:                          "unknown",
		DisplayMode:                 "browser",
		NotificationPermission:      "not-supported",
		ServiceWorkerSupported:      mustPtr(false),
		ServiceWorkerState:          "not-supported",
		PushSupported:               mustPtr(false),
		PushPresent:                 mustPtr(false),
		WebCryptoSupported:          mustPtr(false),
		X25519Supported:             mustPtr(false),
		IndexedDBAvailable:          mustPtr(false),
		NetworkOnline:               mustPtr(true),
		LastPushReceivedAt:          nil,
		LastNotificationDisplayedAt: nil,
	}
}

func TestSubmitRequestValidateAcceptsMinimalValidReport(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:        "abc123",
			Timestamp: "2026-06-16T10:00:00Z",
			Scope:     "app",
			Event:     "app.bootstrap_failed",
			Severity:  "error",
			Message:   "Hive app bootstrap failed",
			Error: &DebugError{
				Name:  "Error",
				Code:  nil,
				Stack: []string{"at bootstrapApp"},
			},
			Context: DebugContext{
				AppVersion:                  "0.0.0",
				BuildID:                     "",
				BrowserFamily:               "Safari",
				BrowserVersionMajor:         "17",
				OS:                          "iOS",
				DisplayMode:                 "standalone",
				NotificationPermission:      "granted",
				ServiceWorkerSupported:      mustPtr(true),
				ServiceWorkerState:          "active",
				PushSupported:               mustPtr(true),
				PushPresent:                 mustPtr(true),
				WebCryptoSupported:          mustPtr(true),
				X25519Supported:             mustPtr(true),
				IndexedDBAvailable:          mustPtr(true),
				NetworkOnline:               mustPtr(true),
				LastPushReceivedAt:          nil,
				LastNotificationDisplayedAt: nil,
			},
			RelatedLogs: []DebugLog{},
		},
	}

	errs := req.Validate(now)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
}

func TestSubmitRequestValidateRejectsUnknownEvent(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:        "abc",
			Timestamp: "2026-06-16T10:00:00Z",
			Scope:     "app",
			Event:     "unknown.event",
			Severity:  "error",
			Message:   "test",
			Context:   validDebugContext(),
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown event")
	}
	if !strings.Contains(errs[0].Error(), "unknown snapshot.event") {
		t.Fatalf("expected unknown event error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsUnknownScope(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:        "abc",
			Timestamp: "2026-06-16T10:00:00Z",
			Scope:     "invalid",
			Event:     "app.started",
			Severity:  "error",
			Message:   "test",
			Context:   validDebugContext(),
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown scope")
	}
	if !strings.Contains(errs[0].Error(), "unknown snapshot.scope") {
		t.Fatalf("expected unknown scope error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsUnknownSeverity(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:        "abc",
			Timestamp: "2026-06-16T10:00:00Z",
			Scope:     "app",
			Event:     "app.started",
			Severity:  "critical",
			Message:   "test",
			Context:   validDebugContext(),
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown severity")
	}
	if !strings.Contains(errs[0].Error(), "unknown snapshot.severity") {
		t.Fatalf("expected unknown severity error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsTooManyRelatedLogs(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	logs := make([]DebugLog, 21)
	for i := range logs {
		logs[i] = DebugLog{
			ID:    fmt.Sprintf("log-%d", i),
			Kind:  "main",
			Scope: "app",
			Event: "app.started",
		}
	}
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:          "abc",
			Timestamp:   "2026-06-16T10:00:00Z",
			Scope:       "app",
			Event:       "app.started",
			Severity:    "error",
			Message:     "test",
			Context:     validDebugContext(),
			RelatedLogs: logs,
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for too many related logs")
	}
	if !strings.Contains(errs[0].Error(), "exceeds 20 entries") {
		t.Fatalf("expected too many logs error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsLongMessage(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:          "abc",
			Timestamp:   "2026-06-16T10:00:00Z",
			Scope:       "app",
			Event:       "app.started",
			Severity:    "error",
			Message:     strings.Repeat("x", 241),
			Context:     validDebugContext(),
			RelatedLogs: []DebugLog{},
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for long message")
	}
	if !strings.Contains(errs[0].Error(), "exceeds 240 characters") {
		t.Fatalf("expected message length error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsInvalidTimestamp(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "not-a-timestamp",
		Snapshot: DebugSnapshot{
			ID:          "abc",
			Timestamp:   "2026-06-16T10:00:00Z",
			Scope:       "app",
			Event:       "app.started",
			Severity:    "error",
			Message:     "test",
			Context:     validDebugContext(),
			RelatedLogs: []DebugLog{},
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid timestamp")
	}
	if !strings.Contains(errs[0].Error(), "valid RFC3339 timestamp") {
		t.Fatalf("expected RFC3339 error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsFutureTimestamp(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	future := "2026-06-16T10:10:01Z" // 10 minutes + 1s ahead
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     future,
		Snapshot: DebugSnapshot{
			ID:          "abc",
			Timestamp:   future,
			Scope:       "app",
			Event:       "app.started",
			Severity:    "error",
			Message:     "test",
			Context:     validDebugContext(),
			RelatedLogs: []DebugLog{},
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for future timestamp")
	}
	if !strings.Contains(errs[0].Error(), "5 minutes in the future") {
		t.Fatalf("expected future timestamp error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsInvalidRoute(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	badRoute := "no-leading-slash"
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:        "abc",
			Timestamp: "2026-06-16T10:00:00Z",
			Scope:     "app",
			Event:     "app.started",
			Severity:  "error",
			Message:   "test",
			Context:   validDebugContext(),
			RelatedLogs: []DebugLog{
				{
					ID:        "log1",
					Timestamp: "2026-06-16T10:00:00Z",
					Kind:      "main",
					Scope:     "app",
					Event:     "app.started",
					Message:   "test",
					Data: &DebugLogData{
						Route: &badRoute,
					},
				},
			},
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid route")
	}
	if !strings.Contains(errs[0].Error(), "must start with /") {
		t.Fatalf("expected route error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsInvalidStatus(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	badStatus := 600
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:        "abc",
			Timestamp: "2026-06-16T10:00:00Z",
			Scope:     "app",
			Event:     "app.started",
			Severity:  "error",
			Message:   "test",
			Context:   validDebugContext(),
			RelatedLogs: []DebugLog{
				{
					ID:        "log1",
					Timestamp: "2026-06-16T10:00:00Z",
					Kind:      "main",
					Scope:     "app",
					Event:     "app.started",
					Message:   "test",
					Data: &DebugLogData{
						Status: &badStatus,
					},
				},
			},
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid status")
	}
	if !strings.Contains(errs[0].Error(), "must be 100..599") {
		t.Fatalf("expected status error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsInvalidErrorCode(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	badCode := "invalid code!"
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:          "abc",
			Timestamp:   "2026-06-16T10:00:00Z",
			Scope:       "app",
			Event:       "app.started",
			Severity:    "error",
			Message:     "test",
			Context:     validDebugContext(),
			RelatedLogs: []DebugLog{},
			Error: &DebugError{
				Name: "Error",
				Code: &badCode,
			},
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for invalid error code")
	}
	if !strings.Contains(errs[0].Error(), "error.code must match") {
		t.Fatalf("expected error code regex error, got %v", errs[0])
	}
}

func TestSubmitRequestValidateRejectsMissingContextBoolean(t *testing.T) {
	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	context := validDebugContext()
	context.PushSupported = nil
	req := SubmitRequest{
		SchemaVersion: 1,
		Source:        "hive",
		ReportType:    "manual_error_report",
		CreatedAt:     "2026-06-16T10:00:00Z",
		Snapshot: DebugSnapshot{
			ID:          "abc",
			Timestamp:   "2026-06-16T10:00:00Z",
			Scope:       "app",
			Event:       "app.started",
			Severity:    "error",
			Message:     "test",
			Context:     context,
			RelatedLogs: []DebugLog{},
		},
	}

	errs := req.Validate(now)
	if len(errs) == 0 {
		t.Fatal("expected error for missing context boolean")
	}
	if !strings.Contains(errs[0].Error(), "context.push_supported is required") {
		t.Fatalf("expected missing push_supported error, got %v", errs[0])
	}
}

func TestSubmitRequestJSONRoundTrip(t *testing.T) {
	raw := `{
		"schema_version": 1,
		"source": "hive",
		"report_type": "manual_error_report",
		"created_at": "2026-06-16T10:00:00Z",
		"snapshot": {
			"id": "abc123",
			"ts": "2026-06-16T10:00:00Z",
			"scope": "app",
			"event": "app.bootstrap_failed",
			"severity": "error",
			"message": "test",
			"error": {
				"name": "Error",
				"code": null,
				"stack": ["at bootstrapApp"]
			},
			"context": {
				"app_version": "0.0.0",
				"build_id": "",
				"browser_family": "Safari",
				"browser_version_major": "17",
				"os": "iOS",
				"display_mode": "standalone",
				"notification_permission": "granted",
				"service_worker_supported": true,
				"service_worker_state": "active",
				"push_supported": true,
				"push_present": true,
				"webcrypto_supported": true,
				"x25519_supported": true,
				"indexeddb_available": true,
				"network_online": true,
				"last_push_received_at": null,
				"last_notification_displayed_at": null
			},
			"related_logs": []
		}
	}`

	var req SubmitRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	now := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)
	if errs := req.Validate(now); len(errs) > 0 {
		t.Fatalf("validation errors: %v", errs)
	}

	// Check canonical marshal round trip
	canonical, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var req2 SubmitRequest
	if err := json.Unmarshal(canonical, &req2); err != nil {
		t.Fatalf("second unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(req, req2) {
		t.Fatalf("round trip mismatch:\nwant %#v\ngot  %#v", req, req2)
	}
}
