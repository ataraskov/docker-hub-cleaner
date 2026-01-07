package filter

import (
	"fmt"
	"regexp"

	"github.com/ataraskov/docker-hub-cleaner/internal/api"
)

// TagFilter represents a filter for Docker image tags
type TagFilter interface {
	Matches(tag string) bool
}

// RegexFilter filters tags based on a regex pattern
type RegexFilter struct {
	pattern *regexp.Regexp
	invert  bool // if true, exclude matches instead of include
}

// NewRegexFilter creates a new regex filter
func NewRegexFilter(pattern string, invert bool) (*RegexFilter, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regex pattern: %w", err)
	}

	return &RegexFilter{
		pattern: re,
		invert:  invert,
	}, nil
}

// Matches returns true if the tag matches the filter criteria
func (f *RegexFilter) Matches(tag string) bool {
	matches := f.pattern.MatchString(tag)
	if f.invert {
		return !matches
	}
	return matches
}

// CompositeFilter combines multiple filters
type CompositeFilter struct {
	filters []TagFilter
}

// NewCompositeFilter creates a new composite filter
func NewCompositeFilter(filters ...TagFilter) *CompositeFilter {
	return &CompositeFilter{
		filters: filters,
	}
}

// Matches returns true if all filters match (AND logic)
func (f *CompositeFilter) Matches(tag string) bool {
	for _, filter := range f.filters {
		if !filter.Matches(tag) {
			return false
		}
	}
	return true
}

// FilterTags filters tags based on the provided filter
func FilterTags(tags []api.Tag, filter TagFilter) []api.Tag {
	if filter == nil {
		return tags
	}

	var filtered []api.Tag
	for _, tag := range tags {
		if filter.Matches(tag.Name) {
			filtered = append(filtered, tag)
		}
	}
	return filtered
}

// AlwaysMatchFilter is a filter that always matches
type AlwaysMatchFilter struct{}

// Matches always returns true
func (f *AlwaysMatchFilter) Matches(tag string) bool {
	return true
}
