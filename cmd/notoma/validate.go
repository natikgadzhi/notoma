package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/lmittmann/tint"
	"github.com/natikgadzhi/notion-based/internal/config"
	"github.com/natikgadzhi/notion-based/internal/notion"
	"github.com/spf13/cobra"
)

var (
	validateConfigPath string
	validateVerbose    bool
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration and Notion connectivity",
	Long: `Validate checks that the configuration file is valid and that
notoma can connect to Notion and access the configured resources.

This command performs the following checks:
1. Config file exists and is valid YAML
2. All required config fields are present
3. NOTION_TOKEN environment variable is set
4. Notion API is accessible (validates token)
5. All configured roots are accessible
6. Workspace discovery works (if enabled)
7. Output vault path exists and is writable
8. State file directory exists or can be created`,
	RunE: runValidate,
}

func init() {
	validateCmd.Flags().StringVarP(&validateConfigPath, "config", "c", "config.yaml", "path to config file")
	validateCmd.Flags().BoolVarP(&validateVerbose, "verbose", "v", false, "enable verbose logging")
}

// ValidationResult holds the result of a single validation check.
type ValidationResult struct {
	Check   string
	Passed  bool
	Message string
}

// runValidate performs all validation checks and reports results.
func runValidate(cmd *cobra.Command, args []string) error {
	// Set up logging
	logLevel := slog.LevelInfo
	if validateVerbose {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("received shutdown signal, canceling...")
		cancel()
	}()

	// Collect all validation results
	var results []ValidationResult
	var hasErrors bool

	// Check 1: Config file exists
	logger.Debug("checking config file", "path", validateConfigPath)
	if _, err := os.Stat(validateConfigPath); err != nil {
		results = append(results, ValidationResult{
			Check:   "Config file exists",
			Passed:  false,
			Message: fmt.Sprintf("cannot access config file: %v", err),
		})
		printResults(cmd.OutOrStdout(), results)
		return fmt.Errorf("validation failed")
	}
	results = append(results, ValidationResult{
		Check:  "Config file exists",
		Passed: true,
	})

	// Check 2: Config file is valid and complete
	logger.Debug("loading configuration")
	cfg, err := config.Load(validateConfigPath)
	if err != nil {
		results = append(results, ValidationResult{
			Check:   "Config file valid",
			Passed:  false,
			Message: err.Error(),
		})
		printResults(cmd.OutOrStdout(), results)
		return fmt.Errorf("validation failed")
	}
	results = append(results, ValidationResult{
		Check:  "Config file valid",
		Passed: true,
	})

	// Check 3: Notion token is set (already validated in config.Load, but explicit check)
	if cfg.NotionToken == "" {
		results = append(results, ValidationResult{
			Check:   "NOTION_TOKEN set",
			Passed:  false,
			Message: "NOTION_TOKEN environment variable is not set",
		})
		hasErrors = true
	} else {
		results = append(results, ValidationResult{
			Check:  "NOTION_TOKEN set",
			Passed: true,
		})
	}

	// Check 4: Notion API connectivity
	logger.Debug("testing Notion API connectivity")
	client := notion.NewClient(cfg.NotionToken, logger)
	user, err := client.GetCurrentUser(ctx)
	if err != nil {
		results = append(results, ValidationResult{
			Check:   "Notion API accessible",
			Passed:  false,
			Message: fmt.Sprintf("failed to connect: %v", err),
		})
		hasErrors = true
	} else {
		results = append(results, ValidationResult{
			Check:   "Notion API accessible",
			Passed:  true,
			Message: fmt.Sprintf("connected as %q", user.Name),
		})
	}

	// Check 5: Configured roots are accessible (only if API is accessible)
	if user != nil && len(cfg.Sync.Roots) > 0 {
		logger.Debug("validating configured roots", "count", len(cfg.Sync.Roots))
		for _, root := range cfg.Sync.Roots {
			parsed, err := notion.ParseURL(root.URL)
			if err != nil {
				results = append(results, ValidationResult{
					Check:   fmt.Sprintf("Root %q URL valid", rootDisplayName(root)),
					Passed:  false,
					Message: fmt.Sprintf("invalid URL: %v", err),
				})
				hasErrors = true
				continue
			}

			resource, err := client.DetectResourceType(ctx, parsed.ID)
			if err != nil {
				results = append(results, ValidationResult{
					Check:   fmt.Sprintf("Root %q accessible", rootDisplayName(root)),
					Passed:  false,
					Message: err.Error(),
				})
				hasErrors = true
			} else {
				results = append(results, ValidationResult{
					Check:   fmt.Sprintf("Root %q accessible", rootDisplayName(root)),
					Passed:  true,
					Message: fmt.Sprintf("%s: %q", resource.Type, resource.Title),
				})
			}
		}
	}

	// Check 6: Workspace discovery works (if enabled and API is accessible)
	if user != nil && cfg.Sync.DiscoverWorkspaceRoots {
		logger.Debug("testing workspace discovery")
		roots, err := client.DiscoverWorkspaceRoots(ctx)
		if err != nil {
			results = append(results, ValidationResult{
				Check:   "Workspace discovery",
				Passed:  false,
				Message: fmt.Sprintf("failed: %v", err),
			})
			hasErrors = true
		} else {
			results = append(results, ValidationResult{
				Check:   "Workspace discovery",
				Passed:  true,
				Message: fmt.Sprintf("found %d root(s)", len(roots)),
			})
		}
	}

	// Check 7: Vault path exists and is writable
	logger.Debug("checking vault path", "path", cfg.Output.VaultPath)
	vaultPathValid, vaultPathMsg := checkVaultPath(cfg.Output.VaultPath)
	results = append(results, ValidationResult{
		Check:   "Vault path writable",
		Passed:  vaultPathValid,
		Message: vaultPathMsg,
	})
	if !vaultPathValid {
		hasErrors = true
	}

	// Check 8: State file directory exists or can be created
	logger.Debug("checking state file path", "path", cfg.State.File)
	statePathValid, statePathMsg := checkStatePath(cfg.State.File)
	results = append(results, ValidationResult{
		Check:   "State file path valid",
		Passed:  statePathValid,
		Message: statePathMsg,
	})
	if !statePathValid {
		hasErrors = true
	}

	// Print all results
	printResults(cmd.OutOrStdout(), results)

	if hasErrors {
		return fmt.Errorf("validation failed")
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nAll checks passed!")
	return nil
}

// rootDisplayName returns a display name for a root configuration.
func rootDisplayName(root config.Root) string {
	if root.Name != "" {
		return root.Name
	}
	// Return last part of URL for brevity
	if len(root.URL) > 30 {
		return "..." + root.URL[len(root.URL)-27:]
	}
	return root.URL
}

// checkVaultPath verifies the vault path exists and is writable.
func checkVaultPath(path string) (bool, string) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Sprintf("directory does not exist: %s", path)
		}
		return false, fmt.Sprintf("cannot access: %v", err)
	}

	if !info.IsDir() {
		return false, fmt.Sprintf("not a directory: %s", path)
	}

	// Check if writable by trying to create a temp file
	testFile := filepath.Join(path, ".notoma_write_test")
	f, err := os.Create(testFile)
	if err != nil {
		return false, fmt.Sprintf("directory not writable: %v", err)
	}
	_ = f.Close()
	_ = os.Remove(testFile)

	return true, ""
}

// checkStatePath verifies the state file directory exists or can be created.
func checkStatePath(path string) (bool, string) {
	dir := filepath.Dir(path)

	// If dir is "." (current directory), it always exists
	if dir == "." {
		return true, ""
	}

	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Try to create the directory
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return false, fmt.Sprintf("cannot create directory %s: %v", dir, err)
			}
			return true, fmt.Sprintf("created directory: %s", dir)
		}
		return false, fmt.Sprintf("cannot access directory: %v", err)
	}

	if !info.IsDir() {
		return false, fmt.Sprintf("parent path is not a directory: %s", dir)
	}

	return true, ""
}

// printResults outputs all validation results in a formatted way.
func printResults(w io.Writer, results []ValidationResult) {
	_, _ = fmt.Fprintln(w, "\nValidation Results:")
	_, _ = fmt.Fprintln(w, "-------------------")

	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}

		if r.Message != "" {
			_, _ = fmt.Fprintf(w, "[%s] %s: %s\n", status, r.Check, r.Message)
		} else {
			_, _ = fmt.Fprintf(w, "[%s] %s\n", status, r.Check)
		}
	}
}
