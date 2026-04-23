package device

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/validator"
)

// Service handles device business logic.
type Service struct {
	repo           *Repository
	topicValidator TopicValidator
	log            *slog.Logger
}

// NewService creates a new device Service.
func NewService(repo *Repository, topicValidator TopicValidator, log *slog.Logger) *Service {
	return &Service{repo: repo, topicValidator: topicValidator, log: log}
}

// CreateDevice creates a device with topics and a pairing OTP. Returns device, raw OTP, and expiry.
func (s *Service) CreateDevice(ctx context.Context, userID, name, description string, topicIDs []string) (*Device, string, time.Time, error) {
	if len(topicIDs) == 0 {
		return nil, "", time.Time{}, ErrAtLeastOneTopic
	}
	if err := s.topicValidator.ValidateTopicIDs(ctx, userID, topicIDs); err != nil {
		if errors.Is(err, ErrInvalidTopicSelection) {
			return nil, "", time.Time{}, err
		}
		return nil, "", time.Time{}, ErrInvalidTopicSelection
	}

	deviceID := uuid.NewString()
	otp, err := secure.NewOTP()
	if err != nil {
		s.log.Error("failed to generate pairing OTP", "error", err, "device_id", deviceID)
		return nil, "", time.Time{}, err
	}

	expiresAt := time.Now().Add(PairingOTPTTL)
	otpHash := secure.Hash(otp)

	if err := s.repo.CreateDeviceWithTopicsAndPairingCode(ctx, deviceID, userID, name, description, topicIDs, otpHash, expiresAt.UnixMilli()); err != nil {
		s.log.Error("failed to create device", "error", err, "user_id", userID, "device_id", deviceID)
		return nil, "", time.Time{}, err
	}

	device, err := s.repo.GetDeviceByIDAndUser(ctx, deviceID, userID)
	if err != nil {
		s.log.Error("failed to get created device", "error", err, "device_id", deviceID)
		return nil, "", time.Time{}, err
	}

	s.log.Info("device created", "device_id", deviceID, "user_id", userID)
	return device, otp, expiresAt, nil
}

// ListDevices returns all active devices for a user with their topics.
func (s *Service) ListDevices(ctx context.Context, userID string) ([]DeviceResponse, error) {
	devices, err := s.repo.ListDevicesByUser(ctx, userID)
	if err != nil {
		s.log.Error("failed to list devices", "error", err, "user_id", userID)
		return nil, err
	}

	responses := make([]DeviceResponse, 0, len(devices))
	for _, d := range devices {
		topicIDs, err := s.repo.GetDeviceTopicIDs(ctx, d.ID)
		if err != nil {
			s.log.Error("failed to get device topics", "error", err, "device_id", d.ID)
			return nil, err
		}
		responses = append(responses, ToDeviceResponse(&d, topicIDs))
	}
	return responses, nil
}

// UpdateDevice validates ownership and updates the device name, description, and topics.
func (s *Service) UpdateDevice(ctx context.Context, userID, deviceID, name, description string, topicIDs []string) error {
	if len(topicIDs) == 0 {
		return ErrAtLeastOneTopic
	}
	if err := s.topicValidator.ValidateTopicIDs(ctx, userID, topicIDs); err != nil {
		if errors.Is(err, ErrInvalidTopicSelection) {
			return err
		}
		return ErrInvalidTopicSelection
	}

	device, err := s.repo.GetDeviceByIDAndUser(ctx, deviceID, userID)
	if err != nil {
		s.log.Error("failed to get device for update", "error", err, "device_id", deviceID)
		return err
	}
	if device == nil {
		return ErrDeviceNotFound
	}

	if err := s.repo.UpdateDeviceWithTopics(ctx, userID, deviceID, name, description, topicIDs); err != nil {
		s.log.Error("failed to update device", "error", err, "device_id", deviceID)
		return err
	}

	s.log.Info("device updated", "device_id", deviceID, "user_id", userID)
	return nil
}

// DeleteDevice validates ownership and soft-deletes the device.
func (s *Service) DeleteDevice(ctx context.Context, userID, deviceID string) error {
	device, err := s.repo.GetDeviceByIDAndUser(ctx, deviceID, userID)
	if err != nil {
		s.log.Error("failed to get device for delete", "error", err, "device_id", deviceID)
		return err
	}
	if device == nil {
		return ErrDeviceNotFound
	}

	if err := s.repo.DeleteDevice(ctx, deviceID); err != nil {
		s.log.Error("failed to delete device", "error", err, "device_id", deviceID)
		return err
	}

	s.log.Info("device deleted", "device_id", deviceID, "user_id", userID)
	return nil
}

// RegeneratePairingOTP validates ownership, invalidates old codes, and creates a new OTP.
func (s *Service) RegeneratePairingOTP(ctx context.Context, userID, deviceID string) (string, time.Time, error) {
	device, err := s.repo.GetDeviceByIDAndUser(ctx, deviceID, userID)
	if err != nil {
		s.log.Error("failed to get device for OTP regen", "error", err, "device_id", deviceID)
		return "", time.Time{}, err
	}
	if device == nil {
		return "", time.Time{}, ErrDeviceNotFound
	}

	if err := s.repo.InvalidatePairingCodes(ctx, deviceID); err != nil {
		s.log.Error("failed to invalidate pairing codes", "error", err, "device_id", deviceID)
		return "", time.Time{}, err
	}

	otp, expiresAt, err := s.createPairingOTP(ctx, deviceID)
	if err != nil {
		return "", time.Time{}, err
	}

	s.log.Info("pairing OTP regenerated", "device_id", deviceID, "user_id", userID)
	return otp, expiresAt, nil
}

// Pair verifies the OTP and consumes it atomically with push subscription storage.
func (s *Service) Pair(ctx context.Context, otp, endpoint, p256dh, authKey, ageRecipient string) (string, string, error) {
	otpHash := secure.Hash(otp)

	pc, err := s.repo.GetActivePairingCode(ctx, otpHash)
	if err != nil {
		s.log.Error("failed to look up pairing OTP", "error", err)
		return "", "", err
	}
	if pc == nil {
		return "", "", ErrPairingCodeInvalid
	}

	logger := s.log.With("device_id", pc.DeviceID)

	pushHost, err := validatePushEndpoint(endpoint)
	if err != nil {
		// Log only the host, never the full endpoint, because the path can carry provider tokens.
		logger.Warn("rejected unsupported push endpoint host", "push_host", pushHost)
		return "", "", ErrInvalidPushEndpoint
	}
	if err := validator.AgeRecipient("age_recipient", ageRecipient); err != nil {
		logger.Warn("rejected invalid age recipient")
		return "", "", ErrInvalidAgeRecipient
	}

	// Atomically increment and check attempt count to prevent TOCTOU race
	attempts, err := s.repo.IncrementAndGetAttempts(ctx, otpHash)
	if err != nil {
		logger.Error("failed to increment pairing attempts", "error", err)
		return "", "", err
	}

	if attempts > maxPairingAttempts {
		logger.Warn("max pairing OTP attempts exceeded", "attempts", attempts)
		return "", "", ErrPairingCodeInvalid
	}

	// Constant-time verification after DB lookup
	if !secure.Verify(otp, pc.CodeHash) {
		return "", "", ErrPairingCodeInvalid
	}

	deviceToken, err := secure.NewDeviceToken()
	if err != nil {
		logger.Error("failed to generate device token", "error", err)
		return "", "", err
	}
	deviceTokenHash := secure.Hash(deviceToken)

	sub := PushSubscription{
		Endpoint:     endpoint,
		P256dh:       p256dh,
		Auth:         authKey,
		AgeRecipient: ageRecipient,
	}

	// Consume code + upsert subscription + set paired_at in a single transaction
	deviceID, err := s.repo.ConsumePairingCode(ctx, otpHash, sub, deviceTokenHash)
	if err != nil {
		logger.Error("failed to consume pairing OTP", "error", err)
		return "", "", err
	}
	if deviceID == "" {
		return "", "", ErrPairingCodeInvalid
	}

	logger.Info("device paired")
	return deviceID, deviceToken, nil
}

// createPairingOTP generates a 6-digit OTP, hashes it, and stores it. Returns raw OTP and expiry.
func (s *Service) createPairingOTP(ctx context.Context, deviceID string) (string, time.Time, error) {
	otp, err := secure.NewOTP()
	if err != nil {
		s.log.Error("failed to generate pairing OTP", "error", err, "device_id", deviceID)
		return "", time.Time{}, err
	}

	otpHash := secure.Hash(otp)
	expiresAt := time.Now().Add(PairingOTPTTL)

	if err := s.repo.CreatePairingCode(ctx, otpHash, deviceID, expiresAt.UnixMilli()); err != nil {
		s.log.Error("failed to create pairing OTP", "error", err, "device_id", deviceID)
		return "", time.Time{}, err
	}

	return otp, expiresAt, nil
}

// GetSubscribedDevices returns push subscriptions for devices subscribed to a topic for a user.
func (s *Service) GetSubscribedDevices(ctx context.Context, userID, topicName string) ([]PushSubscription, error) {
	subs, err := s.repo.GetPushSubscriptionsByUserAndTopic(ctx, userID, topicName)
	if err != nil {
		s.log.Error("failed to get subscribed devices", "error", err, "user_id", userID, "topic", topicName)
		return nil, err
	}
	return subs, nil
}

// GetDeviceKeysByUser returns paired devices with their age public keys for a user.
func (s *Service) GetDeviceKeysByUser(ctx context.Context, userID string) ([]DeviceKeyDescriptor, error) {
	devices, err := s.repo.GetDeviceKeysByUser(ctx, userID)
	if err != nil {
		s.log.Error("failed to get device keys by user", "error", err, "user_id", userID)
		return nil, err
	}

	response := make([]DeviceKeyDescriptor, 0, len(devices))
	for _, d := range devices {
		response = append(response, ToDeviceKeyDescriptor(&d))
	}

	return response, nil
}

// DeletePushSubscription deletes a push subscription by device ID.
func (s *Service) DeletePushSubscription(ctx context.Context, deviceID string) error {
	if err := s.repo.ClearPushSubscriptionWithStatus(ctx, deviceID, PairingStatusUnpaired); err != nil {
		s.log.Error("failed to delete push subscription", "error", err, "device_id", deviceID)
		return err
	}
	return nil
}

// MarkSubscriptionGone removes a push subscription invalidated by the push provider.
func (s *Service) MarkSubscriptionGone(ctx context.Context, deviceID string) error {
	if err := s.repo.ClearPushSubscriptionWithStatus(ctx, deviceID, PairingStatusSubscriptionGone); err != nil {
		s.log.Error("failed to mark subscription gone", "error", err, "device_id", deviceID)
		return err
	}
	return nil
}

// UnpairDevice validates ownership and deletes the push subscription.
func (s *Service) UnpairDevice(ctx context.Context, userID, deviceID string) error {
	device, err := s.repo.GetDeviceByIDAndUser(ctx, deviceID, userID)
	if err != nil {
		s.log.Error("failed to get device for unpair", "error", err, "device_id", deviceID)
		return err
	}
	if device == nil {
		return ErrDeviceNotFound
	}

	if device.SubCreatedAt == nil {
		// Already unpaired
		return nil
	}

	if err := s.repo.ClearPushSubscriptionWithStatus(ctx, deviceID, PairingStatusUnpaired); err != nil {
		s.log.Error("failed to unpair device", "error", err, "device_id", deviceID)
		return err
	}

	s.log.Info("device unpaired", "device_id", deviceID, "user_id", userID)
	return nil
}

// GetPairingStatus returns the canonical pairing status for a device ID, authenticated by device token.
func (s *Service) GetPairingStatus(ctx context.Context, deviceID, deviceToken string) (*PairingStatusResponse, error) {
	tokenHash := secure.Hash(deviceToken)
	device, err := s.repo.GetDeviceByIDAndTokenHash(ctx, deviceID, tokenHash)
	if err != nil {
		s.log.Error("failed to get pairing status", "error", err, "device_id", deviceID)
		return nil, err
	}
	if device == nil {
		return nil, ErrInvalidDeviceToken
	}

	return &PairingStatusResponse{
		DeviceID:      device.ID,
		PairingStatus: device.PairingStatus,
	}, nil
}
