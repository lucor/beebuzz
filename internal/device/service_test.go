package device

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"lucor.dev/beebuzz/internal/auth"
	"lucor.dev/beebuzz/internal/testutil"
	"lucor.dev/beebuzz/internal/topic"
)

const testAgeRecipient = "age1lnhya3u0lkqlw2txk56ltge6n9gm3p4lj2snmt4yaftwx005wetsmyx0zx"

func newTestDeviceService(repo *Repository) *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	topicRepo := topic.NewRepository(repo.db)
	topicSvc := topic.NewService(topicRepo, logger)
	return NewService(repo, topicSvc, logger)
}

func TestCreateDeviceRollsBackOnTopicAssociationFailure(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-create-rollback@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	_, _, _, err = deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{"missing-topic"})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("CreateDevice() error = %v, want %v", err, ErrInvalidTopicSelection)
	}

	devices, err := deviceRepo.ListDevicesByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListDevicesByUser: %v", err)
	}
	if len(devices) != 0 {
		t.Fatalf("ListDevicesByUser() len = %d, want 0", len(devices))
	}
}

func TestUpdateDeviceRollsBackOnTopicAssociationFailure(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-update-rollback@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	originalTopic, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	device, _, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{originalTopic.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	err = deviceSvc.UpdateDevice(ctx, user.ID, device.ID, "tablet", "new desc", []string{"missing-topic"})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("UpdateDevice() error = %v, want %v", err, ErrInvalidTopicSelection)
	}

	storedDevice, err := deviceRepo.GetDeviceByIDAndUser(ctx, device.ID, user.ID)
	if err != nil {
		t.Fatalf("GetDeviceByIDAndUser: %v", err)
	}
	if storedDevice.Name != "phone" {
		t.Fatalf("device name = %q, want %q", storedDevice.Name, "phone")
	}
	if storedDevice.Description != "desc" {
		t.Fatalf("device description = %q, want %q", storedDevice.Description, "desc")
	}

	topicIDs, err := deviceRepo.GetDeviceTopicIDs(ctx, device.ID)
	if err != nil {
		t.Fatalf("GetDeviceTopicIDs: %v", err)
	}
	if len(topicIDs) != 1 || topicIDs[0] != originalTopic.ID {
		t.Fatalf("device topicIDs = %#v, want [%q]", topicIDs, originalTopic.ID)
	}
}

func TestCreateDeviceRejectsTopicOwnedByAnotherUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	owner, _, err := authRepo.GetOrCreateUser(ctx, "device-owner@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser owner: %v", err)
	}
	other, _, err := authRepo.GetOrCreateUser(ctx, "device-other@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser other: %v", err)
	}

	otherTopic, err := topicRepo.Create(ctx, other.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, _, _, err = deviceSvc.CreateDevice(ctx, owner.ID, "phone", "desc", []string{otherTopic.ID})
	if !errors.Is(err, ErrInvalidTopicSelection) {
		t.Fatalf("CreateDevice() error = %v, want %v", err, ErrInvalidTopicSelection)
	}
}

func TestServicePairRejectsUnsupportedPushEndpoint(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-pair@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, otp, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	_, _, err = deviceSvc.Pair(ctx, otp, "https://example.com/push", "p256dh", "auth", "age1lnhya3u0lkqlw2txk56ltge6n9gm3p4lj2snmt4yaftwx005wetsmyx0zx")
	if !errors.Is(err, ErrInvalidPushEndpoint) {
		t.Fatalf("Pair() error = %v, want %v", err, ErrInvalidPushEndpoint)
	}
}

func TestServicePairRejectsInvalidAgeRecipient(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-pair-invalid-age@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, otp, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	_, _, err = deviceSvc.Pair(ctx, otp, "https://fcm.googleapis.com/fcm/send/test", "p256dh", "auth", "invalid-age-recipient")
	if !errors.Is(err, ErrInvalidAgeRecipient) {
		t.Fatalf("Pair() error = %v, want %v", err, ErrInvalidAgeRecipient)
	}
}

func TestServiceDeleteDeviceSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-delete@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	device, _, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	err = deviceSvc.DeleteDevice(ctx, user.ID, device.ID)
	if err != nil {
		t.Fatalf("DeleteDevice: %v", err)
	}

	_, err = deviceRepo.GetDeviceByIDAndUser(ctx, device.ID, user.ID)
	if err != nil {
		t.Fatalf("GetDeviceByIDAndUser: %v", err)
	}
}

func TestServiceDeleteDeviceNotFound(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-delete-notfound@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	err = deviceSvc.DeleteDevice(ctx, user.ID, "non-existent-device-id")
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("DeleteDevice() error = %v, want %v", err, ErrDeviceNotFound)
	}
}

func TestServiceListDevicesEmpty(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-list-empty@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	devices, err := deviceSvc.ListDevices(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 0 {
		t.Fatalf("ListDevices() len = %d, want 0", len(devices))
	}
}

func TestServiceListDevicesReturnsAllActive(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-list-all@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	device1, _, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc1", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice 1: %v", err)
	}

	device2, _, _, err := deviceSvc.CreateDevice(ctx, user.ID, "tablet", "desc2", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice 2: %v", err)
	}

	devices, err := deviceSvc.ListDevices(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListDevices: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("ListDevices() len = %d, want 2", len(devices))
	}

	deviceMap := make(map[string]DeviceResponse)
	for _, d := range devices {
		deviceMap[d.ID] = d
	}

	if _, ok := deviceMap[device1.ID]; !ok {
		t.Errorf("device1 not in list")
	}
	if _, ok := deviceMap[device2.ID]; !ok {
		t.Errorf("device2 not in list")
	}
	if got := deviceMap[device1.ID].PairingStatus; got != PairingStatusPending {
		t.Fatalf("device1 pairing_status = %q, want %q", got, PairingStatusPending)
	}
	if got := deviceMap[device2.ID].PairingStatus; got != PairingStatusPending {
		t.Fatalf("device2 pairing_status = %q, want %q", got, PairingStatusPending)
	}
}

func TestServiceUnpairDevice(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-unpair@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	device, otp, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	_, _, err = deviceSvc.Pair(ctx, otp, "https://fcm.googleapis.com/fcm/send/pair", "p256dh", "auth", "age1lnhya3u0lkqlw2txk56ltge6n9gm3p4lj2snmt4yaftwx005wetsmyx0zx")
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	err = deviceSvc.UnpairDevice(ctx, user.ID, device.ID)
	if err != nil {
		t.Fatalf("UnpairDevice: %v", err)
	}

	storedDevice, err := deviceRepo.GetDeviceByIDAndUser(ctx, device.ID, user.ID)
	if err != nil {
		t.Fatalf("GetDeviceByIDAndUser: %v", err)
	}
	if storedDevice.SubCreatedAt != nil {
		t.Error("device should be unpaired")
	}
	if storedDevice.PairingStatus != PairingStatusUnpaired {
		t.Fatalf("pairing_status = %q, want %q", storedDevice.PairingStatus, PairingStatusUnpaired)
	}
}

func TestServiceMarkSubscriptionGone(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-subscription-gone@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	device, otp, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	if _, _, err := deviceSvc.Pair(ctx, otp, "https://fcm.googleapis.com/fcm/send/pair", "p256dh", "auth", testAgeRecipient); err != nil {
		t.Fatalf("Pair: %v", err)
	}

	if err := deviceSvc.MarkSubscriptionGone(ctx, device.ID); err != nil {
		t.Fatalf("MarkSubscriptionGone: %v", err)
	}

	storedDevice, err := deviceRepo.GetDeviceByIDAndUser(ctx, device.ID, user.ID)
	if err != nil {
		t.Fatalf("GetDeviceByIDAndUser: %v", err)
	}
	if storedDevice.SubCreatedAt != nil {
		t.Fatal("subscription should be deleted")
	}
	if storedDevice.PairingStatus != PairingStatusSubscriptionGone {
		t.Fatalf("pairing_status = %q, want %q", storedDevice.PairingStatus, PairingStatusSubscriptionGone)
	}
}

func TestServiceGetPairingStatus(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-pairing-health@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, otp, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	// Pair to get a device token
	deviceID, dt, err := deviceSvc.Pair(ctx, otp, "https://fcm.googleapis.com/fcm/send/health", "p256dh", "auth", testAgeRecipient)
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	status, err := deviceSvc.GetPairingStatus(ctx, deviceID, dt)
	if err != nil {
		t.Fatalf("GetPairingStatus: %v", err)
	}
	if status.PairingStatus != PairingStatusPaired {
		t.Fatalf("pairing_status = %q, want %q", status.PairingStatus, PairingStatusPaired)
	}
}

func TestServiceUnpairDeviceAlreadyUnpaired(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-unpair-already@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	device, _, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	err = deviceSvc.UnpairDevice(ctx, user.ID, device.ID)
	if err != nil {
		t.Fatalf("UnpairDevice: %v", err)
	}
}

func TestServiceGetDeviceKeysByUser(t *testing.T) {
	db := testutil.NewDB(t)
	ctx := context.Background()

	authRepo := auth.NewRepository(db)
	topicRepo := topic.NewRepository(db)
	deviceRepo := NewRepository(db)
	deviceSvc := newTestDeviceService(deviceRepo)

	user, _, err := authRepo.GetOrCreateUser(ctx, "device-age-keys@example.com")
	if err != nil {
		t.Fatalf("GetOrCreateUser: %v", err)
	}

	keys, err := deviceSvc.GetDeviceKeysByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetDeviceKeysByUser: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("GetDeviceKeysByUser() len = %d, want 0", len(keys))
	}

	topicRow, err := topicRepo.Create(ctx, user.ID, "alerts", "")
	if err != nil {
		t.Fatalf("topic.Create: %v", err)
	}

	_, otp, _, err := deviceSvc.CreateDevice(ctx, user.ID, "phone", "desc", []string{topicRow.ID})
	if err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}

	_, _, err = deviceSvc.Pair(ctx, otp, "https://fcm.googleapis.com/fcm/send/pair", "p256dh", "auth", "age1lnhya3u0lkqlw2txk56ltge6n9gm3p4lj2snmt4yaftwx005wetsmyx0zx")
	if err != nil {
		t.Fatalf("Pair: %v", err)
	}

	keys, err = deviceSvc.GetDeviceKeysByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetDeviceKeysByUser: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("GetDeviceKeysByUser() len = %d, want 1", len(keys))
	}
	if keys[0].DeviceName != "phone" {
		t.Fatalf("DeviceName: got %q, want %q", keys[0].DeviceName, "phone")
	}
	if keys[0].AgeRecipient != "age1lnhya3u0lkqlw2txk56ltge6n9gm3p4lj2snmt4yaftwx005wetsmyx0zx" {
		t.Fatalf("AgeRecipient: got %q", keys[0].AgeRecipient)
	}
	if keys[0].AgeRecipientFingerprint == "" {
		t.Fatal("AgeRecipientFingerprint: got empty value")
	}
}
