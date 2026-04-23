// Package device manages device registration, pairing, and push subscriptions.
package device

import (
	"context"
	"fmt"
	"time"

	"lucor.dev/beebuzz/internal/secure"
	"lucor.dev/beebuzz/internal/validator"
)

const (
	// PairingOTPTTL is the time-to-live for pairing OTPs.
	PairingOTPTTL = 5 * time.Minute
	// maxPairingAttempts is the maximum number of OTP verification attempts.
	maxPairingAttempts = 5
)

// Sentinel errors.
var (
	ErrDeviceNotFound        = fmt.Errorf("device not found")
	ErrPairingCodeInvalid    = fmt.Errorf("pairing code invalid or expired")
	ErrAtLeastOneTopic       = fmt.Errorf("at least one topic is required")
	ErrInvalidTopicSelection = fmt.Errorf("invalid topic selection")
	ErrInvalidPushEndpoint   = fmt.Errorf("invalid push endpoint")
	ErrInvalidAgeRecipient   = fmt.Errorf("invalid age recipient")
	ErrInvalidDeviceToken    = fmt.Errorf("invalid device token")
)

// TopicValidator verifies that topic IDs belong to the given user.
type TopicValidator interface {
	ValidateTopicIDs(ctx context.Context, userID string, topicIDs []string) error
}

// Device is the DB row type for the devices table.
type Device struct {
	ID              string        `db:"id"`
	UserID          string        `db:"user_id"`
	Name            string        `db:"name"`
	Description     string        `db:"description"`
	IsActive        bool          `db:"is_active"`
	PairingStatus   PairingStatus `db:"pairing_status"`
	DeviceTokenHash *string       `db:"device_token_hash"`
	CreatedAt       int64         `db:"created_at"`
	UpdatedAt       int64         `db:"updated_at"`
	// Derived from LEFT JOIN push_subscriptions (not a column in devices table).
	SubCreatedAt    *int64  `db:"sub_created_at"`
	SubAgeRecipient *string `db:"sub_age_recipient"`
}

// PairingStatus is the canonical device pairing state exposed to clients.
type PairingStatus string

const (
	PairingStatusPending          PairingStatus = "pending"
	PairingStatusPaired           PairingStatus = "paired"
	PairingStatusUnpaired         PairingStatus = "unpaired"
	PairingStatusSubscriptionGone PairingStatus = "subscription_gone"
)

// PairingCode is the DB row type for the device_pairing_codes table.
type PairingCode struct {
	CodeHash     string `db:"code_hash"`
	DeviceID     string `db:"device_id"`
	ExpiresAt    int64  `db:"expires_at"`
	UsedAt       *int64 `db:"used_at"`
	AttemptCount int    `db:"attempt_count"`
	CreatedAt    int64  `db:"created_at"`
}

// DeviceTopic is the DB row type for the device_topics table.
type DeviceTopic struct {
	DeviceID  string `db:"device_id"`
	TopicID   string `db:"topic_id"`
	CreatedAt int64  `db:"created_at"`
}

// PushSubscription is the DB row type for the push_subscriptions table.
type PushSubscription struct {
	DeviceID     string `db:"device_id"`
	Endpoint     string `db:"endpoint"`
	P256dh       string `db:"p256dh"`
	Auth         string `db:"auth"`
	AgeRecipient string `db:"age_recipient"`
	CreatedAt    int64  `db:"created_at"`
	UpdatedAt    int64  `db:"updated_at"`
}

// DeviceResponse is the HTTP response for a single device.
type DeviceResponse struct {
	ID                      string        `json:"id"`
	Name                    string        `json:"name"`
	Description             string        `json:"description"`
	IsActive                bool          `json:"is_active"`
	PairingStatus           PairingStatus `json:"pairing_status"`
	PairedAt                *time.Time    `json:"paired_at"`
	AgeRecipient            *string       `json:"age_recipient"`
	AgeRecipientFingerprint *string       `json:"age_recipient_fingerprint"`
	TopicIDs                []string      `json:"topic_ids"`
	CreatedAt               time.Time     `json:"created_at"`
	UpdatedAt               time.Time     `json:"updated_at"`
}

// DevicesListResponse is the HTTP response for a list of devices.
type DevicesListResponse struct {
	Data []DeviceResponse `json:"data"`
}

// DeviceKeyDescriptor is the HTTP/API representation of a paired device key.
type DeviceKeyDescriptor struct {
	DeviceID                string    `json:"device_id"`
	DeviceName              string    `json:"device_name"`
	PairedAt                time.Time `json:"paired_at"`
	AgeRecipient            string    `json:"age_recipient"`
	AgeRecipientFingerprint string    `json:"age_recipient_fingerprint"`
}

// DeviceKeysResponse is the HTTP response for paired device keys.
type DeviceKeysResponse struct {
	Data []DeviceKeyDescriptor `json:"data"`
}

// CreatedDeviceResponse is the HTTP response for a newly created device (includes pairing code + QR).
type CreatedDeviceResponse struct {
	Device      DeviceResponse `json:"device"`
	PairingCode string         `json:"pairing_code"`
	PairingURL  string         `json:"pairing_url"`
	QRCode      string         `json:"qr_code"`
	ExpiresAt   time.Time      `json:"expires_at"`
}

// PairingCodeResponse is the HTTP response for a regenerated pairing code.
type PairingCodeResponse struct {
	PairingCode string    `json:"pairing_code"`
	PairingURL  string    `json:"pairing_url"`
	QRCode      string    `json:"qr_code"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// CreateDeviceRequest is the HTTP request for creating a device.
type CreateDeviceRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Topics      []string `json:"topics"`
}

// Validate checks the create device request fields.
func (r *CreateDeviceRequest) Validate() []error {
	return validator.Validate(
		validator.NotBlank("name", r.Name),
		validator.RequiredSlice("topics", r.Topics),
		validator.UniqueStrings("topics", r.Topics),
		validator.MaxLen("name", r.Name, validator.MaxDisplayNameLen),
		validator.MaxLen("description", r.Description, validator.MaxDescriptionLen),
	)
}

// UpdateDeviceRequest is the HTTP request for updating a device.
type UpdateDeviceRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Topics      []string `json:"topics"`
}

// Validate checks the update device request fields.
func (r *UpdateDeviceRequest) Validate() []error {
	return validator.Validate(
		validator.NotBlank("name", r.Name),
		validator.RequiredSlice("topics", r.Topics),
		validator.UniqueStrings("topics", r.Topics),
		validator.MaxLen("name", r.Name, validator.MaxDisplayNameLen),
		validator.MaxLen("description", r.Description, validator.MaxDescriptionLen),
	)
}

// PairRequest is the HTTP request for pairing a device with a push subscription.
type PairRequest struct {
	PairingCode  string `json:"pairing_code"`
	Endpoint     string `json:"endpoint"`
	P256dh       string `json:"p256dh"`
	Auth         string `json:"auth"`
	AgeRecipient string `json:"age_recipient"`
}

// PairResponse is the HTTP response returned after a successful pairing.
type PairResponse struct {
	DeviceID    string `json:"device_id"`
	DeviceToken string `json:"device_token"`
}

// PairingStatusResponse is the public Hive pairing status payload.
type PairingStatusResponse struct {
	DeviceID      string        `json:"device_id"`
	PairingStatus PairingStatus `json:"pairing_status"`
}

// Validate checks the pair request fields.
func (r *PairRequest) Validate() []error {
	return validator.Validate(
		validator.NotBlank("pairing_code", r.PairingCode),
		validator.NotBlank("endpoint", r.Endpoint),
		validator.NotBlank("p256dh", r.P256dh),
		validator.NotBlank("auth", r.Auth),
		validator.NotBlank("age_recipient", r.AgeRecipient),
		validator.AgeRecipient("age_recipient", r.AgeRecipient),
	)
}

// ToDeviceResponse converts a Device DB row to a DeviceResponse.
func ToDeviceResponse(d *Device, topicIDs []string) DeviceResponse {
	var pairedAt *time.Time
	if d.SubCreatedAt != nil {
		t := time.UnixMilli(*d.SubCreatedAt).UTC()
		pairedAt = &t
	}

	var ageRecipient *string
	var ageRecipientFingerprint *string
	if d.SubAgeRecipient != nil && *d.SubAgeRecipient != "" {
		ageRecipient = d.SubAgeRecipient
		fingerprint := secure.Fingerprint(*d.SubAgeRecipient)
		ageRecipientFingerprint = &fingerprint
	}

	return DeviceResponse{
		ID:                      d.ID,
		Name:                    d.Name,
		Description:             d.Description,
		IsActive:                d.IsActive,
		PairingStatus:           d.PairingStatus,
		PairedAt:                pairedAt,
		AgeRecipient:            ageRecipient,
		AgeRecipientFingerprint: ageRecipientFingerprint,
		TopicIDs:                topicIDs,
		CreatedAt:               time.UnixMilli(d.CreatedAt).UTC(),
		UpdatedAt:               time.UnixMilli(d.UpdatedAt).UTC(),
	}
}

// ToDeviceKeyDescriptor converts a paired device row to a device-key descriptor.
func ToDeviceKeyDescriptor(d *Device) DeviceKeyDescriptor {
	pairedAt := time.UnixMilli(*d.SubCreatedAt).UTC()
	ageRecipient := *d.SubAgeRecipient

	return DeviceKeyDescriptor{
		DeviceID:                d.ID,
		DeviceName:              d.Name,
		PairedAt:                pairedAt,
		AgeRecipient:            ageRecipient,
		AgeRecipientFingerprint: secure.Fingerprint(ageRecipient),
	}
}
