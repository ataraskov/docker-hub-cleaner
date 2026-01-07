package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ataraskov/docker-hub-cleaner/internal/api"
	"github.com/ataraskov/docker-hub-cleaner/internal/cleaner"
	"github.com/ataraskov/docker-hub-cleaner/internal/filter"
	"github.com/ataraskov/docker-hub-cleaner/internal/policy"
	sortpkg "github.com/ataraskov/docker-hub-cleaner/internal/sort"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information (injected at build time via ldflags)
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

var (
	// Authentication flags
	username   string
	password   string
	token      string
	repository string

	// Retention policy flags
	keepDays   int
	keepCount  int
	sortMethod string

	// Filtering flags
	tagPattern     string
	excludePattern string
	stripPrefix    string

	// Execution flags
	dryRun      bool
	verbose     bool
	concurrency int
)

var rootCmd = &cobra.Command{
	Use:   "docker-hub-cleaner",
	Short: "Clean up Docker Hub images based on retention policies",
	Long: `A CLI tool to manage Docker Hub images with retention policies.
Supports filtering by tags, retention by days or count, and dry-run mode.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, GitCommit, BuildTime),
	RunE:    run,
}

func init() {
	// Authentication flags
	rootCmd.Flags().StringVarP(&username, "username", "u", "", "Docker Hub username (or DOCKER_HUB_USERNAME env)")
	rootCmd.Flags().StringVarP(&password, "password", "p", "", "Docker Hub password (or DOCKER_HUB_PASSWORD env)")
	rootCmd.Flags().StringVarP(&token, "token", "t", "", "Personal Access Token (alternative to password)")
	rootCmd.Flags().StringVarP(&repository, "repository", "r", "", "Repository name (format: username/repo)")

	// Retention policy flags
	rootCmd.Flags().IntVar(&keepDays, "keep-days", 0, "Keep images created within X days")
	rootCmd.Flags().IntVar(&keepCount, "keep-count", 0, "Keep last X images")
	rootCmd.Flags().StringVar(&sortMethod, "sort-method", "lexicographical", "Sorting method: lexicographical or semver")

	// Filtering flags
	rootCmd.Flags().StringVar(&tagPattern, "tag-pattern", "", "Regex pattern for tags to include (e.g., ^dev-.*)")
	rootCmd.Flags().StringVar(&excludePattern, "exclude-pattern", "", "Regex pattern for tags to exclude")
	rootCmd.Flags().StringVar(&stripPrefix, "strip-prefix", "", "Regex pattern to strip from tag before semver parsing")

	// Execution flags
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report changes without deleting")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().IntVar(&concurrency, "concurrency", 5, "Number of concurrent API requests")

	// Mark required flags
	_ = rootCmd.MarkFlagRequired("repository")

	// Bind environment variables
	_ = viper.BindEnv("username", "DOCKER_HUB_USERNAME")
	_ = viper.BindEnv("password", "DOCKER_HUB_PASSWORD")
	_ = viper.BindEnv("token", "DOCKER_HUB_TOKEN")
}

func run(cmd *cobra.Command, args []string) error {
	// Setup logger
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Get credentials from flags or environment
	if username == "" {
		username = viper.GetString("username")
	}
	if password == "" {
		password = viper.GetString("password")
	}
	if token == "" {
		token = viper.GetString("token")
	}

	// Validate credentials
	if token == "" && (username == "" || password == "") {
		return fmt.Errorf("either --token or --username/--password must be provided")
	}

	// Validate repository format
	if repository == "" {
		return fmt.Errorf("--repository is required")
	}

	// Validate retention policies
	if keepDays == 0 && keepCount == 0 {
		return fmt.Errorf("at least one retention policy (--keep-days or --keep-count) must be specified")
	}

	// Create API client
	client := api.NewClient()

	// Authenticate
	ctx := context.Background()
	if token != "" {
		client.AuthenticateWithToken(token)
		logger.Info("Authenticated with token")
	} else {
		if err := client.Authenticate(ctx, username, password); err != nil {
			return fmt.Errorf("authentication failed: %w", err)
		}
		logger.Info("Authenticated", "username", username)
	}

	// Setup filter
	var tagFilter filter.TagFilter
	var filters []filter.TagFilter

	if tagPattern != "" {
		f, err := filter.NewRegexFilter(tagPattern, false)
		if err != nil {
			return fmt.Errorf("invalid tag pattern: %w", err)
		}
		filters = append(filters, f)
		logger.Info("Tag pattern filter enabled", "pattern", tagPattern)
	}

	if excludePattern != "" {
		f, err := filter.NewRegexFilter(excludePattern, true)
		if err != nil {
			return fmt.Errorf("invalid exclude pattern: %w", err)
		}
		filters = append(filters, f)
		logger.Info("Exclude pattern filter enabled", "pattern", excludePattern)
	}

	if len(filters) > 0 {
		tagFilter = filter.NewCompositeFilter(filters...)
	}

	// Setup sorter
	var sorter sortpkg.TagSorter
	switch sortMethod {
	case "lexicographical":
		sorter = sortpkg.NewLexicographicalSorter()
		logger.Info("Using lexicographical sorting")
	case "semver":
		s, err := sortpkg.NewSemverSorter(stripPrefix)
		if err != nil {
			return fmt.Errorf("invalid strip-prefix pattern: %w", err)
		}
		sorter = s
		logger.Info("Using semver sorting")
		if stripPrefix != "" {
			logger.Info("Strip prefix enabled", "pattern", stripPrefix)
		}
	default:
		return fmt.Errorf("invalid sort method: %s (must be 'lexicographical' or 'semver')", sortMethod)
	}

	// Fetch and sort tags first (needed for count policy)
	logger.Info("Fetching tags for policy evaluation", "repository", repository)
	allTags, err := client.ListTags(ctx, repository)
	if err != nil {
		return fmt.Errorf("failed to list tags: %w", err)
	}

	// Apply filters before sorting for count policy
	if tagFilter != nil {
		allTags = filter.FilterTags(allTags, tagFilter)
	}

	// Sort tags
	sortedTags := sorter.Sort(allTags)

	// Setup retention policy
	var policies []policy.RetentionPolicy

	if keepDays > 0 {
		policies = append(policies, policy.NewDaysRetentionPolicy(keepDays))
		logger.Info("Days retention policy enabled", "days", keepDays)
	}

	if keepCount > 0 {
		// Use sorted tags for count policy
		policies = append(policies, policy.NewCountRetentionPolicy(keepCount, sortedTags))
		logger.Info("Count retention policy enabled", "count", keepCount)
	}

	var retentionPolicy policy.RetentionPolicy
	if len(policies) == 1 {
		retentionPolicy = policies[0]
	} else {
		// Use OR mode: keep if ANY policy says to keep
		retentionPolicy = policy.NewCompositePolicy(policy.PolicyModeOR, policies...)
		logger.Info("Using OR policy mode (keep if ANY policy matches)")
	}

	// Create cleaner
	c := cleaner.NewCleaner(cleaner.Config{
		Client:  client,
		Filter:  tagFilter,
		Policy:  retentionPolicy,
		Sorter:  sorter,
		DryRun:  dryRun,
		Logger:  logger,
		Verbose: verbose,
	})

	// Run cleaner
	if dryRun {
		logger.Info("=== DRY RUN MODE - No tags will be deleted ===")
	}

	result, err := c.Clean(ctx, repository)
	if err != nil {
		return fmt.Errorf("cleaning failed: %w", err)
	}

	// Print summary
	fmt.Println("\n" + "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("SUMMARY")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Repository:       %s\n", repository)
	fmt.Printf("Total tags:       %d\n", result.TotalTags)
	fmt.Printf("After filtering:  %d\n", result.FilteredTags)
	fmt.Printf("Tags to keep:     %d\n", result.KeptTags)
	fmt.Printf("Tags %s:  %d\n", map[bool]string{true: "would delete", false: "deleted"}[dryRun], len(result.DeletedTags))

	if len(result.DeletedTags) > 0 {
		fmt.Printf("Disk space:       %s\n", formatSize(result.ReclaimedSize))
	}

	if len(result.Errors) > 0 {
		fmt.Printf("Errors:           %d\n", len(result.Errors))
		for _, err := range result.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	if dryRun && len(result.DeletedTags) > 0 {
		fmt.Println("\nRun without --dry-run to execute deletion.")
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return nil
}

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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
