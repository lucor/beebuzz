package core

// Plan represents a hosted BeeBuzz entitlement.
type Plan string

const (
	PlanFree   Plan = "free"
	PlanHosted Plan = "hosted"
)

// IsValid reports whether p is a recognised plan value.
func (p Plan) IsValid() bool {
	switch p {
	case PlanFree, PlanHosted:
		return true
	}
	return false
}
