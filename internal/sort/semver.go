package sort

import (
	"regexp"
	"sort"
	"strings"

	"github.com/ataraskov/docker-hub-cleaner/internal/api"
	"golang.org/x/mod/semver"
)

// SemverSorter sorts tags using semantic versioning
type SemverSorter struct {
	stripPrefixPattern *regexp.Regexp // optional: strip custom prefix before parsing
}

// NewSemverSorter creates a new semver sorter
func NewSemverSorter(stripPrefixPattern string) (*SemverSorter, error) {
	s := &SemverSorter{}

	if stripPrefixPattern != "" {
		re, err := regexp.Compile(stripPrefixPattern)
		if err != nil {
			return nil, err
		}
		s.stripPrefixPattern = re
	}

	return s, nil
}

// stripPrefix removes custom prefix if pattern is set
func (s *SemverSorter) stripPrefix(v string) string {
	if s.stripPrefixPattern != nil {
		return s.stripPrefixPattern.ReplaceAllString(v, "")
	}
	return v
}

// normalizeVersion adds "v" prefix if missing
func normalizeVersion(v string) string {
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

// Sort sorts tags using semantic version comparison
func (s *SemverSorter) Sort(tags []api.Tag) []api.Tag {
	var semverTags, nonSemverTags []api.Tag

	for _, tag := range tags {
		// First strip custom prefix (e.g., "develop-" from "develop-1.2.3")
		stripped := s.stripPrefix(tag.Name)
		// Then normalize with "v" prefix
		normalized := normalizeVersion(stripped)

		if semver.IsValid(normalized) {
			semverTags = append(semverTags, tag)
		} else {
			nonSemverTags = append(nonSemverTags, tag)
		}
	}

	// Sort semver tags using semver.Compare (descending - newest first)
	sort.Slice(semverTags, func(i, j int) bool {
		v1 := normalizeVersion(s.stripPrefix(semverTags[i].Name))
		v2 := normalizeVersion(s.stripPrefix(semverTags[j].Name))
		// Descending order: v2 < v1 means v1 comes first
		return semver.Compare(v1, v2) > 0
	})

	// Sort non-semver lexicographically (descending)
	sort.Slice(nonSemverTags, func(i, j int) bool {
		return nonSemverTags[i].Name > nonSemverTags[j].Name
	})

	// Return semver first, then non-semver
	return append(semverTags, nonSemverTags...)
}
