// Package push defines shared BeeBuzz push protocol constants.
package push

const (
	// DefaultTopicName is used when no topic is explicitly provided.
	DefaultTopicName = "general"
	// PriorityHigh requests higher delivery urgency.
	PriorityHigh = "high"
	// PriorityNormal is the default delivery urgency.
	PriorityNormal = "normal"
	// PriorityHeader carries the desired urgency for octet-stream push requests.
	PriorityHeader = "X-Priority"
)

// ValidPriorities lists all accepted push priority values.
var ValidPriorities = []string{"", PriorityHigh, PriorityNormal}
