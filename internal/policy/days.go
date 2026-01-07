package policy

import (
	"time"

	"github.com/ataraskov/docker-hub-cleaner/internal/api"
)

// DaysRetentionPolicy keeps tags created within X days
type DaysRetentionPolicy struct {
	days int
}

// NewDaysRetentionPolicy creates a new days retention policy
func NewDaysRetentionPolicy(days int) *DaysRetentionPolicy {
	return &DaysRetentionPolicy{
		days: days,
	}
}

// ShouldKeep returns true if the tag was created within the retention period
func (p *DaysRetentionPolicy) ShouldKeep(tag api.Tag) bool {
	cutoff := time.Now().AddDate(0, 0, -p.days)
	return tag.LastUpdated.After(cutoff)
}

// Name returns the policy name
func (p *DaysRetentionPolicy) Name() string {
	return "days"
}
