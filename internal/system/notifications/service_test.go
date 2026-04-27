package notifications

import (
	"context"
	"log/slog"
	"testing"

	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

type testTopicProvider struct {
	svc *topic.Service
}

func (p *testTopicProvider) GetTopicByID(ctx context.Context, userID, topicID string) (*Topic, error) {
	t, err := p.svc.GetTopicByID(ctx, userID, topicID)
	if err != nil || t == nil {
		return nil, err
	}
	return &Topic{ID: t.ID, UserID: t.UserID, Name: t.Name}, nil
}

type testDeviceSubscriptions struct {
	withDevice map[string]bool // key: userID + "|" + topicName
}

func (c *testDeviceSubscriptions) HasActiveDeviceForTopic(_ context.Context, userID, topicName string) (bool, error) {
	return c.withDevice[userID+"|"+topicName], nil
}

func newTestService(t *testing.T) (*Service, *topic.Service, context.Context) {
	t.Helper()
	return newTestServiceWith(t, nil)
}

func newTestServiceWith(t *testing.T, subs DeviceSubscriptionChecker) (*Service, *topic.Service, context.Context) {
	t.Helper()

	db := testutil.NewDBWithUsers(t, "admin-1", "other-1")
	topicSvc := topic.NewService(topic.NewRepository(db), slog.Default())
	svc := NewService(
		NewRepository(db),
		&testTopicProvider{svc: topicSvc},
		nil,
		subs,
		slog.Default(),
	)

	return svc, topicSvc, context.Background()
}

func TestGetSettingsReturnsNilWhenUnset(t *testing.T) {
	svc, _, ctx := newTestService(t)

	settings, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if settings != nil {
		t.Fatalf("GetSettings() = %#v, want nil", settings)
	}
}

func TestUpdateSettingsStoresCurrentAdminAsRecipient(t *testing.T) {
	svc, topicSvc, ctx := newTestService(t)
	topicRow, err := topicSvc.CreateTopic(ctx, "admin-1", "ops", "Operational alerts")
	if err != nil {
		t.Fatalf("CreateTopic() error = %v", err)
	}

	settings, err := svc.UpdateSettings(ctx, "admin-1", UpdateSettingsRequest{
		Enabled:              true,
		TopicID:              topicRow.ID,
		SignupCreatedEnabled: true,
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	if settings.RecipientUserID != "admin-1" {
		t.Fatalf("RecipientUserID = %q, want admin-1", settings.RecipientUserID)
	}
	if settings.TopicID != topicRow.ID {
		t.Fatalf("TopicID = %q, want %q", settings.TopicID, topicRow.ID)
	}
	if !settings.Enabled || !settings.SignupCreatedEnabled {
		t.Fatalf("settings flags = enabled:%v signup:%v, want both true", settings.Enabled, settings.SignupCreatedEnabled)
	}
}

func TestUpdateSettingsRejectsTopicOwnedByAnotherUser(t *testing.T) {
	svc, topicSvc, ctx := newTestService(t)
	otherTopic, err := topicSvc.CreateTopic(ctx, "other-1", "ops", "Other alerts")
	if err != nil {
		t.Fatalf("CreateTopic() error = %v", err)
	}

	_, err = svc.UpdateSettings(ctx, "admin-1", UpdateSettingsRequest{
		Enabled:              true,
		TopicID:              otherTopic.ID,
		SignupCreatedEnabled: true,
	})
	if err != ErrInvalidTopicSelection {
		t.Fatalf("UpdateSettings() error = %v, want %v", err, ErrInvalidTopicSelection)
	}
}

func TestUpdateSettingsRequiresTopicWhenEnabled(t *testing.T) {
	svc, _, ctx := newTestService(t)

	_, err := svc.UpdateSettings(ctx, "admin-1", UpdateSettingsRequest{Enabled: true})
	if err != ErrTopicRequired {
		t.Fatalf("UpdateSettings() error = %v, want %v", err, ErrTopicRequired)
	}
}

func TestGetSettingsReportsRecipientDevicePresence(t *testing.T) {
	subs := &testDeviceSubscriptions{withDevice: map[string]bool{}}
	svc, topicSvc, ctx := newTestServiceWith(t, subs)
	topicRow, err := topicSvc.CreateTopic(ctx, "admin-1", "ops", "Operational alerts")
	if err != nil {
		t.Fatalf("CreateTopic() error = %v", err)
	}
	if _, err := svc.UpdateSettings(ctx, "admin-1", UpdateSettingsRequest{
		Enabled:              true,
		TopicID:              topicRow.ID,
		SignupCreatedEnabled: true,
	}); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	// Recipient has no paired device on the topic: flag must be false.
	got, err := svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if got.RecipientHasActiveDeviceForTopic {
		t.Fatalf("RecipientHasActiveDeviceForTopic = true, want false when no device is paired")
	}

	// Once a device is registered for the recipient on that topic, the flag flips.
	subs.withDevice["admin-1|ops"] = true
	got, err = svc.GetSettings(ctx)
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if !got.RecipientHasActiveDeviceForTopic {
		t.Fatalf("RecipientHasActiveDeviceForTopic = false, want true once a device is paired")
	}
}
