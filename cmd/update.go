package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/getoptimum/mump2p-cli/internal/config"
	"github.com/getoptimum/mump2p-cli/internal/version"
	"github.com/spf13/cobra"
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

var (
	forceUpdate bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update CLI to the latest version",
	Long: `Update the mump2p CLI to the latest version from GitHub releases.
This command checks for updates, downloads the latest binary, and replaces the current installation.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Checking for updates...")

		currentVersion := config.Version
		if currentVersion == "" {
			currentVersion = "unknown"
		}
		fmt.Printf("Current version: %s\n", currentVersion)

		latestRelease, err := fetchLatestRelease()
		if err != nil {
			return fmt.Errorf("failed to fetch latest release: %w", err)
		}

		latestVersion := latestRelease.TagName
		fmt.Printf("Latest version: %s\n", latestVersion)

		if !forceUpdate && currentVersion == latestVersion {
			fmt.Println("✅ You are already on the latest version!")
			return nil
		}

		if !forceUpdate && currentVersion != "unknown" {
			if version.Compare(currentVersion, latestVersion) >= 0 {
				fmt.Println("✅ You are already on the latest version!")
				return nil
			}
		}

		fmt.Printf("Updating from %s to %s...\n", currentVersion, latestVersion)

		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to get executable path: %w", err)
		}

		actualPath, err := filepath.EvalSymlinks(execPath)
		if err != nil {
			actualPath = execPath
		}

		binaryName, err := getBinaryNameForOS()
		if err != nil {
			return err
		}

		downloadURL := ""
		for _, asset := range latestRelease.Assets {
			if asset.Name == binaryName {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}

		if downloadURL == "" {
			return fmt.Errorf("binary %s not found in release %s. Available assets: %v",
				binaryName, latestVersion, getAssetNames(latestRelease.Assets))
		}

		fmt.Printf("Downloading %s...\n", binaryName)

		tempFile, err := downloadBinary(downloadURL)
		if err != nil {
			return fmt.Errorf("failed to download binary: %w", err)
		}
		defer os.Remove(tempFile)

		fmt.Println("Verifying downloaded binary...")
		if err := verifyBinary(tempFile); err != nil {
			return fmt.Errorf("binary verification failed: %w", err)
		}

		fmt.Printf("Replacing binary at %s...\n", actualPath)
		if err := replaceBinary(tempFile, actualPath); err != nil {
			if os.IsPermission(err) || strings.Contains(err.Error(), "permission denied") {
				return fmt.Errorf("permission denied: unable to write to %s\n"+
					"Tip: You may need to run with sudo if the binary is in a system directory:\n"+
					"   sudo %s update", actualPath, execPath)
			}
			return fmt.Errorf("failed to replace binary: %w", err)
		}

		fmt.Println("✅ Verifying installation...")
		if err := verifyInstallation(actualPath); err != nil {
			return fmt.Errorf("installation verification failed: %w", err)
		}

		fmt.Printf("\n✅ Successfully updated to %s!\n", latestVersion)
		fmt.Printf("   Location: %s\n", actualPath)
		return nil
	},
}

func init() {
	updateCmd.Flags().BoolVar(&forceUpdate, "force", false, "Force update even if already on latest version")
	rootCmd.AddCommand(updateCmd)
}

func fetchLatestRelease() (*GitHubRelease, error) {
	url := "https://api.github.com/repos/getoptimum/mump2p-cli/releases/latest"

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "mump2p-cli-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w\n"+
			"Tip: Check your internet connection and try again", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d. Please try again later", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	if release.TagName == "" {
		return nil, fmt.Errorf("no tag name found in release")
	}

	return &release, nil
}

func getBinaryNameForOS() (string, error) {
	osName := runtime.GOOS
	switch osName {
	case "linux":
		return "mump2p-linux", nil
	case "darwin":
		return "mump2p-mac", nil
	default:
		return "", fmt.Errorf("unsupported OS: %s. Supported: Linux, macOS", osName)
	}
}

func downloadBinary(url string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	req, err := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create download request: %w", err)
	}
	
	req.Header.Set("User-Agent", "mump2p-cli-updater")
	
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "mump2p-update-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	if err := os.Chmod(tempFile.Name(), 0755); err != nil {
		os.Remove(tempFile.Name())
		return "", err
	}

	return tempFile.Name(), nil
}

func verifyBinary(binaryPath string) error {
	info, err := os.Stat(binaryPath)
	if err != nil {
		return err
	}

	if info.Mode()&0111 == 0 {
		return fmt.Errorf("downloaded file is not executable")
	}

	cmd := exec.Command(binaryPath, "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("binary verification failed: %w", err)
	}

	return nil
}

func replaceBinary(tempFile, targetPath string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("Windows is not supported")
	}

	targetDir := filepath.Dir(targetPath)
	if info, err := os.Stat(targetDir); err == nil {
		if info.Mode().Perm()&0200 == 0 {
			return fmt.Errorf("permission denied: directory %s is not writable", targetDir)
		}
	}

	backupPath := targetPath + ".backup"
	if err := copyFile(targetPath, backupPath); err != nil {
		_ = os.Remove(backupPath)
	}

	if err := os.Remove(targetPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove old binary: %w", err)
		}
	}

	if err := copyFile(tempFile, targetPath); err != nil {
		if backupPath != "" {
			if _, err := os.Stat(backupPath); err == nil {
				_ = copyFile(backupPath, targetPath)
			}
		}
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	if backupPath != "" {
		_ = os.Remove(backupPath)
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func verifyInstallation(binaryPath string) error {
	cmd := exec.Command(binaryPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to run version command: %w", err)
	}

	if !strings.Contains(string(output), "Version:") {
		return fmt.Errorf("unexpected version output: %s", string(output))
	}

	return nil
}

func getAssetNames(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}) []string {
	names := make([]string, len(assets))
	for i, asset := range assets {
		names[i] = asset.Name
	}
	return names
}
