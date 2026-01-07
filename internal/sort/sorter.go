package sort

import "github.com/ataraskov/docker-hub-cleaner/internal/api"

// TagSorter defines the interface for sorting tags
type TagSorter interface {
	// Sort sorts tags and returns them in the desired order
	Sort(tags []api.Tag) []api.Tag
}
