package policy

import (
	"strings"

	"github.com/ataraskov/docker-hub-cleaner/internal/api"
)

// PolicyMode defines how multiple policies are combined
type PolicyMode int

const (
	// PolicyModeOR keeps a tag if ANY policy says to keep it
	PolicyModeOR PolicyMode = iota
	// PolicyModeAND keeps a tag only if ALL policies say to keep it
	PolicyModeAND
)

// CompositePolicy combines multiple retention policies
type CompositePolicy struct {
	policies []RetentionPolicy
	mode     PolicyMode
}

// NewCompositePolicy creates a new composite policy
func NewCompositePolicy(mode PolicyMode, policies ...RetentionPolicy) *CompositePolicy {
	return &CompositePolicy{
		policies: policies,
		mode:     mode,
	}
}

// ShouldKeep returns true based on the policy mode
func (p *CompositePolicy) ShouldKeep(tag api.Tag) bool {
	if len(p.policies) == 0 {
		return true
	}

	switch p.mode {
	case PolicyModeOR:
		// Keep if ANY policy says to keep
		for _, policy := range p.policies {
			if policy.ShouldKeep(tag) {
				return true
			}
		}
		return false
	case PolicyModeAND:
		// Keep only if ALL policies say to keep
		for _, policy := range p.policies {
			if !policy.ShouldKeep(tag) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Name returns the policy name
func (p *CompositePolicy) Name() string {
	var names []string
	for _, policy := range p.policies {
		names = append(names, policy.Name())
	}

	mode := "OR"
	if p.mode == PolicyModeAND {
		mode = "AND"
	}

	return strings.Join(names, " "+mode+" ")
}
