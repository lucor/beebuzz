package plan

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"go.beebuzz.app/beebuzz/internal/core"
)

type stubReader struct {
	entitlement *Entitlement
	usage       map[int64]*Usage
}

func (r *stubReader) GetEntitlement(context.Context, string) (*Entitlement, error) {
	return r.entitlement, nil
}

func (r *stubReader) GetUsage(_ context.Context, _ string, fromMs, _ int64) (*Usage, error) {
	if usage, ok := r.usage[fromMs]; ok {
		return usage, nil
	}
	return &Usage{}, nil
}

func TestEnforcerAllow(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	dayStart, _ := dayBoundsMs(now)
	monthStart, _ := monthBoundsMs(now)
	baseCfg := Config{
		Hosted:                     true,
		FreeMaxMessagesDay:         50,
		FreeMaxMessagesMonth:       500,
		HostedFairUseMessagesMonth: 100_000,
	}

	tests := []struct {
		name        string
		cfg         Config
		entitlement *Entitlement
		usage       map[int64]*Usage
		action      Action
		wantErr     error
	}{
		{
			name:    "self-hosted mode allows without reading quotas",
			cfg:     Config{Hosted: false, FreeMaxMessagesDay: 1, FreeMaxMessagesMonth: 1},
			action:  Action{Messages: 10},
			wantErr: nil,
		},
		{
			name:        "free under limits",
			cfg:         baseCfg,
			entitlement: &Entitlement{Plan: core.PlanFree},
			usage: map[int64]*Usage{
				dayStart:   {Messages: 10},
				monthStart: {Messages: 100},
			},
			action: Action{Messages: 1},
		},
		{
			name:        "free over daily limit",
			cfg:         baseCfg,
			entitlement: &Entitlement{Plan: core.PlanFree},
			usage: map[int64]*Usage{
				dayStart:   {Messages: 50},
				monthStart: {Messages: 100},
			},
			action:  Action{Messages: 1},
			wantErr: ErrQuotaExceeded,
		},
		{
			name:        "free over monthly limit",
			cfg:         baseCfg,
			entitlement: &Entitlement{Plan: core.PlanFree},
			usage: map[int64]*Usage{
				dayStart:   {Messages: 10},
				monthStart: {Messages: 500},
			},
			action:  Action{Messages: 1},
			wantErr: ErrQuotaExceeded,
		},
		{
			name:        "hosted over fair use does not block",
			cfg:         baseCfg,
			entitlement: &Entitlement{Plan: core.PlanHosted},
			usage: map[int64]*Usage{
				monthStart: {Messages: 100_000},
			},
			action: Action{Messages: 1},
		},
		{
			name: "expired hosted is treated as free",
			cfg:  baseCfg,
			entitlement: &Entitlement{
				Plan:          core.PlanHosted,
				PlanExpiresAt: int64Ptr(now.Add(-time.Hour).UnixMilli()),
			},
			usage: map[int64]*Usage{
				dayStart:   {Messages: 50},
				monthStart: {Messages: 100},
			},
			action:  Action{Messages: 1},
			wantErr: ErrQuotaExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enforcer := NewEnforcer(tt.cfg, &stubReader{entitlement: tt.entitlement, usage: tt.usage}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			enforcer.now = func() time.Time { return now }

			err := enforcer.Allow(context.Background(), "user_1", tt.action)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Allow() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceGetUsage(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	dayStart, _ := dayBoundsMs(now)
	monthStart, _ := monthBoundsMs(now)
	cfg := Config{
		Hosted:                     true,
		FreeMaxMessagesDay:         50,
		FreeMaxMessagesMonth:       500,
		HostedFairUseMessagesMonth: 100_000,
	}

	t.Run("free includes daily and monthly limits", func(t *testing.T) {
		svc := NewService(cfg, &stubReader{
			entitlement: &Entitlement{Plan: core.PlanFree},
			usage: map[int64]*Usage{
				dayStart:   {Messages: 4},
				monthStart: {Messages: 124},
			},
		})
		svc.now = func() time.Time { return now }

		got, err := svc.GetUsage(context.Background(), "user_1")
		if err != nil {
			t.Fatalf("GetUsage() error = %v", err)
		}
		if got.Daily == nil {
			t.Fatal("daily usage = nil, want Free daily limit")
		}
		if got.Daily.Used != 4 || got.Daily.Limit != 50 {
			t.Fatalf("daily usage = %+v, want 4 of 50", got.Daily)
		}
		if got.Monthly == nil {
			t.Fatal("monthly usage = nil, want Free monthly limit")
		}
		if got.Monthly.Used != 124 || got.Monthly.Limit != 500 {
			t.Fatalf("monthly usage = %+v, want 124 of 500", got.Monthly)
		}
	})

	t.Run("hosted omits daily limit", func(t *testing.T) {
		svc := NewService(cfg, &stubReader{
			entitlement: &Entitlement{Plan: core.PlanHosted},
			usage: map[int64]*Usage{
				dayStart:   {Messages: 4},
				monthStart: {Messages: 1_240},
			},
		})
		svc.now = func() time.Time { return now }

		got, err := svc.GetUsage(context.Background(), "user_1")
		if err != nil {
			t.Fatalf("GetUsage() error = %v", err)
		}
		if got.Daily != nil {
			t.Fatalf("daily usage = %+v, want nil for Hosted", got.Daily)
		}
		if got.Monthly == nil {
			t.Fatal("monthly usage = nil, want Hosted monthly fair-use limit")
		}
		if got.Monthly.Used != 1_240 || got.Monthly.Limit != 100_000 {
			t.Fatalf("monthly usage = %+v, want 1240 of 100000", got.Monthly)
		}
	})
}

func int64Ptr(v int64) *int64 {
	return &v
}
