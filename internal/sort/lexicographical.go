package sort

import (
	"sort"

	"github.com/ataraskov/docker-hub-cleaner/internal/api"
)

// LexicographicalSorter sorts tags lexicographically (descending)
type LexicographicalSorter struct{}

// NewLexicographicalSorter creates a new lexicographical sorter
func NewLexicographicalSorter() *LexicographicalSorter {
	return &LexicographicalSorter{}
}

// Sort sorts tags lexicographically in descending order (newest first)
func (s *LexicographicalSorter) Sort(tags []api.Tag) []api.Tag {
	sorted := make([]api.Tag, len(tags))
	copy(sorted, tags)

	sort.Slice(sorted, func(i, j int) bool {
		// Descending order (newest first)
		return sorted[i].Name > sorted[j].Name
	})

	return sorted
}
