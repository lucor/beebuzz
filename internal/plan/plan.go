// Package plan enforces hosted BeeBuzz plan entitlements.
package plan

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.beebuzz.app/beebuzz/internal/config"
	"go.beebuzz.app/beebuzz/internal/core"
)

var ErrQuotaExceeded = errors.New("quota exceeded")

// Action describes what a single operation consumes.
type Action struct {
	Messages int
}

// Entitlement is the runtime plan state for a user.
type Entitlement struct {
	Plan          core.Plan `db:"plan"`
	PlanExpiresAt *int64    `db:"plan_expires_at"`
}

// Usage is a privacy-safe usage aggregate.
type Usage struct {
	Messages int `db:"messages"`
}

// LimitUsage is a single operational quota window for the current plan.
type LimitUsage struct {
	Used     int       `json:"used"`
	Limit    int       `json:"limit"`
	ResetsAt time.Time `json:"resets_at"`
}

// UsageResponse reports current plan quota state for account dashboards.
type UsageResponse struct {
	Plan    core.Plan   `json:"plan"`
	Daily   *LimitUsage `json:"daily"`
	Monthly *LimitUsage `json:"monthly"`
}

// Reader reads plan entitlement and usage data.
type Reader interface {
	GetEntitlement(ctx context.Context, userID string) (*Entitlement, error)
	GetUsage(ctx context.Context, userID string, fromMs, toMs int64) (*Usage, error)
}

// Config holds plan enforcement settings derived from application config.
type Config struct {
	Hosted                     bool
	FreeMaxMessagesDay         int
	FreeMaxMessagesMonth       int
	HostedFairUseMessagesMonth int
}

// ConfigFromApp maps global application config into plan enforcement config.
func ConfigFromApp(cfg *config.Config) Config {
	return Config{
		Hosted:                     cfg.IsHosted(),
		FreeMaxMessagesDay:         cfg.FreeMaxMessagesDay,
		FreeMaxMessagesMonth:       cfg.FreeMaxMessagesMonth,
		HostedFairUseMessagesMonth: cfg.HostedFairUseMessagesMonth,
	}
}

// Enforcer decides whether hosted plan limits allow an action.
type Enforcer struct {
	cfg    Config
	reader Reader
	log    *slog.Logger
	now    func() time.Time
}

// NewEnforcer creates a plan enforcer.
func NewEnforcer(cfg Config, reader Reader, log *slog.Logger) *Enforcer {
	if log == nil {
		log = slog.Default()
	}
	return &Enforcer{
		cfg:    cfg,
		reader: reader,
		log:    log,
		now:    time.Now,
	}
}

// Allow returns nil when an action is allowed or ErrQuotaExceeded when a hosted
// Free account exceeds a configured hard message quota.
func (e *Enforcer) Allow(ctx context.Context, userID string, action Action) error {
	if e == nil || !e.cfg.Hosted {
		return nil
	}
	if e.reader == nil {
		return fmt.Errorf("plan enforcer reader is nil")
	}

	messages := action.Messages
	if messages <= 0 {
		messages = 1
	}

	entitlement, err := e.reader.GetEntitlement(ctx, userID)
	if err != nil {
		return err
	}
	effectivePlan := effectivePlan(entitlement, e.now())

	now := e.now().UTC()
	monthStart, monthEnd := monthBoundsMs(now)
	monthlyUsage, err := e.reader.GetUsage(ctx, userID, monthStart, monthEnd)
	if err != nil {
		return err
	}

	if effectivePlan == core.PlanHosted {
		if e.cfg.HostedFairUseMessagesMonth > 0 && monthlyUsage.Messages+messages > e.cfg.HostedFairUseMessagesMonth {
			e.log.Warn(
				"hosted fair-use message threshold exceeded",
				"user_id", userID,
				"messages_month", monthlyUsage.Messages,
				"action_messages", messages,
				"threshold", e.cfg.HostedFairUseMessagesMonth,
			)
		}
		return nil
	}

	if e.cfg.FreeMaxMessagesMonth > 0 && monthlyUsage.Messages+messages > e.cfg.FreeMaxMessagesMonth {
		return ErrQuotaExceeded
	}

	dayStart, dayEnd := dayBoundsMs(now)
	dailyUsage, err := e.reader.GetUsage(ctx, userID, dayStart, dayEnd)
	if err != nil {
		return err
	}
	if e.cfg.FreeMaxMessagesDay > 0 && dailyUsage.Messages+messages > e.cfg.FreeMaxMessagesDay {
		return ErrQuotaExceeded
	}

	return nil
}

// Service exposes read-only plan state for product surfaces.
type Service struct {
	cfg    Config
	reader Reader
	now    func() time.Time
}

// NewService creates a plan service.
func NewService(cfg Config, reader Reader) *Service {
	return &Service{
		cfg:    cfg,
		reader: reader,
		now:    time.Now,
	}
}

// GetUsage returns the current plan usage windows for a user.
func (s *Service) GetUsage(ctx context.Context, userID string) (*UsageResponse, error) {
	if s == nil || s.reader == nil {
		return nil, fmt.Errorf("plan service reader is nil")
	}

	entitlement, err := s.reader.GetEntitlement(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	effectivePlan := effectivePlan(entitlement, now)

	monthStart, monthEnd := monthBoundsMs(now)
	monthlyUsage, err := s.reader.GetUsage(ctx, userID, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	var daily *LimitUsage
	if s.hasDailyLimit(effectivePlan) {
		dayStart, dayEnd := dayBoundsMs(now)
		dailyUsage, err := s.reader.GetUsage(ctx, userID, dayStart, dayEnd)
		if err != nil {
			return nil, err
		}
		daily = &LimitUsage{
			Used:     dailyUsage.Messages,
			Limit:    s.cfg.FreeMaxMessagesDay,
			ResetsAt: time.UnixMilli(dayEnd + 1).UTC(),
		}
	}

	return &UsageResponse{
		Plan:    effectivePlan,
		Daily:   daily,
		Monthly: s.monthlyLimit(effectivePlan, monthlyUsage.Messages, monthEnd),
	}, nil
}

func (s *Service) hasDailyLimit(effectivePlan core.Plan) bool {
	return s.cfg.Hosted && effectivePlan != core.PlanHosted && s.cfg.FreeMaxMessagesDay > 0
}

func (s *Service) monthlyLimit(effectivePlan core.Plan, used int, resetEndMs int64) *LimitUsage {
	if !s.cfg.Hosted {
		return nil
	}

	limit := s.cfg.FreeMaxMessagesMonth
	if effectivePlan == core.PlanHosted {
		limit = s.cfg.HostedFairUseMessagesMonth
	}
	if limit <= 0 {
		return nil
	}

	return &LimitUsage{
		Used:     used,
		Limit:    limit,
		ResetsAt: time.UnixMilli(resetEndMs + 1).UTC(),
	}
}

func effectivePlan(entitlement *Entitlement, now time.Time) core.Plan {
	if entitlement == nil {
		return core.PlanFree
	}
	if entitlement.Plan != core.PlanHosted {
		return core.PlanFree
	}
	if entitlement.PlanExpiresAt != nil && now.UTC().After(time.UnixMilli(*entitlement.PlanExpiresAt).UTC()) {
		return core.PlanFree
	}
	return core.PlanHosted
}

func dayBoundsMs(t time.Time) (int64, int64) {
	start := time.Date(t.UTC().Year(), t.UTC().Month(), t.UTC().Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1).Add(-time.Millisecond)
	return start.UnixMilli(), end.UnixMilli()
}

func monthBoundsMs(t time.Time) (int64, int64) {
	start := time.Date(t.UTC().Year(), t.UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Millisecond)
	return start.UnixMilli(), end.UnixMilli()
}
