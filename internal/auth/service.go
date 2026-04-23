package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"lucor.dev/beebuzz/internal/core"
	"lucor.dev/beebuzz/internal/mailer"
	"lucor.dev/beebuzz/internal/secure"
)

const (
	challengeExpiry        = 10 * time.Minute
	sessionExpiry          = 30 * 24 * time.Hour
	sessionIdleTimeout     = 7 * 24 * time.Hour
	sessionRefreshInterval = 5 * time.Minute
	maxOTPAttempts         = 5
)

// TopicInitializer creates default topics for new users.
type TopicInitializer interface {
	CreateDefaultTopic(ctx context.Context, userID string) error
}

// Service provides authentication business logic.
type Service struct {
	repo                *Repository
	mailer              mailer.Mailer
	baseURL             string
	privateBeta         bool
	bootstrapAdminEmail string
	globalThrottle      *GlobalAuthThrottle
	emailThrottle       *EmailThrottle
	topicInitializer    TopicInitializer
	log                 *slog.Logger
}

// NewService creates a new auth service.
func NewService(repo *Repository, m mailer.Mailer, baseURL string, topicInit TopicInitializer, logger *slog.Logger) *Service {
	return &Service{
		repo:             repo,
		mailer:           m,
		baseURL:          baseURL,
		privateBeta:      false,
		topicInitializer: topicInit,
		log:              logger,
	}
}

// UsePrivateBeta toggles waitlist gating for the auth flow.
func (s *Service) UsePrivateBeta(enabled bool) {
	s.privateBeta = enabled
}

// SetBootstrapAdminEmail configures the email identity allowed to bootstrap admin access.
// The value is normalized once so later comparisons stay consistent with auth identity lookup.
func (s *Service) SetBootstrapAdminEmail(email string) {
	s.bootstrapAdminEmail = normalizeEmail(email)
}

// SetGlobalThrottle installs the instance-wide auth throttle used as a safety valve.
func (s *Service) SetGlobalThrottle(throttle *GlobalAuthThrottle) {
	s.globalThrottle = throttle
}

// SetEmailThrottle installs the login-email throttle used to silently absorb abuse.
func (s *Service) SetEmailThrottle(throttle *EmailThrottle) {
	s.emailThrottle = throttle
}

// RequestAuth initiates authentication for the given email.
func (s *Service) RequestAuth(ctx context.Context, email string, state string, reason *string) (bool, error) {
	email = normalizeEmail(email)

	if s.globalThrottle != nil && !s.globalThrottle.Allow() {
		s.log.Warn("global auth throttle triggered")
		return false, ErrGlobalRateLimit
	}

	if s.emailThrottle != nil && !s.emailThrottle.Allow(email) {
		s.log.Info("auth email request throttled")
		return true, nil
	}

	opts := CreateUserOptions{AccountStatus: core.AccountStatusActive}
	if s.privateBeta && !s.isBootstrapAdminEmail(email) {
		opts.AccountStatus = core.AccountStatusPending
		opts.SignupReason = reason
	} else if !s.privateBeta {
		opts.StartTrialOnCreate = true
	}

	user, created, err := s.repo.GetOrCreateUser(ctx, email, opts)
	if err != nil {
		s.log.Error("failed to get or create user", "error", err)
		return false, err
	}

	logger := s.log.With("user_id", user.ID)

	if created {
		logger.Info("new user created", "is_admin", user.IsAdmin, "account_status", user.AccountStatus)
		if s.topicInitializer != nil {
			if err := s.topicInitializer.CreateDefaultTopic(ctx, user.ID); err != nil {
				logger.Warn("failed to create default topic, will be retried on first access", "error", err)
			}
		}
	}

	switch user.AccountStatus {
	case core.AccountStatusBlocked:
		logger.Info("account blocked, ignoring auth request")
		return false, nil
	case core.AccountStatusPending:
		logger.Info("account pending approval, ignoring auth request")
		return false, nil
	case core.AccountStatusActive:
	}

	otp, err := secure.NewOTP()
	if err != nil {
		msg := "failed to generate OTP"
		logger.Error(msg, "error", err)
		return false, fmt.Errorf("%s: %w", msg, err)
	}

	expiresAt := time.Now().UTC().Add(challengeExpiry).UnixMilli()
	_, err = s.repo.CreateAuthChallenge(ctx, user.ID, state, secure.Hash(otp), expiresAt)
	if err != nil {
		msg := "failed to save auth challenge"
		logger.Error(msg, "error", err)
		return false, fmt.Errorf("%s: %w", msg, err)
	}

	logger.Debug("auth challenge created", "expires_at", expiresAt)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err = s.mailer.SendRequestAuth(ctx, user.Email, otp)
	if err != nil {
		msg := "failed to send request auth email"
		logger.Error(msg, "error", err)
		return false, fmt.Errorf("%s: %w", msg, err)
	}

	logger.Info("auth email sent")
	return true, nil
}

// VerifyOTP verifies an OTP code against the stored challenge.
func (s *Service) VerifyOTP(ctx context.Context, otp string, state string) (string, error) {
	challenge, err := s.repo.GetAuthChallengeByState(ctx, state)
	if err != nil {
		msg := "failed to retrieve auth challenge for OTP verification"
		s.log.Error(msg, "error", err)
		return "", fmt.Errorf("%s: %w", msg, err)
	}

	if challenge == nil {
		s.log.Warn("challenge not found for OTP verification")
		return "", ErrOTPNotFound
	}

	logger := s.log.With("user_id", challenge.UserID)
	if challenge.ExpiresAt < time.Now().UnixMilli() {
		logger.Warn("challenge expired for OTP verification")
		return "", ErrOTPExpired
	}

	if challenge.UsedAt != nil {
		logger.Warn("challenge already used for OTP verification")
		return "", ErrOTPUsed
	}

	if challenge.AttemptCount >= maxOTPAttempts {
		logger.Warn("max OTP attempts exceeded", "attempts", challenge.AttemptCount)
		return "", ErrOTPMaxAttempts
	}

	if !secure.Verify(otp, challenge.OTPHash) {
		if err := s.repo.IncrementAuthChallengeAttempts(ctx, challenge.ID); err != nil {
			logger.Error("failed to increment OTP attempts", "error", err)
		}
		logger.Warn("invalid OTP", "attempts", challenge.AttemptCount+1)
		return "", ErrOTPInvalid
	}

	if err := s.repo.MarkAuthChallengeAsUsed(ctx, challenge.ID); err != nil {
		msg := "failed to mark challenge as used after OTP verification"
		logger.Error(msg, "error", err)
		return "", fmt.Errorf("%s: %w", msg, err)

	}

	logger.Info("OTP verified")
	return challenge.UserID, nil
}

// CreateSession creates a new authenticated session for the user.
func (s *Service) CreateSession(ctx context.Context, userID string) (*Session, error) {
	logger := s.log.With("user_id", userID)

	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		msg := "failed to get user before session creation"
		logger.Error(msg, "error", err)
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	if user == nil {
		msg := "user missing before session creation"
		logger.Error(msg)
		return nil, fmt.Errorf("%s", msg)
	}

	if user.AccountStatus != core.AccountStatusActive {
		return nil, core.ErrUnauthorized
	}

	if s.isBootstrapAdminEmail(user.Email) {
		changed, err := s.repo.EnsureUserAdmin(ctx, userID)
		if err != nil {
			msg := "failed to ensure bootstrap admin"
			logger.Error(msg, "error", err)
			return nil, fmt.Errorf("%s: %w", msg, err)
		}
		if changed {
			logger.Info("bootstrap admin granted")
		}
	}

	sessionToken, err := secure.NewSessionToken()
	if err != nil {
		msg := "failed to generate session token"
		logger.Error(msg, "error", err)
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	// Session tokens are hashed at rest so a DB leak does not become immediate session takeover.
	sessionTokenHash := secure.Hash(sessionToken)
	expiresAt := time.Now().UTC().Add(sessionExpiry)
	if err := s.repo.CreateSession(ctx, sessionTokenHash, userID, expiresAt.UnixMilli()); err != nil {
		msg := "failed to create session"
		logger.Error(msg, "error", err)
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	logger.Debug("session created")
	return &Session{
		Token:     sessionToken,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateSession validates a session token and returns the associated user.
func (s *Service) ValidateSession(ctx context.Context, sessionToken string) (*User, error) {
	now := time.Now().UTC()
	sessionTokenHash := secure.Hash(sessionToken)
	session, err := s.repo.GetSession(ctx, sessionTokenHash)
	if err != nil {
		msg := "failed to get session"
		s.log.Error(msg, "error", err)
		return nil, fmt.Errorf("%s: %w", msg, err)
	}

	if session == nil {
		s.log.Warn("session unauthorized")
		return nil, core.ErrUnauthorized
	}

	logger := s.log.With("user_id", session.UserID)
	if session.ExpiresAt < now.UnixMilli() {
		logger.Warn("session expired")
		return nil, core.ErrUnauthorized
	}
	if session.LastSeenAt < now.Add(-sessionIdleTimeout).UnixMilli() {
		logger.Warn("session idle timeout exceeded")
		return nil, core.ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil {
		msg := "failed to get user for session"
		logger.Error(msg, "error", err)
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	if user == nil {
		logger.Warn("session user not found")
		return nil, core.ErrUnauthorized
	}

	if user.AccountStatus != core.AccountStatusActive {
		logger.Warn("session for non-active user", "account_status", user.AccountStatus)
		return nil, core.ErrUnauthorized
	}

	if now.Sub(time.UnixMilli(session.LastSeenAt).UTC()) >= sessionRefreshInterval {
		if err := s.repo.TouchSession(ctx, session.TokenHash); err != nil {
			logger.Warn("failed to refresh session last seen", "error", err)
		}
	}

	logger.Debug("session validated")

	return user, nil
}

// RevokeSession revokes a session for the given user.
func (s *Service) RevokeSession(ctx context.Context, userID string, sessionToken string) error {
	logger := s.log.With("user_id", userID)
	sessionTokenHash := secure.Hash(sessionToken)
	deletedSessions, err := s.repo.DeleteSession(ctx, userID, sessionTokenHash)
	if err != nil {
		msg := "failed to revoke session"
		logger.Error(msg, "error", err)
		return fmt.Errorf("%s: %w", msg, err)
	}
	if deletedSessions > 0 {
		logger.Info("user logged out")
	}
	return nil
}

// RevokeAllSessions revokes all sessions for the given user.
func (s *Service) RevokeAllSessions(ctx context.Context, userID string) error {
	logger := s.log.With("user_id", userID)
	deletedSessions, err := s.repo.DeleteSessionsByUserID(ctx, userID)
	if err != nil {
		msg := "failed to revoke all sessions"
		logger.Error(msg, "error", err)
		return fmt.Errorf("%s: %w", msg, err)
	}

	if deletedSessions > 0 {
		logger.Info("all user sessions revoked", "deleted_sessions", deletedSessions)
	}

	return nil
}

// CleanupExpired removes expired auth artifacts that no longer have operational value.
func (s *Service) CleanupExpired(ctx context.Context) error {
	expiredBefore := time.Now().UTC().UnixMilli()
	idleBefore := time.Now().UTC().Add(-sessionIdleTimeout).UnixMilli()
	deletedSessions, err := s.repo.DeleteStaleSessions(ctx, expiredBefore, idleBefore)
	if err != nil {
		return fmt.Errorf("cleanup stale sessions: %w", err)
	}

	deletedChallenges, err := s.repo.DeleteStaleAuthChallenges(ctx)
	if err != nil {
		return fmt.Errorf("cleanup stale auth challenges: %w", err)
	}

	if deletedSessions > 0 || deletedChallenges > 0 {
		s.log.Info(
			"auth cleanup removed stale rows",
			"deleted_sessions", deletedSessions,
			"deleted_challenges", deletedChallenges,
		)
	}

	return nil
}

// normalizeEmail applies the conservative auth identity canonical form used for lookup and storage.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (s *Service) isBootstrapAdminEmail(email string) bool {
	return s.bootstrapAdminEmail != "" && normalizeEmail(email) == s.bootstrapAdminEmail
}
