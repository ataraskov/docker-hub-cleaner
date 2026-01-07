package policy

import "github.com/ataraskov/docker-hub-cleaner/internal/api"

// RetentionPolicy defines the interface for retention policies
type RetentionPolicy interface {
	// ShouldKeep returns true if the tag should be kept
	ShouldKeep(tag api.Tag) bool
	// Name returns the name of the policy
	Name() string
}
