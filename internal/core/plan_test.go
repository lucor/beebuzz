package core

import "testing"

func TestPlanIsValid(t *testing.T) {
	t.Run("valid plans", func(t *testing.T) {
		for _, plan := range []Plan{PlanFree, PlanHosted} {
			if !plan.IsValid() {
				t.Fatalf("%q should be valid", plan)
			}
		}
	})

	t.Run("invalid plan", func(t *testing.T) {
		if Plan("premium").IsValid() {
			t.Fatal("premium should be invalid")
		}
	})
}
