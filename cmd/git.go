package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/your-org/kargo-bootstrap/pkg/git"
)

var (
	// Git command flags
	repoURL      string
	branch       string
	tag          string
	commitHash   string
	shallow      bool
	authToken    string
	authUsername string
	authPassword string
	sshKeyPath   string
	sshKey       string
	knownHosts   string
	tempDir      string
	chartName    string
	chartVersion string
	chartPath    string
)

// gitCmd represents the git command
var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Test Git repository operations",
	Long: `Test Git repository operations like cloning, listing branches and tags,
and checking out specific revisions.`,
}

// gitCloneCmd represents the git clone command
var gitCloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone a Git repository",
	Long: `Clone a Git repository to a temporary directory.
This command tests the repository cloning functionality with various authentication methods.`,
	Example: `  # Clone a public repository
  kargo-bootstrap git clone --url https://github.com/example/repo.git

  # Clone a private repository with token
  kargo-bootstrap git clone --url https://github.com/example/private-repo.git --token YOUR_TOKEN

  # Clone a specific branch
  kargo-bootstrap git clone --url https://github.com/example/repo.git --branch develop

  # Clone with SSH key
  kargo-bootstrap git clone --url git@github.com:example/repo.git --ssh-key-path ~/.ssh/id_rsa`,
	Run: func(cmd *cobra.Command, args []string) {
		if repoURL == "" {
			fmt.Fprintf(os.Stderr, "Error: repository URL is required\n")
			os.Exit(1)
		}

		// Create authentication configuration
		auth := createAuthConfig()

		// Create clone options
		opts := &git.CloneOptions{
			URL:     repoURL,
			Branch:  branch,
			Tag:     tag,
			Commit:  commitHash,
			Shallow: shallow,
			Auth:    auth,
			TempDir: tempDir,
		}

		// Clone the repository
		fmt.Printf("Cloning repository: %s\n", repoURL)
		info, err := git.CloneRepository(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cloning repository: %v\n", err)
			os.Exit(1)
		}

		// Display repository information
		fmt.Println("✅ Repository cloned successfully!")
		fmt.Printf("Path: %s\n", info.Path)
		fmt.Printf("URL: %s\n", info.URL)
		fmt.Printf("Default branch: %s\n", info.DefaultBranch)
		fmt.Printf("Current branch: %s\n", info.CurrentBranch)
		fmt.Printf("Current commit: %s\n", info.CurrentCommit)

		// Clean up
		if err := git.CleanupTempDir(info.Path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean up temporary directory: %v\n", err)
		} else {
			fmt.Println("✅ Temporary directory cleaned up")
		}
	},
}

// gitListBranchesCmd represents the git list-branches command
var gitListBranchesCmd = &cobra.Command{
	Use:   "list-branches",
	Short: "List branches in a Git repository",
	Long: `List all available branches in a Git repository.
This command clones the repository temporarily and lists its branches.`,
	Example: `  # List branches in a public repository
  kargo-bootstrap git list-branches --url https://github.com/example/repo.git

  # List branches in a private repository
  kargo-bootstrap git list-branches --url https://github.com/example/private-repo.git --token YOUR_TOKEN`,
	Run: func(cmd *cobra.Command, args []string) {
		if repoURL == "" {
			fmt.Fprintf(os.Stderr, "Error: repository URL is required\n")
			os.Exit(1)
		}

		// Create authentication configuration
		auth := createAuthConfig()

		// Clone the repository temporarily
		opts := &git.CloneOptions{
			URL:     repoURL,
			Auth:    auth,
			TempDir: tempDir,
		}

		fmt.Printf("Cloning repository to list branches: %s\n", repoURL)
		info, err := git.CloneRepository(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cloning repository: %v\n", err)
			os.Exit(1)
		}
		defer git.CleanupTempDir(info.Path)

		// List branches
		branches, err := git.ListBranches(info.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing branches: %v\n", err)
			os.Exit(1)
		}

		// Display branches
		fmt.Printf("✅ Found %d branch(es):\n", len(branches))
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "BRANCH\t")
		fmt.Fprintln(w, "------\t")

		for _, branch := range branches {
			if branch == info.CurrentBranch {
				fmt.Fprintf(w, "* %s\t\n", branch) // Mark current branch
			} else {
				fmt.Fprintf(w, "%s\t\n", branch)
			}
		}

		w.Flush()
	},
}

// gitListTagsCmd represents the git list-tags command
var gitListTagsCmd = &cobra.Command{
	Use:   "list-tags",
	Short: "List tags in a Git repository",
	Long: `List all available tags in a Git repository.
This command clones the repository temporarily and lists its tags.`,
	Example: `  # List tags in a public repository
  kargo-bootstrap git list-tags --url https://github.com/example/repo.git

  # List tags in a private repository
  kargo-bootstrap git list-tags --url https://github.com/example/private-repo.git --token YOUR_TOKEN`,
	Run: func(cmd *cobra.Command, args []string) {
		if repoURL == "" {
			fmt.Fprintf(os.Stderr, "Error: repository URL is required\n")
			os.Exit(1)
		}

		// Create authentication configuration
		auth := createAuthConfig()

		// Clone the repository temporarily
		opts := &git.CloneOptions{
			URL:     repoURL,
			Auth:    auth,
			TempDir: tempDir,
		}

		fmt.Printf("Cloning repository to list tags: %s\n", repoURL)
		info, err := git.CloneRepository(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cloning repository: %v\n", err)
			os.Exit(1)
		}
		defer git.CleanupTempDir(info.Path)

		// List tags
		tags, err := git.ListTags(info.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing tags: %v\n", err)
			os.Exit(1)
		}

		// Display tags
		fmt.Printf("✅ Found %d tag(s):\n", len(tags))
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TAG\t")
		fmt.Fprintln(w, "---\t")

		for _, tag := range tags {
			fmt.Fprintf(w, "%s\t\n", tag)
		}

		w.Flush()
	},
}

// gitCheckoutCmd represents the git checkout command
var gitCheckoutCmd = &cobra.Command{
	Use:   "checkout",
	Short: "Test checking out a specific revision",
	Long: `Test checking out a specific branch, tag, or commit in a Git repository.
This command clones the repository and checks out the specified revision.`,
	Example: `  # Checkout a specific branch
  kargo-bootstrap git checkout --url https://github.com/example/repo.git --branch develop

  # Checkout a specific tag
  kargo-bootstrap git checkout --url https://github.com/example/repo.git --tag v1.0.0

  # Checkout a specific commit
  kargo-bootstrap git checkout --url https://github.com/example/repo.git --commit abc123`,
	Run: func(cmd *cobra.Command, args []string) {
		if repoURL == "" {
			fmt.Fprintf(os.Stderr, "Error: repository URL is required\n")
			os.Exit(1)
		}

		// Check that exactly one of branch, tag, or commit is specified
		revisions := []string{branch, tag, commitHash}
		nonEmpty := 0
		for _, rev := range revisions {
			if rev != "" {
				nonEmpty++
			}
		}

		if nonEmpty == 0 {
			fmt.Fprintf(os.Stderr, "Error: one of --branch, --tag, or --commit must be specified\n")
			os.Exit(1)
		}

		if nonEmpty > 1 {
			fmt.Fprintf(os.Stderr, "Error: only one of --branch, --tag, or --commit can be specified\n")
			os.Exit(1)
		}

		// Create authentication configuration
		auth := createAuthConfig()

		// Create clone options
		opts := &git.CloneOptions{
			URL:     repoURL,
			Branch:  branch,
			Tag:     tag,
			Commit:  commitHash,
			Auth:    auth,
			TempDir: tempDir,
		}

		// Clone the repository
		fmt.Printf("Cloning repository: %s\n", repoURL)
		info, err := git.CloneRepository(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cloning repository: %v\n", err)
			os.Exit(1)
		}
		defer git.CleanupTempDir(info.Path)

		// Display checkout information
		var revisionType string
		var revision string
		if branch != "" {
			revisionType = "branch"
			revision = branch
		} else if tag != "" {
			revisionType = "tag"
			revision = tag
		} else {
			revisionType = "commit"
			revision = commitHash
		}

		fmt.Printf("✅ Successfully checked out %s '%s'\n", revisionType, revision)
		fmt.Printf("Current branch: %s\n", info.CurrentBranch)
		fmt.Printf("Current commit: %s\n", info.CurrentCommit)
	},
}

// gitListChartsCmd represents the git list-charts command
var gitListChartsCmd = &cobra.Command{
	Use:   "list-charts",
	Short: "List Helm charts in a Git repository",
	Long: `List all Helm charts found in a Git repository.
This command clones the repository temporarily and scans for Chart.yaml files.`,
	Example: `  # List charts in a public repository
	 kargo-bootstrap git list-charts --url https://github.com/example/helm-charts.git

	 # List charts in a private repository
	 kargo-bootstrap git list-charts --url https://github.com/example/private-charts.git --token YOUR_TOKEN

	 # Filter charts by name
	 kargo-bootstrap git list-charts --url https://github.com/example/helm-charts.git --chart-name myapp`,
	Run: func(cmd *cobra.Command, args []string) {
		if repoURL == "" {
			fmt.Fprintf(os.Stderr, "Error: repository URL is required\n")
			os.Exit(1)
		}

		// Create authentication configuration
		auth := createAuthConfig()

		// Create clone options
		opts := &git.CloneOptions{
			URL:     repoURL,
			Branch:  branch,
			Tag:     tag,
			Commit:  commitHash,
			Auth:    auth,
			TempDir: tempDir,
		}

		fmt.Printf("Cloning repository to discover charts: %s\n", repoURL)
		charts, err := git.ListChartsInRepo(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error discovering charts: %v\n", err)
			os.Exit(1)
		}

		// Apply filters if specified
		if chartName != "" || chartVersion != "" {
			filter := &git.ChartFilter{
				Name:    chartName,
				Version: chartVersion,
			}
			charts = git.FilterCharts(charts, filter)
		}

		// Sort charts for consistent display
		git.SortCharts(charts)

		// Display charts
		if len(charts) == 0 {
			fmt.Println("⚠️  No charts found")
			return
		}

		fmt.Printf("✅ Found %d chart(s):\n", len(charts))
		fmt.Println()

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tAPP VERSION\tDESCRIPTION\tPATH\t")
		fmt.Fprintln(w, "----\t-------\t-----------\t-----------\t----\t")

		for _, chart := range charts {
			description := chart.Description
			if description == "" {
				description = "No description"
			}
			appVersion := chart.AppVersion
			if appVersion == "" {
				appVersion = "N/A"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t\n", chart.Name, chart.Version, appVersion, description, chart.Path)
		}

		w.Flush()
	},
}

// gitChartInfoCmd represents the git chart-info command
var gitChartInfoCmd = &cobra.Command{
	Use:   "chart-info",
	Short: "Show detailed information about a specific Helm chart",
	Long: `Show detailed information about a specific Helm chart in a Git repository.
This command clones the repository temporarily and displays detailed chart information.`,
	Example: `  # Show info for a chart at a specific path
	 kargo-bootstrap git chart-info --url https://github.com/example/helm-charts.git --chart-path ./myapp

	 # Show info for a chart by name (first match)
	 kargo-bootstrap git chart-info --url https://github.com/example/helm-charts.git --chart-name myapp`,
	Run: func(cmd *cobra.Command, args []string) {
		if repoURL == "" {
			fmt.Fprintf(os.Stderr, "Error: repository URL is required\n")
			os.Exit(1)
		}

		if chartPath == "" && chartName == "" {
			fmt.Fprintf(os.Stderr, "Error: either --chart-path or --chart-name must be specified\n")
			os.Exit(1)
		}

		// Create authentication configuration
		auth := createAuthConfig()

		// Create clone options
		opts := &git.CloneOptions{
			URL:     repoURL,
			Branch:  branch,
			Tag:     tag,
			Commit:  commitHash,
			Auth:    auth,
			TempDir: tempDir,
		}

		fmt.Printf("Cloning repository: %s\n", repoURL)
		repoInfo, err := git.CloneRepository(opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cloning repository: %v\n", err)
			os.Exit(1)
		}
		defer git.CleanupTempDir(repoInfo.Path)

		var chartInfo *git.ChartInfo

		// If chart path is specified, get info directly
		if chartPath != "" {
			chartInfo, err = git.GetChartInfo(repoInfo.Path, chartPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting chart info: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Find charts by name
			charts, err := git.ListChartsInRepo(opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error discovering charts: %v\n", err)
				os.Exit(1)
			}

			// Filter by name
			filter := &git.ChartFilter{
				Name: chartName,
			}
			filteredCharts := git.FilterCharts(charts, filter)

			if len(filteredCharts) == 0 {
				fmt.Printf("⚠️  No chart found with name: %s\n", chartName)
				os.Exit(1)
			}

			// Use the first match
			chartInfo = filteredCharts[0]
		}

		// Display chart information
		fmt.Printf("✅ Chart Information:\n\n")
		fmt.Printf("Name: %s\n", chartInfo.Name)
		fmt.Printf("Version: %s\n", chartInfo.Version)
		if chartInfo.AppVersion != "" {
			fmt.Printf("App Version: %s\n", chartInfo.AppVersion)
		}
		if chartInfo.Description != "" {
			fmt.Printf("Description: %s\n", chartInfo.Description)
		}
		if chartInfo.Home != "" {
			fmt.Printf("Home: %s\n", chartInfo.Home)
		}
		if chartInfo.Path != "" {
			fmt.Printf("Path: %s\n", chartInfo.Path)
		}

		if len(chartInfo.Sources) > 0 {
			fmt.Println("\nSources:")
			for _, source := range chartInfo.Sources {
				fmt.Printf("  - %s\n", source)
			}
		}

		if len(chartInfo.Keywords) > 0 {
			fmt.Println("\nKeywords:")
			for _, keyword := range chartInfo.Keywords {
				fmt.Printf("  - %s\n", keyword)
			}
		}

		if len(chartInfo.Maintainers) > 0 {
			fmt.Println("\nMaintainers:")
			for _, maintainer := range chartInfo.Maintainers {
				if maintainer.Email != "" {
					fmt.Printf("  - %s <%s>\n", maintainer.Name, maintainer.Email)
				} else {
					fmt.Printf("  - %s\n", maintainer.Name)
				}
			}
		}

		// Validate the chart
		if err := git.ValidateChart(repoInfo.Path, chartInfo.Path); err != nil {
			fmt.Printf("\n⚠️  Chart validation warning: %v\n", err)
		} else {
			fmt.Println("\n✅ Chart structure is valid")
		}
	},
}

// createAuthConfig creates an authentication configuration based on the provided flags
func createAuthConfig() *git.AuthConfig {
	if authToken != "" {
		return &git.AuthConfig{
			Method: git.AuthToken,
			Token:  authToken,
		}
	}

	if authUsername != "" && authPassword != "" {
		return &git.AuthConfig{
			Method:   git.AuthUserPass,
			Username: authUsername,
			Password: authPassword,
		}
	}

	if sshKeyPath != "" || sshKey != "" {
		return &git.AuthConfig{
			Method:     git.AuthSSHKey,
			SSHKey:     sshKey,
			KeyPath:    sshKeyPath,
			KnownHosts: knownHosts,
		}
	}

	// No authentication
	return nil
}

func init() {
	rootCmd.AddCommand(gitCmd)
	gitCmd.AddCommand(gitCloneCmd)
	gitCmd.AddCommand(gitListBranchesCmd)
	gitCmd.AddCommand(gitListTagsCmd)
	gitCmd.AddCommand(gitCheckoutCmd)
	gitCmd.AddCommand(gitListChartsCmd)
	gitCmd.AddCommand(gitChartInfoCmd)

	// Common flags for all git subcommands
	gitCmd.PersistentFlags().StringVar(&repoURL, "url", "", "Repository URL (required)")
	gitCmd.PersistentFlags().StringVar(&authToken, "token", "", "Authentication token for private repositories")
	gitCmd.PersistentFlags().StringVar(&authUsername, "username", "", "Username for authentication")
	gitCmd.PersistentFlags().StringVar(&authPassword, "password", "", "Password for authentication")
	gitCmd.PersistentFlags().StringVar(&sshKeyPath, "ssh-key-path", "", "Path to SSH private key")
	gitCmd.PersistentFlags().StringVar(&sshKey, "ssh-key", "", "SSH private key content")
	gitCmd.PersistentFlags().StringVar(&knownHosts, "known-hosts", "", "Path to known hosts file")
	gitCmd.PersistentFlags().StringVar(&tempDir, "temp-dir", "", "Temporary directory for cloning (default: auto-generated)")

	// Flags specific to clone command
	gitCloneCmd.Flags().StringVar(&branch, "branch", "", "Branch to clone")
	gitCloneCmd.Flags().StringVar(&tag, "tag", "", "Tag to clone")
	gitCloneCmd.Flags().StringVar(&commitHash, "commit", "", "Commit to checkout after cloning")
	gitCloneCmd.Flags().BoolVar(&shallow, "shallow", true, "Perform a shallow clone (default: true)")

	// Flags specific to checkout command
	gitCheckoutCmd.Flags().StringVar(&branch, "branch", "", "Branch to checkout")
	gitCheckoutCmd.Flags().StringVar(&tag, "tag", "", "Tag to checkout")
	gitCheckoutCmd.Flags().StringVar(&commitHash, "commit", "", "Commit to checkout")

	// Flags specific to list-charts command
	gitListChartsCmd.Flags().StringVar(&chartName, "chart-name", "", "Filter charts by name")
	gitListChartsCmd.Flags().StringVar(&chartVersion, "chart-version", "", "Filter charts by version")

	// Flags specific to chart-info command
	gitChartInfoCmd.Flags().StringVar(&chartPath, "chart-path", "", "Path to the chart within the repository")
	gitChartInfoCmd.Flags().StringVar(&chartName, "chart-name", "", "Name of the chart to show info for")
}
