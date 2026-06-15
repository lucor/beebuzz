// Package debugreport handles Hive PWA diagnostic report submissions.
package debugreport

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// MaxPayloadSize limits debug report body size (256KB).
	MaxPayloadSize = 256 * 1024

	// SchemaVersion is the current debug report schema version.
	SchemaVersion = 1
	// ExpectedSource is the required source value for Hive reports.
	ExpectedSource = "hive"
	// ExpectedReportType is the required report_type value.
	ExpectedReportType = "manual_error_report"

	MaxIDLength                  = 64
	MaxTimestampSkewFuture       = 5 * time.Minute
	MaxMessageLength             = 240
	MaxErrorNameLength           = 80
	MaxErrorCodeLength           = 64
	MaxStackFrames               = 8
	MaxStackFrameLength          = 180
	MaxRelatedLogs               = 20
	MaxRouteLength               = 120
	MaxBuildIDLength             = 80
	MaxAppVersionLength          = 40
	MaxBrowserVersionMajorLength = 8
	MaxLogDataErrorCodeLength    = 64
	MaxDurationMS                = 10 * 60 * 1000
)

// Sentinel errors.
var (
	ErrInvalidReport  = fmt.Errorf("invalid debug report")
	ErrReportTooLarge = fmt.Errorf("report payload too large")
)

// When saved to the DB, Payload is the validated JSON string.
type DebugReport struct {
	ReportID    string `db:"report_id"`
	DeviceID    string `db:"device_id"`
	CreatedAt   int64  `db:"created_at"`
	PayloadJSON string `db:"payload_json"`
}

// SubmitRequest is the expected JSON body for POST /v1/hive/debug-reports.
type SubmitRequest struct {
	SchemaVersion int           `json:"schema_version"`
	Source        string        `json:"source"`
	ReportType    string        `json:"report_type"`
	CreatedAt     string        `json:"created_at"`
	Snapshot      DebugSnapshot `json:"snapshot"`
}

// DebugSnapshot is the diagnostic snapshot payload.
type DebugSnapshot struct {
	ID          string       `json:"id"`
	Timestamp   string       `json:"ts"`
	Scope       string       `json:"scope"`
	Event       string       `json:"event"`
	Severity    string       `json:"severity"`
	Message     string       `json:"message"`
	Error       *DebugError  `json:"error"`
	Context     DebugContext `json:"context"`
	RelatedLogs []DebugLog   `json:"related_logs"`
}

// DebugError describes an error attached to a snapshot.
type DebugError struct {
	Name  string   `json:"name"`
	Code  *string  `json:"code"`
	Stack []string `json:"stack"`
}

// DebugContext captures the PWA environment at snapshot time.
type DebugContext struct {
	AppVersion                  string  `json:"app_version"`
	BuildID                     string  `json:"build_id"`
	BrowserFamily               string  `json:"browser_family"`
	BrowserVersionMajor         string  `json:"browser_version_major"`
	OS                          string  `json:"os"`
	DisplayMode                 string  `json:"display_mode"`
	NotificationPermission      string  `json:"notification_permission"`
	ServiceWorkerSupported      *bool   `json:"service_worker_supported"`
	ServiceWorkerState          string  `json:"service_worker_state"`
	PushSupported               *bool   `json:"push_supported"`
	PushPresent                 *bool   `json:"push_present"`
	WebCryptoSupported          *bool   `json:"webcrypto_supported"`
	X25519Supported             *bool   `json:"x25519_supported"`
	IndexedDBAvailable          *bool   `json:"indexeddb_available"`
	NetworkOnline               *bool   `json:"network_online"`
	LastPushReceivedAt          *string `json:"last_push_received_at"`
	LastNotificationDisplayedAt *string `json:"last_notification_displayed_at"`
}

// DebugLog is a single diagnostic log entry.
type DebugLog struct {
	ID        string        `json:"id"`
	Timestamp string        `json:"ts"`
	Kind      string        `json:"kind"`
	Scope     string        `json:"scope"`
	Event     string        `json:"event"`
	Message   string        `json:"message"`
	Data      *DebugLogData `json:"data,omitempty"`
}

// DebugLogData carries optional structured data for a log entry.
type DebugLogData struct {
	Status     *int    `json:"status,omitempty"`
	DurationMS *int    `json:"duration_ms,omitempty"`
	Route      *string `json:"route,omitempty"`
	Method     *string `json:"method,omitempty"`
	ErrorCode  *string `json:"error_code,omitempty"`
	OK         *bool   `json:"ok,omitempty"`
}

// SubmitResponse is the success response.
type SubmitResponse struct {
	ReportID  string `json:"report_id"`
	CreatedAt string `json:"created_at"`
}

// Allowed enums.
var (
	allowedScopes = map[string]bool{
		"app": true, "push": true, "payload": true, "notification": true,
		"service_worker": true, "storage": true, "pairing": true,
		"network": true, "encryption": true,
	}
	allowedKinds = map[string]bool{
		"main": true, "developer": true,
	}
	allowedSeverities = map[string]bool{
		"warn": true, "error": true,
	}
	allowedBrowserFamilies = map[string]bool{
		"unknown": true, "Chrome": true, "Safari": true, "Firefox": true, "Edge": true,
	}
	allowedOS = map[string]bool{
		"unknown": true, "Windows": true, "macOS": true, "Linux": true, "Android": true, "iOS": true,
	}
	allowedDisplayModes = map[string]bool{
		"browser": true, "standalone": true, "minimal-ui": true, "fullscreen": true,
	}
	allowedNotificationPermissions = map[string]bool{
		"not-supported": true, "default": true, "denied": true, "granted": true,
	}
	allowedServiceWorkerStates = map[string]bool{
		"not-supported": true, "active": true, "waiting": true, "installing": true,
		"no-registration": true, "error": true,
	}
	allowedMethods = map[string]bool{
		"GET": true, "POST": true, "PUT": true, "PATCH": true, "DELETE": true,
	}
	allowedEvents = map[string]bool{
		"app.started": true, "app.bootstrap_failed": true,
		"service_worker.registered": true, "service_worker.activated": true,
		"service_worker.skip_waiting": true,
		"pairing.reconnect_required":  true,
		"push.received":               true, "push.empty_payload": true, "push.resolved": true,
		"push.subscription_changed": true,
		"payload.resolve":           true, "payload.detected_encrypted": true,
		"payload.detected_plain": true, "payload.decrypt_failed": true,
		"payload.invalid":              true,
		"storage.credentials_failed":   true,
		"notification.persist_started": true, "notification.persist_failed": true,
		"notification.displayed": true,
		"clients.match_failed":   true, "clients.focus_failed": true,
		"clients.open_window_failed": true, "clients.post_message_failed": true,
		"debug_report.missing_device_token": true, "debug_report.submit_failed": true,
	}
)

var errorCodeRegex = regexp.MustCompile(`^[A-Za-z0-9_:\-]{1,64}$`)

// Validate checks the submit request against business rules.
func (r *SubmitRequest) Validate(now time.Time) []error {
	var errs []error

	if r.SchemaVersion != SchemaVersion {
		errs = append(errs, fmt.Errorf("unsupported schema_version: %d", r.SchemaVersion))
	}
	if r.Source != ExpectedSource {
		errs = append(errs, fmt.Errorf("invalid source: %s", r.Source))
	}
	if r.ReportType != ExpectedReportType {
		errs = append(errs, fmt.Errorf("invalid report_type: %s", r.ReportType))
	}

	// validate created_at
	if r.CreatedAt == "" {
		errs = append(errs, fmt.Errorf("created_at is required"))
	} else {
		ts, err := parseRFC3339(r.CreatedAt)
		if err != nil {
			errs = append(errs, fmt.Errorf("created_at must be a valid RFC3339 timestamp"))
		} else if ts.After(now.Add(MaxTimestampSkewFuture)) {
			errs = append(errs, fmt.Errorf("created_at must not be more than 5 minutes in the future"))
		}
	}

	// validate snapshot
	errs = append(errs, validateSnapshot(&r.Snapshot, now)...)

	return errs
}

func validateSnapshot(s *DebugSnapshot, now time.Time) []error {
	var errs []error

	if s.ID == "" {
		errs = append(errs, fmt.Errorf("snapshot.id is required"))
	} else if len(s.ID) > MaxIDLength {
		errs = append(errs, fmt.Errorf("snapshot.id exceeds %d characters", MaxIDLength))
	}

	if s.Timestamp == "" {
		errs = append(errs, fmt.Errorf("snapshot.ts is required"))
	} else {
		ts, err := parseRFC3339(s.Timestamp)
		if err != nil {
			errs = append(errs, fmt.Errorf("snapshot.ts must be a valid RFC3339 timestamp"))
		} else if ts.After(now.Add(MaxTimestampSkewFuture)) {
			errs = append(errs, fmt.Errorf("snapshot.ts must not be more than 5 minutes in the future"))
		}
	}

	if s.Scope == "" {
		errs = append(errs, fmt.Errorf("snapshot.scope is required"))
	} else if !allowedScopes[s.Scope] {
		errs = append(errs, fmt.Errorf("unknown snapshot.scope: %s", s.Scope))
	}

	if s.Event == "" {
		errs = append(errs, fmt.Errorf("snapshot.event is required"))
	} else if !allowedEvents[s.Event] {
		errs = append(errs, fmt.Errorf("unknown snapshot.event: %s", s.Event))
	}

	if s.Severity == "" {
		errs = append(errs, fmt.Errorf("snapshot.severity is required"))
	} else if !allowedSeverities[s.Severity] {
		errs = append(errs, fmt.Errorf("unknown snapshot.severity: %s", s.Severity))
	}

	if s.Message == "" {
		errs = append(errs, fmt.Errorf("snapshot.message is required"))
	} else if len(s.Message) > MaxMessageLength {
		errs = append(errs, fmt.Errorf("snapshot.message exceeds %d characters", MaxMessageLength))
	}

	if s.Error != nil {
		errs = append(errs, validateError(s.Error)...)
	}

	errs = append(errs, validateContext(&s.Context)...)

	if s.RelatedLogs == nil {
		errs = append(errs, fmt.Errorf("snapshot.related_logs is required"))
	} else if len(s.RelatedLogs) > MaxRelatedLogs {
		errs = append(errs, fmt.Errorf("snapshot.related_logs exceeds %d entries", MaxRelatedLogs))
	}
	for i := range s.RelatedLogs {
		errs = append(errs, validateLog(&s.RelatedLogs[i], now)...)
	}

	return errs
}

func validateError(e *DebugError) []error {
	var errs []error
	if e.Name != "" && len(e.Name) > MaxErrorNameLength {
		errs = append(errs, fmt.Errorf("error.name exceeds %d characters", MaxErrorNameLength))
	}
	if e.Code != nil {
		if !errorCodeRegex.MatchString(*e.Code) {
			errs = append(errs, fmt.Errorf("error.code must match ^[A-Za-z0-9_:\\-]{1,64}$"))
		}
	}
	if len(e.Stack) > MaxStackFrames {
		errs = append(errs, fmt.Errorf("error.stack exceeds %d frames", MaxStackFrames))
	}
	for i, frame := range e.Stack {
		if len(frame) > MaxStackFrameLength {
			errs = append(errs, fmt.Errorf("error.stack[%d] exceeds %d characters", i, MaxStackFrameLength))
		}
	}
	return errs
}

func validateContext(c *DebugContext) []error {
	var errs []error

	if c.AppVersion == "" {
		errs = append(errs, fmt.Errorf("context.app_version is required"))
	} else if len(c.AppVersion) > MaxAppVersionLength {
		errs = append(errs, fmt.Errorf("context.app_version exceeds %d characters", MaxAppVersionLength))
	}
	if len(c.BuildID) > MaxBuildIDLength {
		errs = append(errs, fmt.Errorf("context.build_id exceeds %d characters", MaxBuildIDLength))
	}
	if !allowedBrowserFamilies[c.BrowserFamily] {
		errs = append(errs, fmt.Errorf("unknown context.browser_family: %s", c.BrowserFamily))
	}
	if c.BrowserVersionMajor == "" {
		errs = append(errs, fmt.Errorf("context.browser_version_major is required"))
	} else if len(c.BrowserVersionMajor) > MaxBrowserVersionMajorLength {
		errs = append(errs, fmt.Errorf("context.browser_version_major exceeds %d characters", MaxBrowserVersionMajorLength))
	}
	if !allowedOS[c.OS] {
		errs = append(errs, fmt.Errorf("unknown context.os: %s", c.OS))
	}
	if !allowedDisplayModes[c.DisplayMode] {
		errs = append(errs, fmt.Errorf("unknown context.display_mode: %s", c.DisplayMode))
	}
	if !allowedNotificationPermissions[c.NotificationPermission] {
		errs = append(errs, fmt.Errorf("unknown context.notification_permission: %s", c.NotificationPermission))
	}
	if c.ServiceWorkerSupported == nil {
		errs = append(errs, fmt.Errorf("context.service_worker_supported is required"))
	}
	if !allowedServiceWorkerStates[c.ServiceWorkerState] {
		errs = append(errs, fmt.Errorf("unknown context.service_worker_state: %s", c.ServiceWorkerState))
	}
	if c.PushSupported == nil {
		errs = append(errs, fmt.Errorf("context.push_supported is required"))
	}
	if c.PushPresent == nil {
		errs = append(errs, fmt.Errorf("context.push_present is required"))
	}
	if c.WebCryptoSupported == nil {
		errs = append(errs, fmt.Errorf("context.webcrypto_supported is required"))
	}
	if c.X25519Supported == nil {
		errs = append(errs, fmt.Errorf("context.x25519_supported is required"))
	}
	if c.IndexedDBAvailable == nil {
		errs = append(errs, fmt.Errorf("context.indexeddb_available is required"))
	}
	if c.NetworkOnline == nil {
		errs = append(errs, fmt.Errorf("context.network_online is required"))
	}

	if c.LastPushReceivedAt != nil {
		if _, err := parseRFC3339(*c.LastPushReceivedAt); err != nil {
			errs = append(errs, fmt.Errorf("context.last_push_received_at must be a valid RFC3339 timestamp"))
		}
	}
	if c.LastNotificationDisplayedAt != nil {
		if _, err := parseRFC3339(*c.LastNotificationDisplayedAt); err != nil {
			errs = append(errs, fmt.Errorf("context.last_notification_displayed_at must be a valid RFC3339 timestamp"))
		}
	}

	return errs
}

func validateLog(l *DebugLog, now time.Time) []error {
	var errs []error

	if l.ID == "" {
		errs = append(errs, fmt.Errorf("related_logs[].id is required"))
	} else if len(l.ID) > MaxIDLength {
		errs = append(errs, fmt.Errorf("related_logs[].id exceeds %d characters", MaxIDLength))
	}
	if l.Timestamp == "" {
		errs = append(errs, fmt.Errorf("related_logs[].ts is required"))
	} else if _, err := parseRFC3339(l.Timestamp); err != nil {
		errs = append(errs, fmt.Errorf("related_logs[].ts must be a valid RFC3339 timestamp"))
	}
	if l.Kind == "" {
		errs = append(errs, fmt.Errorf("related_logs[].kind is required"))
	} else if !allowedKinds[l.Kind] {
		errs = append(errs, fmt.Errorf("unknown related_logs[].kind: %s", l.Kind))
	}
	if l.Scope == "" {
		errs = append(errs, fmt.Errorf("related_logs[].scope is required"))
	} else if !allowedScopes[l.Scope] {
		errs = append(errs, fmt.Errorf("unknown related_logs[].scope: %s", l.Scope))
	}
	if l.Event == "" {
		errs = append(errs, fmt.Errorf("related_logs[].event is required"))
	} else if !allowedEvents[l.Event] {
		errs = append(errs, fmt.Errorf("unknown related_logs[].event: %s", l.Event))
	}
	if l.Message == "" {
		errs = append(errs, fmt.Errorf("related_logs[].message is required"))
	} else if len(l.Message) > MaxMessageLength {
		errs = append(errs, fmt.Errorf("related_logs[].message exceeds %d characters", MaxMessageLength))
	}

	if l.Data != nil {
		errs = append(errs, validateLogData(l.Data)...)
	}

	return errs
}

func validateLogData(d *DebugLogData) []error {
	var errs []error

	if d.Status != nil {
		if *d.Status < 100 || *d.Status > 599 {
			errs = append(errs, fmt.Errorf("related_logs[].data.status must be 100..599"))
		}
	}
	if d.DurationMS != nil {
		if *d.DurationMS < 0 || *d.DurationMS > MaxDurationMS {
			errs = append(errs, fmt.Errorf("related_logs[].data.duration_ms must be 0..%d", MaxDurationMS))
		}
	}
	if d.Route != nil {
		route := *d.Route
		if !strings.HasPrefix(route, "/") {
			errs = append(errs, fmt.Errorf("related_logs[].data.route must start with /"))
		}
		if strings.Contains(route, "?") {
			errs = append(errs, fmt.Errorf("related_logs[].data.route must not contain ?"))
		}
		if strings.Contains(route, "#") {
			errs = append(errs, fmt.Errorf("related_logs[].data.route must not contain #"))
		}
		if len(route) > MaxRouteLength {
			errs = append(errs, fmt.Errorf("related_logs[].data.route exceeds %d characters", MaxRouteLength))
		}
	}
	if d.ErrorCode != nil {
		if len(*d.ErrorCode) > MaxLogDataErrorCodeLength {
			errs = append(errs, fmt.Errorf("related_logs[].data.error_code exceeds %d characters", MaxLogDataErrorCodeLength))
		}
		if !errorCodeRegex.MatchString(*d.ErrorCode) {
			errs = append(errs, fmt.Errorf("related_logs[].data.error_code must match ^[A-Za-z0-9_:\\-]{1,64}$"))
		}
	}
	if d.Method != nil && !allowedMethods[*d.Method] {
		errs = append(errs, fmt.Errorf("unknown related_logs[].data.method: %s", *d.Method))
	}

	return errs
}

func parseRFC3339(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse(time.RFC3339Nano, s)
	if err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid RFC3339 timestamp")
}
