package event

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	// retentionDays is the number of days to keep raw events.
	retentionDays = 30
	// compactionBatchSize is the number of rows to delete per batch.
	compactionBatchSize = 1000
	allTimeRangeDays    = 0
	todayRangeDays      = 1
)

// Service provides event tracking business logic.
type Service struct {
	repo *Repository
	log  *slog.Logger
}

// NewService creates a new event service.
func NewService(repo *Repository, log *slog.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// RecordNotificationCreated records that a notification was created.
func (s *Service) RecordNotificationCreated(ctx context.Context, userID, topic, source, encryptionMode string, attachmentBytes *int64) {
	ev := &NotificationEvent{
		ID:              NewEventID(),
		UserID:          userID,
		EventType:       TypeNotificationCreated,
		Topic:           strPtr(topic),
		Source:          strPtr(source),
		EncryptionMode:  strPtr(encryptionMode),
		AttachmentBytes: attachmentBytes,
		CreatedAt:       time.Now().UTC().UnixMilli(),
	}

	if err := s.repo.InsertEvent(ctx, ev); err != nil {
		s.log.Error("failed to record notification_created event", "user_id", userID, "error", err)
	}
}

// RecordNotificationDelivered records a successful push delivery to a device.
func (s *Service) RecordNotificationDelivered(ctx context.Context, userID, deviceID string) {
	ev := &NotificationEvent{
		ID:        NewEventID(),
		UserID:    userID,
		EventType: TypeNotificationDelivered,
		DeviceID:  strPtr(deviceID),
		CreatedAt: time.Now().UTC().UnixMilli(),
	}

	if err := s.repo.InsertEvent(ctx, ev); err != nil {
		s.log.Error("failed to record notification_delivered event", "user_id", userID, "error", err)
	}
}

// RecordNotificationFailed records a failed push delivery to a device.
func (s *Service) RecordNotificationFailed(ctx context.Context, userID, deviceID, failReason string) {
	ev := &NotificationEvent{
		ID:         NewEventID(),
		UserID:     userID,
		EventType:  TypeNotificationFailed,
		DeviceID:   strPtr(deviceID),
		FailReason: strPtr(failReason),
		CreatedAt:  time.Now().UTC().UnixMilli(),
	}

	if err := s.repo.InsertEvent(ctx, ev); err != nil {
		s.log.Error("failed to record notification_failed event", "user_id", userID, "error", err)
	}
}

// GetAccountUsage retrieves daily usage for a user over the last N days,
// with zero-filled gaps so every day in the range is represented.
func (s *Service) GetAccountUsage(ctx context.Context, userID string, days int) (*AccountUsageResponse, error) {
	fromMs, toMs := resolveDashboardRange(time.Now().UTC(), days)

	summaries, err := s.repo.GetDailySummaries(ctx, userID, fromMs, toMs)
	if err != nil {
		return nil, fmt.Errorf("get account usage: %w", err)
	}

	if len(summaries) == 0 {
		return &AccountUsageResponse{Data: []AccountUsageDayResponse{}}, nil
	}

	if days == allTimeRangeDays {
		fromMs = summaries[0].DayStartMs
	}

	resp := ToAccountUsageResponse(summaries, fromMs, toMs)
	return &resp, nil
}

// CompactOldEvents deletes raw events older than the retention period.
func (s *Service) CompactOldEvents(ctx context.Context) (int64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays).UnixMilli()

	deleted, err := s.repo.DeleteEventsOlderThan(ctx, cutoff, compactionBatchSize)
	if err != nil {
		s.log.Error("event compaction failed", "error", err, "deleted_so_far", deleted)
		return deleted, err
	}

	if deleted > 0 {
		s.log.Info("event compaction completed", "deleted", deleted)
	}

	return deleted, nil
}

// GetPlatformDashboard retrieves platform-wide dashboard data for the last N days.
func (s *Service) GetPlatformDashboard(ctx context.Context, days int) (*PlatformDashboardResponse, error) {
	fromMs, toMs := resolveDashboardRange(time.Now().UTC(), days)

	summary, err := s.repo.GetPlatformSummary(ctx, fromMs, toMs)
	if err != nil {
		return nil, fmt.Errorf("get platform dashboard: %w", err)
	}

	dailyRows, err := s.repo.GetPlatformDailyBreakdown(ctx, fromMs, toMs)
	if err != nil {
		return nil, fmt.Errorf("get platform daily breakdown: %w", err)
	}

	var deliverySuccessRate float64
	deliveryAttempts := summary.NotificationsDelivered + summary.NotificationsFailed
	if deliveryAttempts > 0 {
		deliverySuccessRate = float64(summary.NotificationsDelivered) / float64(deliveryAttempts)
	}

	dailyBreakdown := []DailyUsageSummaryResponse{}
	if len(dailyRows) > 0 {
		fillFromMs := fromMs
		if days == allTimeRangeDays {
			fillFromMs = dailyRows[0].DayStartMs
		}
		dailyBreakdown = FillDailyUsageSummaryResponses(dailyRows, fillFromMs, toMs)
	}

	return &PlatformDashboardResponse{
		NotificationsCreated:        summary.NotificationsTotal,
		DeliveryAttempts:            deliveryAttempts,
		DeliveriesSucceeded:         summary.NotificationsDelivered,
		DeliveriesFailed:            summary.NotificationsFailed,
		DeliverySuccessRate:         deliverySuccessRate,
		ActiveUsers:                 summary.ActiveUsers,
		NotificationsServerTrusted:  summary.NotificationsServerTrusted,
		NotificationsE2E:            summary.NotificationsE2E,
		NotificationsWithAttachment: summary.AttachmentsCount,
		AttachmentBytesTotal:        summary.AttachmentsBytesTotal,
		SourcesCLI:                  summary.SourcesCLI,
		SourcesWebhook:              summary.SourcesWebhook,
		SourcesAPI:                  summary.SourcesAPI,
		DevicesLost:                 summary.DevicesLost,
		DailyBreakdown:              dailyBreakdown,
	}, nil
}

// resolveDashboardRange converts a range selector to an inclusive UTC day window.
func resolveDashboardRange(now time.Time, days int) (int64, int64) {
	todayStartMs := DayStartMs(now)

	if days == allTimeRangeDays {
		return 0, todayStartMs
	}

	if days == todayRangeDays {
		return todayStartMs, todayStartMs
	}

	toMs := todayStartMs
	fromMs := toMs - int64(days-1)*dayMillis
	return fromMs, toMs
}

// strPtr returns a pointer to s, or nil if s is empty.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
