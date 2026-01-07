package cleaner

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ataraskov/docker-hub-cleaner/internal/api"
	"github.com/ataraskov/docker-hub-cleaner/internal/filter"
	"github.com/ataraskov/docker-hub-cleaner/internal/policy"
	sortpkg "github.com/ataraskov/docker-hub-cleaner/internal/sort"
)

// Cleaner orchestrates the tag cleaning process
type Cleaner struct {
	client  *api.Client
	filter  filter.TagFilter
	policy  policy.RetentionPolicy
	sorter  sortpkg.TagSorter
	dryRun  bool
	logger  *slog.Logger
	verbose bool
}

// Config holds the configuration for the cleaner
type Config struct {
	Client  *api.Client
	Filter  filter.TagFilter
	Policy  policy.RetentionPolicy
	Sorter  sortpkg.TagSorter
	DryRun  bool
	Logger  *slog.Logger
	Verbose bool
}

// NewCleaner creates a new cleaner instance
func NewCleaner(cfg Config) *Cleaner {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Cleaner{
		client:  cfg.Client,
		filter:  cfg.Filter,
		policy:  cfg.Policy,
		sorter:  cfg.Sorter,
		dryRun:  cfg.DryRun,
		logger:  cfg.Logger,
		verbose: cfg.Verbose,
	}
}

// CleanResult contains the results of a cleaning operation
type CleanResult struct {
	TotalTags     int
	FilteredTags  int
	KeptTags      int
	DeletedTags   []string
	Errors        []error
	TotalSize     int64
	ReclaimedSize int64
}

// Clean performs the tag cleaning operation
func (c *Cleaner) Clean(ctx context.Context, repo string) (*CleanResult, error) {
	result := &CleanResult{}

	// Step 1: Fetch all tags
	c.logger.Info("Fetching tags from repository", "repository", repo)
	tags, err := c.client.ListTags(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	result.TotalTags = len(tags)
	c.logger.Info("Fetched tags", "count", result.TotalTags)

	if result.TotalTags == 0 {
		c.logger.Info("No tags found in repository")
		return result, nil
	}

	// Calculate total size
	for _, tag := range tags {
		result.TotalSize += tag.FullSize
	}

	// Step 2: Apply filters
	if c.filter != nil {
		filtered := filter.FilterTags(tags, c.filter)
		result.FilteredTags = len(filtered)
		c.logger.Info("Applied filters", "matched", result.FilteredTags, "total", result.TotalTags)
		tags = filtered
	} else {
		result.FilteredTags = result.TotalTags
	}

	if len(tags) == 0 {
		c.logger.Info("No tags match the filter")
		return result, nil
	}

	// Step 3: Sort tags
	if c.sorter != nil {
		tags = c.sorter.Sort(tags)
		c.logger.Debug("Sorted tags", "count", len(tags))
	}

	// Step 4: Determine which tags to keep/delete
	var tagsToKeep, tagsToDelete []api.Tag
	for _, tag := range tags {
		if c.policy != nil && c.policy.ShouldKeep(tag) {
			tagsToKeep = append(tagsToKeep, tag)
		} else {
			tagsToDelete = append(tagsToDelete, tag)
		}
	}

	result.KeptTags = len(tagsToKeep)

	// Calculate reclaimed size
	for _, tag := range tagsToDelete {
		result.ReclaimedSize += tag.FullSize
	}

	if c.verbose {
		c.logger.Info("Retention analysis",
			"total_filtered", len(tags),
			"to_keep", len(tagsToKeep),
			"to_delete", len(tagsToDelete))

		if len(tagsToKeep) > 0 {
			c.logger.Debug("Tags to keep", "count", len(tagsToKeep))
			for _, tag := range tagsToKeep {
				c.logger.Debug("  Keep", "tag", tag.Name, "updated", tag.LastUpdated)
			}
		}
	}

	// Step 5: Delete tags (or report in dry-run mode)
	if len(tagsToDelete) == 0 {
		c.logger.Info("No tags to delete")
		return result, nil
	}

	if c.dryRun {
		c.logger.Info("DRY RUN: Would delete tags", "count", len(tagsToDelete))
		for _, tag := range tagsToDelete {
			result.DeletedTags = append(result.DeletedTags, tag.Name)
			c.logger.Info("  Would delete", "tag", tag.Name, "updated", tag.LastUpdated, "size", formatSize(tag.FullSize))
		}
	} else {
		c.logger.Info("Deleting tags", "count", len(tagsToDelete))
		for _, tag := range tagsToDelete {
			if err := c.client.DeleteTag(ctx, repo, tag.Name); err != nil {
				c.logger.Error("Failed to delete tag", "tag", tag.Name, "error", err)
				result.Errors = append(result.Errors, fmt.Errorf("failed to delete tag %s: %w", tag.Name, err))
			} else {
				result.DeletedTags = append(result.DeletedTags, tag.Name)
				c.logger.Info("  Deleted", "tag", tag.Name, "size", formatSize(tag.FullSize))
			}
		}
	}

	return result, nil
}

// formatSize formats a size in bytes to a human-readable string
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
