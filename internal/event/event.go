// Package event tracks notification events and daily usage summaries.
package event

import "time"

const dayMillis = 86_400_000

// Event type constants.
const (
	TypeNotificationCreated   = "notification_created"
	TypeNotificationDelivered = "notification_delivered"
	TypeNotificationFailed    = "notification_failed"
)

// Source constants.
const (
	SourceCLI      = "cli"
	SourceWebhook  = "webhook"
	SourceAPI      = "api"
	SourceInternal = "internal"
)

// Encryption mode constants.
const (
	EncryptionServerTrusted = "server_trusted"
	EncryptionE2E           = "e2e"
)

// Fail reason constants for stable analytics codes.
const (
	FailSubscriptionGone = "subscription_gone"
	FailRateLimited      = "rate_limited"
	FailServerError      = "server_error"
	FailUnknown          = "unknown"
)

// NotificationEvent is the DB struct for the notification_events table.
type NotificationEvent struct {
	ID              string  `db:"id"`
	UserID          string  `db:"user_id"`
	EventType       string  `db:"event_type"`
	DeviceID        *string `db:"device_id"`
	Topic           *string `db:"topic"`
	Source          *string `db:"source"`
	EncryptionMode  *string `db:"encryption_mode"`
	AttachmentBytes *int64  `db:"attachment_bytes"`
	FailReason      *string `db:"fail_reason"`
	CreatedAt       int64   `db:"created_at"`
}

// DailyUsageSummary is the DB struct for the daily_usage_summary table.
type DailyUsageSummary struct {
	UserID                     string `db:"user_id"`
	DayStartMs                 int64  `db:"day_start_ms"`
	NotificationsTotal         int    `db:"notifications_total"`
	NotificationsServerTrusted int    `db:"notifications_server_trusted"`
	NotificationsE2E           int    `db:"notifications_e2e"`
	NotificationsDelivered     int    `db:"notifications_delivered"`
	NotificationsFailed        int    `db:"notifications_failed"`
	AttachmentsCount           int    `db:"attachments_count"`
	AttachmentsBytesTotal      int64  `db:"attachments_bytes_total"`
	SourcesCLI                 int    `db:"sources_cli"`
	SourcesWebhook             int    `db:"sources_webhook"`
	SourcesAPI                 int    `db:"sources_api"`
	SourcesInternal            int    `db:"sources_internal"`
	DevicesLost                int    `db:"devices_lost"`
	CreatedAt                  int64  `db:"created_at"`
	UpdatedAt                  int64  `db:"updated_at"`
}

// DailyUsageSummaryResponse is the HTTP response struct for dashboard data.
type DailyUsageSummaryResponse struct {
	Date                        string    `json:"date"`
	NotificationsCreated        int       `json:"notifications_created"`
	DeliveryAttempts            int       `json:"delivery_attempts"`
	DeliveriesSucceeded         int       `json:"deliveries_succeeded"`
	DeliveriesFailed            int       `json:"deliveries_failed"`
	NotificationsWithAttachment int       `json:"notifications_with_attachment"`
	AttachmentBytesTotal        int64     `json:"attachment_bytes_total"`
	NotificationsServerTrusted  int       `json:"notifications_server_trusted"`
	NotificationsE2E            int       `json:"notifications_e2e"`
	SourcesCLI                  int       `json:"sources_cli"`
	SourcesWebhook              int       `json:"sources_webhook"`
	SourcesAPI                  int       `json:"sources_api"`
	SourcesInternal             int       `json:"sources_internal"`
	DevicesLost                 int       `json:"devices_lost"`
	UpdatedAt                   time.Time `json:"updated_at"`
}

// ToDailyUsageSummaryResponse converts a DailyUsageSummary DB struct to a response.
func ToDailyUsageSummaryResponse(s *DailyUsageSummary) DailyUsageSummaryResponse {
	return DailyUsageSummaryResponse{
		Date:                        time.UnixMilli(s.DayStartMs).UTC().Format("2006-01-02"),
		NotificationsCreated:        s.NotificationsTotal,
		DeliveryAttempts:            s.NotificationsDelivered + s.NotificationsFailed,
		DeliveriesSucceeded:         s.NotificationsDelivered,
		DeliveriesFailed:            s.NotificationsFailed,
		NotificationsWithAttachment: s.AttachmentsCount,
		AttachmentBytesTotal:        s.AttachmentsBytesTotal,
		NotificationsServerTrusted:  s.NotificationsServerTrusted,
		NotificationsE2E:            s.NotificationsE2E,
		SourcesCLI:                  s.SourcesCLI,
		SourcesWebhook:              s.SourcesWebhook,
		SourcesAPI:                  s.SourcesAPI,
		SourcesInternal:             s.SourcesInternal,
		DevicesLost:                 s.DevicesLost,
		UpdatedAt:                   time.UnixMilli(s.UpdatedAt).UTC(),
	}
}

// FillDailyUsageSummaryResponses fills missing days with zero-value responses.
func FillDailyUsageSummaryResponses(summaries []DailyUsageSummary, fromMs, toMs int64) []DailyUsageSummaryResponse {
	if fromMs > toMs {
		return []DailyUsageSummaryResponse{}
	}

	byDay := make(map[int64]*DailyUsageSummary, len(summaries))
	for i := range summaries {
		byDay[summaries[i].DayStartMs] = &summaries[i]
	}

	days := make([]DailyUsageSummaryResponse, 0, ((toMs-fromMs)/dayMillis)+1)
	for ms := fromMs; ms <= toMs; ms += dayMillis {
		if summary, ok := byDay[ms]; ok {
			days = append(days, ToDailyUsageSummaryResponse(summary))
			continue
		}

		days = append(days, DailyUsageSummaryResponse{
			Date: time.UnixMilli(ms).UTC().Format("2006-01-02"),
		})
	}

	return days
}

// DayStartMs returns the UnixMilli timestamp for the start of the UTC day containing t.
func DayStartMs(t time.Time) int64 {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).UnixMilli()
}

// PlatformDashboardResponse is the HTTP response for the admin dashboard.
type PlatformDashboardResponse struct {
	NotificationsCreated        int                         `json:"notifications_created"`
	DeliveryAttempts            int                         `json:"delivery_attempts"`
	DeliveriesSucceeded         int                         `json:"deliveries_succeeded"`
	DeliveriesFailed            int                         `json:"deliveries_failed"`
	DeliverySuccessRate         float64                     `json:"delivery_success_rate"`
	ActiveUsers                 int                         `json:"active_users"`
	NotificationsServerTrusted  int                         `json:"notifications_server_trusted"`
	NotificationsE2E            int                         `json:"notifications_e2e"`
	NotificationsWithAttachment int                         `json:"notifications_with_attachment"`
	AttachmentBytesTotal        int64                       `json:"attachment_bytes_total"`
	SourcesCLI                  int                         `json:"sources_cli"`
	SourcesWebhook              int                         `json:"sources_webhook"`
	SourcesAPI                  int                         `json:"sources_api"`
	SourcesInternal             int                         `json:"sources_internal"`
	DevicesLost                 int                         `json:"devices_lost"`
	DailyBreakdown              []DailyUsageSummaryResponse `json:"daily_breakdown"`
}

// AccountUsageDayResponse is a single day's usage for the account dashboard.
type AccountUsageDayResponse struct {
	Date                        string `json:"date"`
	NotificationsCreated        int    `json:"notifications_created"`
	DeliveryAttempts            int    `json:"delivery_attempts"`
	DeliveriesSucceeded         int    `json:"deliveries_succeeded"`
	DeliveriesFailed            int    `json:"deliveries_failed"`
	DevicesLost                 int    `json:"devices_lost"`
	NotificationsWithAttachment int    `json:"notifications_with_attachment"`
	AttachmentBytesTotal        int64  `json:"attachment_bytes_total"`
	NotificationsServerTrusted  int    `json:"notifications_server_trusted"`
	NotificationsE2E            int    `json:"notifications_e2e"`
	SourcesCLI                  int    `json:"sources_cli"`
	SourcesWebhook              int    `json:"sources_webhook"`
	SourcesAPI                  int    `json:"sources_api"`
	SourcesInternal             int    `json:"sources_internal"`
}

// AccountUsageResponse is the HTTP response for the user account dashboard.
type AccountUsageResponse struct {
	Data []AccountUsageDayResponse `json:"data"`
}

// ToAccountUsageDayResponse converts a DailyUsageSummary to an account-level response.
func ToAccountUsageDayResponse(s *DailyUsageSummary) AccountUsageDayResponse {
	return AccountUsageDayResponse{
		Date:                        time.UnixMilli(s.DayStartMs).UTC().Format("2006-01-02"),
		NotificationsCreated:        s.NotificationsTotal,
		DeliveryAttempts:            s.NotificationsDelivered + s.NotificationsFailed,
		DeliveriesSucceeded:         s.NotificationsDelivered,
		DeliveriesFailed:            s.NotificationsFailed,
		DevicesLost:                 s.DevicesLost,
		NotificationsWithAttachment: s.AttachmentsCount,
		AttachmentBytesTotal:        s.AttachmentsBytesTotal,
		NotificationsServerTrusted:  s.NotificationsServerTrusted,
		NotificationsE2E:            s.NotificationsE2E,
		SourcesCLI:                  s.SourcesCLI,
		SourcesWebhook:              s.SourcesWebhook,
		SourcesAPI:                  s.SourcesAPI,
		SourcesInternal:             s.SourcesInternal,
	}
}

// ToAccountUsageResponse converts a slice of summaries to an account usage response,
// filling gaps with zero-value days so every day in [fromMs, toMs] is represented.
func ToAccountUsageResponse(summaries []DailyUsageSummary, fromMs, toMs int64) AccountUsageResponse {
	// Index existing data by day_start_ms for O(1) lookup.
	byDay := make(map[int64]*DailyUsageSummary, len(summaries))
	for i := range summaries {
		byDay[summaries[i].DayStartMs] = &summaries[i]
	}

	// Walk every day in the range and emit a response entry.
	var days []AccountUsageDayResponse
	for ms := fromMs; ms <= toMs; ms += dayMillis {
		if s, ok := byDay[ms]; ok {
			days = append(days, ToAccountUsageDayResponse(s))
		} else {
			days = append(days, AccountUsageDayResponse{
				Date:                        time.UnixMilli(ms).UTC().Format("2006-01-02"),
				NotificationsCreated:        0,
				DeliveryAttempts:            0,
				DeliveriesSucceeded:         0,
				DeliveriesFailed:            0,
				DevicesLost:                 0,
				NotificationsWithAttachment: 0,
				AttachmentBytesTotal:        0,
				NotificationsServerTrusted:  0,
				NotificationsE2E:            0,
				SourcesCLI:                  0,
				SourcesWebhook:              0,
				SourcesAPI:                  0,
				SourcesInternal:             0,
			})
		}
	}

	return AccountUsageResponse{Data: days}
}
