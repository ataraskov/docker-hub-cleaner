package policy

import "github.com/ataraskov/docker-hub-cleaner/internal/api"

// CountRetentionPolicy keeps the last X tags
type CountRetentionPolicy struct {
	keepSet map[string]bool
}

// NewCountRetentionPolicy creates a new count retention policy
// The sorted parameter should contain tags already sorted in the desired order
func NewCountRetentionPolicy(count int, sorted []api.Tag) *CountRetentionPolicy {
	keepSet := make(map[string]bool)

	// Keep the first 'count' tags from the sorted list
	for i := 0; i < min(count, len(sorted)); i++ {
		keepSet[sorted[i].Name] = true
	}

	return &CountRetentionPolicy{
		keepSet: keepSet,
	}
}

// ShouldKeep returns true if the tag is in the keep set
func (p *CountRetentionPolicy) ShouldKeep(tag api.Tag) bool {
	return p.keepSet[tag.Name]
}

// Name returns the policy name
func (p *CountRetentionPolicy) Name() string {
	return "count"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
