package billing

import "testing"

func TestSubscriptionStatus(t *testing.T) {
	t.Run("valid statuses", func(t *testing.T) {
		for _, status := range []SubscriptionStatus{
			SubscriptionStatusIncomplete,
			SubscriptionStatusActive,
			SubscriptionStatusScheduled,
			SubscriptionStatusPastDue,
			SubscriptionStatusCanceled,
			SubscriptionStatusExpired,
		} {
			if !status.IsValid() {
				t.Fatalf("%q should be valid", status)
			}
		}
	})

	t.Run("active grants hosted plan", func(t *testing.T) {
		if !SubscriptionStatusActive.GrantsHostedPlan() {
			t.Fatal("active should grant hosted plan")
		}
		if !SubscriptionStatusPastDue.GrantsHostedPlan() {
			t.Fatal("past_due should grant hosted plan during grace period")
		}
		if !SubscriptionStatusScheduled.GrantsHostedPlan() {
			t.Fatal("scheduled_cancel should grant hosted plan until period end")
		}
	})
}
