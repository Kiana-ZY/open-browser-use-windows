package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Kiana-ZY/open-browser-use-windows/internal/host"
	"github.com/Kiana-ZY/open-browser-use-windows/internal/wire"
	"github.com/spf13/cobra"
)

const version = "0.1.29"
const defaultChromeExtensionID = "bgjoihaepiejlfjinojjfgokghnodnhd"
const defaultCLISessionID = "obu-cli"
const defaultMCPSessionID = "obu-mcp"
const chromeWebStoreUpdateURL = "https://clients2.google.com/service/update2/crx"
const betaExtensionPublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAnBLT95WWVnHYH0pOBRH/eP+BWtlKVmLE/RHkERUTI2+PGDSQrbWVabmTw4CZ3yhjko04dijSX2Az8cnp65xh23Dh5mP5TCtiP9LexRFJokd8EsyeFdtKamMYr0hF1ZUc1/8ZpLnetAU65ZMB9VzHQBqpJWeUwuIvecgfRtGklDgJMjnvcq5J6pttZrzWrI/2B0BNufwsTQfEt7qLtDFPHXmUdtZfQbc2EfYFvkXLDAXicYviiocedrsAGIKUxpyQegobhUFL+tNLOuXKBpZlLFQn3xgm5CyGZwN6bueiV/S7reigVTKAMQ8BX0eacT22e8r0UzjsjkugeHOIonIvtQIDAQAB"
const browserChrome = "chrome"
const browserEdge = "edge"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && isNativeMessagingLaunch(args[0]) {
		return runHost(host.DefaultSocketDir(), "")
	}
	cmd := newRootCommand()
	cmd.SetArgs(args)
	return cmd.Execute()
}

func isNativeMessagingLaunch(arg string) bool {
	return strings.HasPrefix(arg, "chrome-extension://")
}

func newRootCommand() *cobra.Command {
	var showVersion bool
	root := &cobra.Command{
		Use:           "open-browser-use",
		Short:         "Open Browser Use native host and CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showVersion {
				fmt.Fprintln(cmd.OutOrStdout(), version)
				return nil
			}
			return renderStartupStatus(cmd.OutOrStdout())
		},
	}
	root.Flags().BoolVarP(&showVersion, "version", "v", false, "print version")
	root.AddCommand(
		newHostCommand(),
		newSetupCommand(),
		newManifestCommand(),
		newInstallManifestCommand(),
		newCallCommand(),
		newRunCommand(),
		newMCPCommand(),
		newOpenTabCommand(),
		newNavigateCommand(),
		newSimpleRPCCommand("ping", "ping", "Ping the browser backend"),
		newSimpleRPCCommand("info", "getInfo", "Print browser backend info"),
		newSimpleRPCCommand("tabs", "getTabs", "List session tabs"),
		newSimpleRPCCommand("user-tabs", "getUserTabs", "List user browser tabs"),
		newHistoryCommand(),
		newClaimTabCommand(),
		newFinalizeTabsCommand(),
		newNameSessionCommand(),
		newCdpCommand(),
		newPageInfoCommand(),
		newTextCommand(),
		newScreenshotCommand(),
		newWaitCommand(),
		newSnapshotCommand(),
		newClickCommand(),
		newFillCommand(),
		newMoveMouseCommand(),
		newWaitFileChooserCommand(),
		newSetFileChooserFilesCommand(),
		newSimpleRPCCommand("turn-ended", "turnEnded", "End the current browser turn"),
		newVersionCommand(),
	)
	return root
}

func newHostCommand() *cobra.Command {
	var socketDir string
	var socketPath string
	cmd := &cobra.Command{
		Use:   "host",
		Short: "Run the native messaging host relay",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runHost(socketDir, socketPath)
		},
	}
	cmd.Flags().StringVar(&socketDir, "socket-dir", host.DefaultSocketDir(), "directory for SDK relay registry")
	cmd.Flags().StringVar(&socketPath, "socket-path", "", "explicit SDK relay path")
	return cmd
}

func runHost(socketDir string, socketPath string) error {
	relay := host.NewRelay(host.Config{
		SocketDir:  socketDir,
		SocketPath: socketPath,
	}, os.Stdin, os.Stdout)
	return relay.Serve(context.Background())
}

func newSetupCommand() *cobra.Command {
	extensionID := defaultChromeExtensionID
	var binaryPath string
	var externalExtensionOutput string
	var browser string
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Register browser integration for Open Browser Use",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := setupChrome(browser, extensionID, binaryPath, externalExtensionOutput)
			if err != nil {
				return err
			}
			status := detectBrowserExtension(browser, host.DefaultSocketDir(), 700*time.Millisecond)
			return renderStoreSetupResult(cmd.OutOrStdout(), result, status)
		},
	}
	cmd.Flags().StringVar(&extensionID, "extension-id", defaultChromeExtensionID, "extension id for allowed_origins")
	cmd.Flags().StringVar(&binaryPath, "path", "", "native host binary target for the stable host link")
	cmd.Flags().StringVar(&externalExtensionOutput, "external-extension-output", "", "external extension JSON output path")
	cmd.Flags().StringVar(&browser, "browser", browserChrome, "browser to configure (chrome or edge)")
	cmd.AddCommand(newSetupBetaCommand())
	return cmd
}

func newSetupBetaCommand() *cobra.Command {
	extensionID := defaultChromeExtensionID
	var binaryPath string
	var zipPath string
	var noOpen bool
	var browser string
	cmd := &cobra.Command{
		Use:   "beta",
		Short: "Register the native host and prepare the beta extension package",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			resolvedZIPPath := zipPath
			if resolvedZIPPath == "" {
				var err error
				resolvedZIPPath, err = downloadLatestReleaseZIP()
				if err != nil {
					return err
				}
			} else {
				var err error
				resolvedZIPPath, err = resolveExistingExtensionZIP(resolvedZIPPath)
				if err != nil {
					return err
				}
			}
			unpackedPath, unpackedExtensionID, err := installUnpackedExtension(resolvedZIPPath)
			if err != nil {
				return err
			}
			installZIPPath, err := writeBetaInstallZIP(unpackedPath, resolvedZIPPath)
			if err != nil {
				return err
			}
			effectiveExtensionID := extensionID
			if !cmd.Flags().Changed("extension-id") {
				effectiveExtensionID = unpackedExtensionID
			}
			if effectiveExtensionID != unpackedExtensionID {
				return fmt.Errorf("--extension-id %s does not match keyed beta ZIP extension id %s", effectiveExtensionID, unpackedExtensionID)
			}
			manifestPath, err := installNativeManifest(browser, effectiveExtensionID, binaryPath, "")
			if err != nil {
				return err
			}
			if !noOpen {
				if err := openBrowserExtensionsPage(browser); err != nil {
					return err
				}
				if err := revealFile(installZIPPath); err != nil {
					return err
				}
			}
			status := detectBrowserExtension(browser, host.DefaultSocketDir(), 700*time.Millisecond)
			status.InstallCommand = "open-browser-use setup beta"
			status.UpgradeCommand = "open-browser-use setup beta"
			return renderManualSetupResult(cmd.OutOrStdout(), manualSetupResult{
				NativeManifestPath: manifestPath,
				ExtensionID:        effectiveExtensionID,
				ZIPPath:            installZIPPath,
				UnpackedPath:       unpackedPath,
				OpenedBrowser:      !noOpen,
				OpenedFileManager:  !noOpen,
				Browser:            browser,
			}, status)
		},
	}
	cmd.Flags().StringVar(&extensionID, "extension-id", defaultChromeExtensionID, "extension id for allowed_origins")
	cmd.Flags().StringVar(&binaryPath, "path", "", "native host binary target for the stable host link")
	cmd.Flags().StringVar(&zipPath, "zip", "", "existing extension zip path; defaults to the latest GitHub Release zip")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "download and unpack the extension without opening the browser")
	cmd.Flags().StringVar(&browser, "browser", browserChrome, "browser to configure (chrome or edge)")
	return cmd
}

func newManifestCommand() *cobra.Command {
	extensionID := defaultChromeExtensionID
	var hostPath string
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Print a Chrome native messaging host manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			manifest, err := nativeManifest(extensionID, hostPath)
			if err != nil {
				return err
			}
			return writeJSONTo(cmd.OutOrStdout(), manifest)
		},
	}
	cmd.Flags().StringVar(&extensionID, "extension-id", defaultChromeExtensionID, "Chrome extension id for allowed_origins")
	cmd.Flags().StringVar(&hostPath, "path", "", "native host path written to manifest; defaults to the stable host link")
	return cmd
}

func newInstallManifestCommand() *cobra.Command {
	extensionID := defaultChromeExtensionID
	var binaryPath string
	var outputPath string
	var browser string
	cmd := &cobra.Command{
		Use:   "install-manifest",
		Short: "Install the native messaging host manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := installNativeManifest(browser, extensionID, binaryPath, outputPath)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
	cmd.Flags().StringVar(&extensionID, "extension-id", defaultChromeExtensionID, "extension id for allowed_origins")
	cmd.Flags().StringVar(&binaryPath, "path", "", "native host binary target for the stable host link")
	cmd.Flags().StringVar(&outputPath, "output", "", "native host manifest output path")
	cmd.Flags().StringVar(&browser, "browser", browserChrome, "browser to configure (chrome or edge)")
	return cmd
}

func installNativeManifest(browser string, extensionID string, binaryPath string, outputPath string) (string, error) {
	targetPath, err := resolveNativeHostTarget(binaryPath)
	if err != nil {
		return "", err
	}
	hostPath, err := stableNativeHostPath()
	if err != nil {
		return "", err
	}
	if err := installStableNativeHostLink(targetPath, hostPath); err != nil {
		return "", err
	}
	manifest, err := nativeManifest(extensionID, hostPath)
	if err != nil {
		return "", err
	}
	path := outputPath
	if path == "" {
		path, err = defaultNativeHostManifestPath(browser)
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		if err := ensureWindowsRegistryKey(path, browser); err != nil {
			return "", fmt.Errorf("manifest installed but failed to create registry key: %w", err)
		}
	}
	return path, nil
}

func ensureWindowsRegistryKey(manifestPath string, browser string) error {
	var regPath string
	switch browser {
	case browserEdge:
		regPath = `HKCU\Software\Microsoft\Edge\NativeMessagingHosts\` + host.NativeHostName
	case browserChrome:
		regPath = `HKCU\Software\Google\Chrome\NativeMessagingHosts\` + host.NativeHostName
	default:
		return nil
	}
	cmd := exec.Command("reg", "add", regPath, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("reg add %s: %w", regPath, err)
	}
	return nil
}

type setupResult struct {
	NativeManifestPath    string
	ExternalExtensionPath string
}

type manualSetupResult struct {
	NativeManifestPath string
	ExtensionID        string
	ZIPPath            string
	UnpackedPath       string
	OpenedBrowser      bool
	OpenedFileManager  bool
	Browser            string
}

type browserExtensionStatus struct {
	Installed       bool
	Reachable       bool
	ExtensionID     string
	Version         string
	VersionSource   string
	ExpectedVersion string
	InstallCommand  string
	UpgradeCommand  string
	Error           string
}

func setupChrome(browser string, extensionID string, binaryPath string, externalExtensionOutput string) (setupResult, error) {
	manifestPath, err := installNativeManifest(browser, extensionID, binaryPath, "")
	if err != nil {
		return setupResult{}, err
	}
	extensionPath, err := installChromeExternalExtension(browser, extensionID, externalExtensionOutput)
	if err != nil {
		return setupResult{}, err
	}
	return setupResult{
		NativeManifestPath:    manifestPath,
		ExternalExtensionPath: extensionPath,
	}, nil
}

func renderStartupStatus(writer io.Writer) error {
	status := detectBrowserExtension("", host.DefaultSocketDir(), 700*time.Millisecond)
	fmt.Fprintln(writer, "Open Browser Use")
	fmt.Fprintf(writer, "📦 CLI version: %s\n", version)
	fmt.Fprintf(writer, "🧩 Browser extension: %s\n", status.summary())
	fmt.Fprintln(writer)
	fmt.Fprintln(writer, "Next steps:")
	if status.needsInstall() {
		fmt.Fprintf(writer, "  1. Install the browser extension: %s\n", status.InstallCommand)
		fmt.Fprintln(writer, "     If the Chrome Web Store item is unavailable, use: open-browser-use setup beta")
		fmt.Fprintln(writer, "  2. Restart Chrome if it asks you to enable the extension.")
		fmt.Fprintln(writer, "  3. Verify the connection: open-browser-use info")
		return nil
	}
	if status.needsUpgrade() {
		fmt.Fprintf(writer, "  1. Upgrade the browser extension: %s\n", status.UpgradeCommand)
		fmt.Fprintln(writer, "  2. Restart Chrome if it asks you to reload the extension.")
		fmt.Fprintln(writer, "  3. Verify the connection: open-browser-use info")
		return nil
	}
	if !status.Reachable {
		fmt.Fprintln(writer, "  1. Open Chrome and make sure the Open Browser Use extension is enabled.")
		fmt.Fprintln(writer, "  2. Verify the connection: open-browser-use info")
		return nil
	}
	fmt.Fprintln(writer, "  1. You are ready. Try: open-browser-use user-tabs")
	return nil
}

func renderStoreSetupResult(writer io.Writer, result setupResult, status browserExtensionStatus) error {
	fmt.Fprintln(writer, "✅ Open Browser Use setup")
	fmt.Fprintf(writer, "1. ✅ Registered native host\n   %s\n", result.NativeManifestPath)
	fmt.Fprintf(writer, "2. ✅ Requested Chrome extension install\n   Extension id: %s\n   Chrome install file: %s\n", defaultChromeExtensionID, result.ExternalExtensionPath)
	fmt.Fprintf(writer, "3. 🧩 Browser extension\n   %s\n", status.summaryForSetup("Not installed yet. Chrome still needs to install or enable it."))
	fmt.Fprintln(writer)
	if status.isReady() {
		fmt.Fprintln(writer, "All set. The browser extension is installed, connected, and on the expected version.")
		return nil
	}
	if status.needsUpgrade() {
		fmt.Fprintf(writer, "Next: upgrade the browser extension with `%s`, then run `open-browser-use info`.\n", status.UpgradeCommand)
		return nil
	}
	fmt.Fprintln(writer, "Next:")
	fmt.Fprintln(writer, "  1. Restart Chrome if it is already open.")
	fmt.Fprintln(writer, "  2. Approve or enable the Open Browser Use extension if Chrome asks.")
	fmt.Fprintln(writer, "  3. Verify the connection: open-browser-use info")
	return nil
}

func renderManualSetupResult(writer io.Writer, result manualSetupResult, status browserExtensionStatus) error {
	extURL := "chrome://extensions"
	browserName := "Chrome"
	if result.Browser == browserEdge {
		extURL = "edge://extensions"
		browserName = "Edge"
	}
	fmt.Fprintln(writer, "✅ Open Browser Use beta setup")
	fmt.Fprintf(writer, "1. ✅ Registered native host\n   %s\n", result.NativeManifestPath)
	fmt.Fprintf(writer, "2. ✅ Prepared browser extension package\n   Extension id: %s\n   ZIP: %s\n   Includes stable extension key for this id.\n", result.ExtensionID, result.ZIPPath)
	fmt.Fprintf(writer, "3. ✅ Prepared unpacked extension directory\n   %s\n", result.UnpackedPath)
	fmt.Fprintf(writer, "4. 🧩 Browser extension\n   %s\n", status.summaryForSetup("Not installed yet. Finish the ZIP install below."))
	fmt.Fprintln(writer)
	if status.isReady() {
		fmt.Fprintln(writer, "All set. The browser extension is installed, connected, and on the expected version.")
		return nil
	}
	if result.OpenedBrowser {
		fmt.Fprintf(writer, "Opened %s for you.\n", extURL)
	} else {
		fmt.Fprintf(writer, "Open %s manually.\n", extURL)
	}
	if result.OpenedFileManager {
		fmt.Fprintln(writer, "Opened the extension package in Finder/file manager.")
	} else {
		fmt.Fprintf(writer, "Open the folder containing the package: %s\n", filepath.Dir(result.ZIPPath))
	}
	fmt.Fprintln(writer, "Next:")
	fmt.Fprintf(writer, "  1. Turn on Developer mode in %s.\n", extURL)
	fmt.Fprintf(writer, "  2. Drag the ZIP file into the %s extensions page to install it manually.\n", browserName)
	fmt.Fprintf(writer, "  3. Approve or enable the Open Browser Use extension if %s asks.\n", browserName)
	fmt.Fprintln(writer, "  4. Verify the connection: open-browser-use info")
	return nil
}

func detectBrowserExtension(browser string, socketDir string, timeout time.Duration) browserExtensionStatus {
	status := browserExtensionStatus{
		ExpectedVersion: version,
		InstallCommand:  "open-browser-use setup",
		UpgradeCommand:  "open-browser-use setup",
	}
	if response, err := invoke("", socketDir, "getInfo", map[string]any{}, timeout); err == nil {
		if result, ok := response["result"].(map[string]any); ok {
			status.Installed = true
			status.Reachable = true
			status.VersionSource = "connected extension"
			if extensionVersion, ok := result["version"].(string); ok {
				status.Version = extensionVersion
			}
			if metadata, ok := result["metadata"].(map[string]any); ok {
				if extensionID, ok := metadata["extensionId"].(string); ok {
					status.ExtensionID = extensionID
				}
			}
			return status
		}
	} else {
		status.Error = err.Error()
	}

	if browser == "" && runtime.GOOS == "windows" {
		for _, b := range []string{browserEdge, browserChrome} {
			if detected, ok := detectInstalledChromeExtension(b); ok {
				status.Installed = true
				status.ExtensionID = detected.ExtensionID
				status.Version = detected.Version
				status.VersionSource = detected.Source
				return status
			}
		}
	}
	if detected, ok := detectInstalledChromeExtension(browser); ok {
		status.Installed = true
		status.ExtensionID = detected.ExtensionID
		status.Version = detected.Version
		status.VersionSource = detected.Source
	}
	return status
}

type detectedExtension struct {
	ExtensionID string
	Version     string
	Source      string
}

func detectInstalledChromeExtension(browser string) (detectedExtension, bool) {
	candidates := []string{defaultChromeExtensionID}
	if betaID, err := extensionIDFromPublicKey(betaExtensionPublicKey); err == nil && betaID != defaultChromeExtensionID {
		candidates = append(candidates, betaID)
	}
	var best detectedExtension
	for _, extensionID := range candidates {
		detected, ok := detectInstalledChromeExtensionByID(browser, extensionID)
		if !ok {
			continue
		}
		if best.Version == "" || compareChromeVersions(detected.Version, best.Version) > 0 {
			best = detected
		}
	}
	return best, best.Version != ""
}

func detectInstalledChromeExtensionByID(browser string, extensionID string) (detectedExtension, bool) {
	root, err := defaultChromeUserDataDir(browser)
	if err != nil {
		return detectedExtension{}, false
	}
	profiles, err := chromeProfileDirs(root)
	if err != nil {
		return detectedExtension{}, false
	}
	var best detectedExtension
	for _, profile := range profiles {
		versionDirs, err := filepath.Glob(filepath.Join(profile, "Extensions", extensionID, "*"))
		if err != nil {
			continue
		}
		for _, versionDir := range versionDirs {
			manifestPath := filepath.Join(versionDir, "manifest.json")
			payload, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}
			var manifest map[string]any
			if err := json.Unmarshal(payload, &manifest); err != nil {
				continue
			}
			extensionVersion, ok := manifest["version"].(string)
			if !ok || strings.TrimSpace(extensionVersion) == "" {
				continue
			}
			detected := detectedExtension{
				ExtensionID: extensionID,
				Version:     extensionVersion,
				Source:      manifestPath,
			}
			if best.Version == "" || compareChromeVersions(detected.Version, best.Version) > 0 {
				best = detected
			}
		}
	}
	return best, best.Version != ""
}

func defaultChromeUserDataDir(browser string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library/Application Support/Google/Chrome"), nil
	case "linux":
		return filepath.Join(home, ".config/google-chrome"), nil
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		if browser == browserEdge {
			return filepath.Join(localAppData, "Microsoft", "Edge", "User Data"), nil
		}
		return filepath.Join(localAppData, "Google", "Chrome", "User Data"), nil
	default:
		return "", fmt.Errorf("Chrome extension detection is not implemented for %s", runtime.GOOS)
	}
}

func chromeProfileDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	profiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name == "Default" || strings.HasPrefix(name, "Profile ") {
			profiles = append(profiles, filepath.Join(root, name))
		}
	}
	return profiles, nil
}

func (status browserExtensionStatus) summary() string {
	if status.isReady() {
		return fmt.Sprintf("Ready, installed and connected at v%s.", status.Version)
	}
	if status.needsUpgrade() {
		return fmt.Sprintf("Installed v%s, but CLI expects v%s. Run `%s` to upgrade.", status.Version, status.ExpectedVersion, status.UpgradeCommand)
	}
	if status.Installed {
		if status.Version != "" {
			if compareChromeVersions(status.Version, status.ExpectedVersion) >= 0 {
				return fmt.Sprintf("Installed v%s and version matches CLI v%s. Open Chrome and run `open-browser-use info` to verify the connection.", status.Version, status.ExpectedVersion)
			}
			return fmt.Sprintf("Installed v%s, but not connected yet. Open Chrome and run `open-browser-use info`.", status.Version)
		}
		return "Installed, but not connected yet. Open Chrome and run `open-browser-use info`."
	}
	return fmt.Sprintf("Not installed yet. Run `%s`.", status.InstallCommand)
}

func (status browserExtensionStatus) summaryForSetup(missingMessage string) string {
	if status.needsInstall() {
		return missingMessage
	}
	return status.summary()
}

func (status browserExtensionStatus) isReady() bool {
	return status.Reachable && status.Installed && compareChromeVersions(status.Version, status.ExpectedVersion) >= 0
}

func (status browserExtensionStatus) needsInstall() bool {
	return !status.Installed
}

func (status browserExtensionStatus) needsUpgrade() bool {
	return status.Installed && status.Version != "" && compareChromeVersions(status.Version, status.ExpectedVersion) < 0
}

func compareChromeVersions(left string, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	maxParts := len(leftParts)
	if len(rightParts) > maxParts {
		maxParts = len(rightParts)
	}
	for i := 0; i < maxParts; i++ {
		leftValue := 0
		rightValue := 0
		if i < len(leftParts) {
			_, _ = fmt.Sscanf(leftParts[i], "%d", &leftValue)
		}
		if i < len(rightParts) {
			_, _ = fmt.Sscanf(rightParts[i], "%d", &rightValue)
		}
		if leftValue > rightValue {
			return 1
		}
		if leftValue < rightValue {
			return -1
		}
	}
	return 0
}

func installChromeExternalExtension(browser string, extensionID string, outputPath string) (string, error) {
	allowedExtensionID := strings.TrimSpace(extensionID)
	if allowedExtensionID == "" {
		allowedExtensionID = defaultChromeExtensionID
	}
	path := outputPath
	if path == "" {
		var err error
		path, err = defaultChromeExternalExtensionPath(browser, allowedExtensionID)
		if err != nil {
			return "", err
		}
	}
	manifest := map[string]any{
		"external_update_url": chromeWebStoreUpdateURL,
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("failed to create Chrome external extension directory %q: %w", filepath.Dir(path), err)
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("failed to write Chrome external extension manifest %q: %w", path, err)
	}
	return path, nil
}

type socketOptions struct {
	socketPath string
	socketDir  string
	timeout    time.Duration
	sessionID  string
	jsonOutput bool
	traceLog   string
}

func addSocketFlags(cmd *cobra.Command, options *socketOptions) {
	if options.sessionID == "" {
		if env := os.Getenv("OBU_SESSION_ID"); env != "" {
			options.sessionID = env
		} else {
			options.sessionID = defaultCLISessionID
		}
	}
	cmd.Flags().StringVar(&options.socketPath, "socket", "", "Unix socket or TCP relay path")
	cmd.Flags().StringVar(&options.socketDir, "socket-dir", host.DefaultSocketDir(), "directory containing active relay registry")
	cmd.Flags().DurationVar(&options.timeout, "timeout", 10*time.Second, "request timeout")
	cmd.Flags().StringVar(&options.sessionID, "session-id", options.sessionID, "browser session id (env: OBU_SESSION_ID)")
	cmd.Flags().BoolVar(&options.jsonOutput, "json", false, "output raw result value only, no JSON-RPC wrapper")
	cmd.Flags().StringVar(&options.traceLog, "trace-log", "", "write browser action trace JSONL to this file")
}

func newCallCommand() *cobra.Command {
	var options socketOptions
	var method string
	var params string
	cmd := &cobra.Command{
		Use:   "call",
		Short: "Send an unrestricted JSON-RPC request",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if method == "" {
				return errors.New("call requires --method")
			}
			var paramValue map[string]any
			if err := json.Unmarshal([]byte(params), &paramValue); err != nil {
				return err
			}
			return invokeAndWriteTo(cmd.OutOrStdout(), options, method, paramValue)
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().StringVar(&method, "method", "", "JSON-RPC method")
	cmd.Flags().StringVar(&params, "params", "{}", "JSON object params")
	return cmd
}

func newRunCommand() *cobra.Command {
	var options socketOptions
	var command string
	var file string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a line-oriented browser action plan",
		Long: `Run a line-oriented browser action plan.

The plan is not a general programming language. Each non-empty, non-comment line
is one supported browser action. Actions share the same session and turn, and
open-tab or claim-tab sets the default tab for later tab-scoped actions.

Example:
  open-browser-use run -c '
  name-session "Docs scan - OBU"
  open-tab https://docs.browser-use.com
  wait-load domcontentloaded
  page-info
  finalize-tabs []
  '`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if command != "" && file != "" {
				return errors.New("run accepts either --command or --file, not both")
			}
			script := command
			if file != "" {
				payload, err := os.ReadFile(file)
				if err != nil {
					return err
				}
				script = string(payload)
			}
			if strings.TrimSpace(script) == "" {
				return errors.New("run requires --command or --file")
			}
			runner := newActionRunner(options)
			output, err := runner.run(script)
			if err != nil {
				return err
			}
			return writeJSONTo(cmd.OutOrStdout(), output)
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().StringVarP(&command, "command", "c", "", "line-oriented action plan")
	cmd.Flags().StringVar(&file, "file", "", "file containing a line-oriented action plan")
	return cmd
}

func newSimpleRPCCommand(use string, method string, short string) *cobra.Command {
	var options socketOptions
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return invokeAndWriteTo(cmd.OutOrStdout(), options, method, map[string]any{})
		},
	}
	addSocketFlags(cmd, &options)
	return cmd
}

func newHistoryCommand() *cobra.Command {
	var options socketOptions
	var query string
	var limit int
	var from string
	var to string
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Search Chrome history",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			params := map[string]any{"query": query, "limit": limit}
			if from != "" {
				params["from"] = from
			}
			if to != "" {
				params["to"] = to
			}
			response, err := invokeWithOptions(options, "getUserHistory", params)
			if err != nil {
				return err
			}
			if options.jsonOutput {
				result, _ := response["result"].([]any)
				return writeJSONTo(cmd.OutOrStdout(), map[string]any{"items": result})
			}
			return writeJSONTo(cmd.OutOrStdout(), response)
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().StringVar(&query, "query", "", "history search query")
	cmd.Flags().IntVar(&limit, "limit", 100, "maximum result count")
	cmd.Flags().StringVar(&from, "from", "", "optional ISO start date")
	cmd.Flags().StringVar(&to, "to", "", "optional ISO end date")
	return cmd
}

func newClaimTabCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	cmd := &cobra.Command{
		Use:   "claim-tab",
		Short: "Claim an existing Chrome tab",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if tabID <= 0 {
				return errors.New("claim-tab requires --tab-id")
			}
			return invokeAndWriteTo(cmd.OutOrStdout(), options, "claimUserTab", map[string]any{"tabId": tabID})
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "Chrome tab id to claim")
	return cmd
}

func newFinalizeTabsCommand() *cobra.Command {
	var options socketOptions
	var keepJSON string
	cmd := &cobra.Command{
		Use:   "finalize-tabs",
		Short: "Finalize session tabs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var keep []any
			if err := json.Unmarshal([]byte(keepJSON), &keep); err != nil {
				return fmt.Errorf("finalize-tabs --keep must be a JSON array: %w", err)
			}
			return invokeAndWriteTo(cmd.OutOrStdout(), options, "finalizeTabs", map[string]any{"keep": keep})
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().StringVar(&keepJSON, "keep", "[]", `tabs to keep, e.g. '[{"tabId":123,"status":"handoff"}]'`)
	return cmd
}

func newNameSessionCommand() *cobra.Command {
	var options socketOptions
	var name string
	cmd := &cobra.Command{
		Use:   "name-session",
		Short: "Set the session tab group name",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if name == "" {
				return errors.New("name-session requires --name")
			}
			return invokeAndWriteTo(cmd.OutOrStdout(), options, "nameSession", map[string]any{"name": name})
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().StringVar(&name, "name", "", "session group title")
	return cmd
}

func newCdpCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	var method string
	var commandParamsJSON string
	cmd := &cobra.Command{
		Use:   "cdp",
		Short: "Run a Chrome DevTools Protocol command",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if method == "" {
				return errors.New("cdp requires --method")
			}
			var commandParams map[string]any
			if err := json.Unmarshal([]byte(commandParamsJSON), &commandParams); err != nil {
				return fmt.Errorf("cdp --params must be a JSON object: %w", err)
			}
			target := map[string]any{}
			if tabID > 0 {
				target["tabId"] = tabID
				_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
			}
			return invokeAndWriteTo(cmd.OutOrStdout(), options, "executeCdp", map[string]any{
				"target":        target,
				"method":        method,
				"commandParams": commandParams,
			})
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "optional tab id target")
	cmd.Flags().StringVar(&method, "method", "", "Chrome DevTools Protocol method")
	cmd.Flags().StringVar(&commandParamsJSON, "params", "{}", "Chrome DevTools Protocol command params")
	return cmd
}

func newTextCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	var selector string
	var maxChars int
	cmd := &cobra.Command{
		Use:   "text",
		Short: "Get page text content",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tabID = envTabID(tabID)
			if tabID <= 0 {
				return errors.New("text requires --tab-id or OBU_TAB_ID env var")
			}
			_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
			response, err := invokeWithOptions(options, "executeCdp", map[string]any{
				"target": map[string]any{"tabId": tabID},
				"method": "Runtime.evaluate",
				"commandParams": map[string]any{
					"expression":    pageTextJS(selector, maxChars),
					"returnByValue": true,
				},
			})
			if err != nil {
				return err
			}
			if options.jsonOutput {
				return writeJSONTo(cmd.OutOrStdout(), map[string]any{"text": runtimeEvaluateString(response)})
			}
			return writeJSONTo(cmd.OutOrStdout(), response)
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id (env: OBU_TAB_ID)")
	cmd.Flags().StringVar(&selector, "selector", "", "optional CSS selector to read")
	cmd.Flags().IntVar(&maxChars, "max-chars", 0, "maximum characters to return")
	return cmd
}

func newPageInfoCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	var selector string
	var maxChars int
	cmd := &cobra.Command{
		Use:   "page-info",
		Short: "Read page title, URL, readyState, and body text",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tabID = envTabID(tabID)
			if tabID <= 0 {
				return errors.New("page-info requires --tab-id or OBU_TAB_ID env var")
			}
			_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
			response, err := invokeWithOptions(options, "executeCdp", map[string]any{
				"target": map[string]any{"tabId": tabID},
				"method": "Runtime.evaluate",
				"commandParams": map[string]any{
					"expression":    pageInfoJS(selector, maxChars),
					"returnByValue": true,
				},
			})
			if err != nil {
				return err
			}
			if options.jsonOutput {
				result, _ := response["result"].(map[string]any)
				cdpResult, _ := result["result"].(map[string]any)
				return writeJSONTo(cmd.OutOrStdout(), cdpResult["value"])
			}
			return writeJSONTo(cmd.OutOrStdout(), response)
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id (env: OBU_TAB_ID)")
	cmd.Flags().StringVar(&selector, "selector", "", "optional CSS selector to read as text")
	cmd.Flags().IntVar(&maxChars, "max-chars", 0, "maximum text characters to return")
	return cmd
}

func newScreenshotCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	var output string
	var selector string
	var fullPage bool
	cmd := &cobra.Command{
		Use:   "screenshot",
		Short: "Take a page screenshot",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tabID = envTabID(tabID)
			if tabID <= 0 {
				return errors.New("screenshot requires --tab-id or OBU_TAB_ID env var")
			}
			result, err := captureScreenshot(options, tabID, output, selector, fullPage)
			if err != nil {
				return err
			}
			if options.jsonOutput {
				return writeJSONTo(cmd.OutOrStdout(), result)
			}
			fmt.Fprintln(cmd.OutOrStdout(), result["path"])
			return nil
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id (env: OBU_TAB_ID)")
	cmd.Flags().StringVar(&output, "output", "", "output file path")
	cmd.Flags().StringVar(&selector, "selector", "", "optional CSS selector to screenshot")
	cmd.Flags().BoolVar(&fullPage, "full-page", false, "capture the full page")
	return cmd
}

func newWaitCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	var state string
	cmd := &cobra.Command{
		Use:   "wait",
		Short: "Wait for page load state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tabID = envTabID(tabID)
			if tabID <= 0 {
				return errors.New("wait requires --tab-id or OBU_TAB_ID env var")
			}
			if state == "" {
				state = "load"
			}
			deadline := time.Now().Add(options.timeout)
			_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
			for {
				response, err := invokeWithOptions(options, "executeCdp", map[string]any{
					"target": map[string]any{"tabId": tabID},
					"method": "Runtime.evaluate",
					"commandParams": map[string]any{
						"expression":    "document.readyState",
						"returnByValue": true,
					},
				})
				if err != nil {
					return err
				}
				result, _ := response["result"].(map[string]any)
				cdpResult, _ := result["result"].(map[string]any)
				if val, ok := cdpResult["value"].(string); ok {
					if val == "complete" || (state == "domcontentloaded" && val == "interactive") {
						if options.jsonOutput {
							return writeJSONTo(cmd.OutOrStdout(), map[string]any{"readyState": val})
						}
						fmt.Fprintln(cmd.OutOrStdout(), val)
						return nil
					}
				}
				if time.Now().After(deadline) {
					return fmt.Errorf("timed out waiting for %s in tab %d", state, tabID)
				}
				time.Sleep(200 * time.Millisecond)
			}
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id (env: OBU_TAB_ID)")
	cmd.Flags().StringVar(&state, "state", "load", "load state: load or domcontentloaded")
	return cmd
}

func pageTextJS(selector string, maxChars int) string {
	selectorJSON, _ := json.Marshal(selector)
	return fmt.Sprintf(`(() => {
  const selector = %s;
  const root = selector ? document.querySelector(selector) : document.body;
  let text = root?.innerText ?? "";
  if (%d > 0 && text.length > %d) text = text.slice(0, %d);
  return text;
})()`, string(selectorJSON), maxChars, maxChars, maxChars)
}

func pageInfoJS(selector string, maxChars int) string {
	selectorJSON, _ := json.Marshal(selector)
	return fmt.Sprintf(`(() => {
  const selector = %s;
  const root = selector ? document.querySelector(selector) : document.body;
  let text = root?.innerText ?? "";
  if (%d > 0 && text.length > %d) text = text.slice(0, %d);
  return { title: document.title ?? "", url: location.href, readyState: document.readyState, text };
})()`, string(selectorJSON), maxChars, maxChars, maxChars)
}

func selectorClipJS(selector string) string {
	selectorJSON, _ := json.Marshal(selector)
	return fmt.Sprintf(`(() => {
  const selector = %s;
  const el = document.querySelector(selector);
  if (!el) return { ok: false, reason: "not-found", selector };
  el.scrollIntoView({ block: "center", inline: "center", behavior: "instant" });
  const rect = el.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) {
    return { ok: false, reason: "not-visible", selector, rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height } };
  }
  return {
    ok: true,
    selector,
    clip: {
      x: Math.max(0, rect.left + window.scrollX),
      y: Math.max(0, rect.top + window.scrollY),
      width: rect.width,
      height: rect.height,
      scale: 1
    }
  };
})()`, string(selectorJSON))
}

func fullPageClipJS() string {
	return `(() => {
  const doc = document.documentElement;
  const body = document.body;
  const width = Math.max(doc?.scrollWidth ?? 0, body?.scrollWidth ?? 0, doc?.clientWidth ?? 0, window.innerWidth);
  const height = Math.max(doc?.scrollHeight ?? 0, body?.scrollHeight ?? 0, doc?.clientHeight ?? 0, window.innerHeight);
  return { ok: true, clip: { x: 0, y: 0, width, height, scale: 1 } };
})()`
}

func captureScreenshot(options socketOptions, tabID int, output string, selector string, fullPage bool) (map[string]any, error) {
	if _, err := invokeWithOptions(options, "attach", map[string]any{"tabId": tabID}); err != nil {
		return nil, err
	}
	params := map[string]any{"format": "png"}
	var clip map[string]any
	if selector != "" {
		resolvedClip, err := resolveScreenshotClip(options, tabID, selectorClipJS(selector), "selector")
		if err != nil {
			return nil, err
		}
		clip = resolvedClip
		params["clip"] = clip
	} else if fullPage {
		resolvedClip, err := resolveScreenshotClip(options, tabID, fullPageClipJS(), "full-page")
		if err != nil {
			return nil, err
		}
		clip = resolvedClip
		params["clip"] = clip
	}
	response, err := invokeWithOptions(options, "executeCdp", map[string]any{
		"target":        map[string]any{"tabId": tabID},
		"method":        "Page.captureScreenshot",
		"commandParams": params,
	})
	if err != nil {
		return nil, err
	}
	result, _ := response["result"].(map[string]any)
	data, _ := result["data"].(string)
	if data == "" {
		return nil, errors.New("screenshot returned no data")
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	path := output
	if path == "" {
		path = fmt.Sprintf("screenshot-%d.png", time.Now().Unix())
	}
	if err := os.WriteFile(path, decoded, 0o644); err != nil {
		return nil, err
	}
	out := map[string]any{
		"path":   path,
		"bytes":  len(decoded),
		"format": "png",
		"tabId":  tabID,
	}
	if selector != "" {
		out["selector"] = selector
	}
	if clip != nil {
		out["clip"] = clip
	}
	return out, nil
}

func resolveScreenshotClip(options socketOptions, tabID int, expression string, label string) (map[string]any, error) {
	response, err := invokeWithOptions(options, "executeCdp", map[string]any{
		"target": map[string]any{"tabId": tabID},
		"method": "Runtime.evaluate",
		"commandParams": map[string]any{
			"expression":    expression,
			"returnByValue": true,
		},
	})
	if err != nil {
		return nil, err
	}
	value, err := runtimeEvaluateMap(response)
	if err != nil {
		return nil, err
	}
	ok, _ := value["ok"].(bool)
	if !ok {
		reason, _ := value["reason"].(string)
		if reason == "" {
			reason = "unknown"
		}
		return nil, fmt.Errorf("%s screenshot clip failed: %s", label, reason)
	}
	clip, ok := value["clip"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s screenshot clip returned no clip", label)
	}
	return clip, nil
}

func snapshotJS(limit int) string {
	if limit <= 0 {
		limit = 100
	}
	return fmt.Sprintf(`(() => {
  if (typeof window.__obu_refs !== 'undefined' && window.__obu_refs_limit === %d) return window.__obu_refs;
  const els = document.querySelectorAll('button, a, input, select, textarea, [role=button], [role=textbox], [role=combobox]');
  const results = [];
  let idx = 0;
  for (const el of els) {
    if (el.offsetParent === null && el.tagName !== 'SELECT') continue;
    const tag = el.tagName.toLowerCase();
    const text = (el.textContent || el.value || el.placeholder || el.getAttribute('aria-label') || '').trim().slice(0, 60);
    const id = el.id || '';
    const type = (el.type || '').toLowerCase();
    const href = el.href || '';
    const selector = id ? '#' + CSS.escape(id) : '';
    el.setAttribute('data-obu-ref', String(idx + 1));
    results.push({ index: idx + 1, tag, text, type, href, selector });
    idx++;
    if (idx >= %d) break;
  }
  window.__obu_refs = results;
  window.__obu_refs_limit = %d;
  return results;
})()`, limit, limit, limit)
}

func snapshotDir() string {
	return filepath.Join(os.TempDir(), "obu-snapshots")
}

func snapshotPath(sessionID string) string {
	return filepath.Join(snapshotDir(), sessionID+".json")
}

func saveSnapshot(sessionID string, data []map[string]any) error {
	dir := snapshotDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(snapshotPath(sessionID), payload, 0o600)
}

func loadSnapshot(sessionID string) ([]map[string]any, error) {
	payload, err := os.ReadFile(snapshotPath(sessionID))
	if err != nil {
		return nil, err
	}
	var data []map[string]any
	if err := json.Unmarshal(payload, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func snapshotItemsFromResponse(response map[string]any) ([]map[string]any, error) {
	result, _ := response["result"].(map[string]any)
	cdpResult, _ := result["result"].(map[string]any)
	value, _ := cdpResult["value"]
	elements, _ := value.([]any)
	if elements == nil {
		return nil, errors.New("snapshot returned no elements")
	}
	items := make([]map[string]any, 0, len(elements))
	for _, el := range elements {
		item, ok := el.(map[string]any)
		if !ok {
			return nil, errors.New("snapshot returned a non-object element")
		}
		items = append(items, item)
	}
	return items, nil
}

func interactionPreludeJS(ref string) string {
	refJSON, _ := json.Marshal(ref)
	return fmt.Sprintf(`const ref = %s;
  const el = [...document.querySelectorAll("[data-obu-ref]")].find((candidate) => candidate.getAttribute("data-obu-ref") === ref);
  const describe = (reason, extra = {}) => ({
    ok: false,
    ref,
    reason,
    tag: el?.tagName?.toLowerCase?.() ?? "",
    text: (el?.innerText || el?.value || el?.placeholder || el?.getAttribute?.("aria-label") || "").trim().slice(0, 120),
    ...extra,
  });
  if (!el) return { ok: false, ref, reason: "not-found" };
  if (el.disabled || el.getAttribute("aria-disabled") === "true") return describe("disabled");
  el.scrollIntoView({ block: "center", inline: "center", behavior: "instant" });
  const rect = el.getBoundingClientRect();
  const style = getComputedStyle(el);
  if (rect.width <= 0 || rect.height <= 0 || style.visibility === "hidden" || style.display === "none") {
    return describe("not-visible", { rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height } });
  }
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;`, string(refJSON))
}

func clickJS(ref string) string {
	return fmt.Sprintf(`(() => {
  %s
  const eventInit = { bubbles: true, cancelable: true, view: window, clientX: x, clientY: y };
  for (const type of ["pointerover", "mouseover", "pointermove", "mousemove", "pointerdown", "mousedown", "pointerup", "mouseup"]) {
    const EventClass = type.startsWith("pointer") && window.PointerEvent ? PointerEvent : MouseEvent;
    el.dispatchEvent(new EventClass(type, eventInit));
  }
  if (typeof el.click === "function") el.click();
  return {
    ok: true,
    ref,
    action: "click",
    tag: el.tagName.toLowerCase(),
    text: (el.innerText || el.value || el.placeholder || el.getAttribute("aria-label") || "").trim().slice(0, 120),
    rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
  };
})()`, interactionPreludeJS(ref))
}

func fillJS(ref string, text string) string {
	textJSON, _ := json.Marshal(text)
	return fmt.Sprintf(`(() => {
  %s
  const text = %s;
  el.focus();
  if (el.isContentEditable) {
    el.textContent = text;
  } else if ("value" in el) {
    const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype :
      el instanceof HTMLSelectElement ? HTMLSelectElement.prototype : HTMLInputElement.prototype;
    const descriptor = Object.getOwnPropertyDescriptor(proto, "value");
    if (descriptor?.set) {
      descriptor.set.call(el, text);
    } else {
      el.value = text;
    }
  } else {
    return describe("not-fillable");
  }
  el.dispatchEvent(new InputEvent("beforeinput", { bubbles: true, cancelable: true, inputType: "insertText", data: text }));
  el.dispatchEvent(new InputEvent("input", { bubbles: true, inputType: "insertText", data: text }));
  el.dispatchEvent(new Event("change", { bubbles: true }));
  return {
    ok: true,
    ref,
    action: "fill",
    tag: el.tagName.toLowerCase(),
    text: (el.innerText || el.value || el.placeholder || el.getAttribute("aria-label") || "").trim().slice(0, 120),
    valueLength: String(("value" in el ? el.value : el.textContent) ?? "").length,
    rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
  };
})()`, interactionPreludeJS(ref), string(textJSON))
}

func interactionResultFromResponse(action string, response map[string]any) (map[string]any, error) {
	result, err := runtimeEvaluateMap(response)
	if err != nil {
		return nil, err
	}
	ok, _ := result["ok"].(bool)
	if !ok {
		reason, _ := result["reason"].(string)
		if reason == "" {
			reason = "unknown"
		}
		return result, fmt.Errorf("%s failed: %s", action, reason)
	}
	return result, nil
}

func newSnapshotCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	var limit int
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "List interactive elements with ref indices (@1, @2, ...)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tabID = envTabID(tabID)
			if tabID <= 0 {
				return errors.New("snapshot requires --tab-id or OBU_TAB_ID env var")
			}
			_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
			response, err := invokeWithOptions(options, "executeCdp", map[string]any{
				"target":        map[string]any{"tabId": tabID},
				"method":        "Runtime.evaluate",
				"commandParams": map[string]any{"expression": snapshotJS(limit), "returnByValue": true},
			})
			if err != nil {
				return err
			}
			items, err := snapshotItemsFromResponse(response)
			if err != nil {
				return err
			}
			_ = saveSnapshot(options.sessionID, items)

			if options.jsonOutput {
				return writeJSONTo(cmd.OutOrStdout(), map[string]any{"items": items})
			}
			writer := cmd.OutOrStdout()
			for _, item := range items {
				idx := int(item["index"].(float64))
				fmt.Fprintf(writer, "@%d %s", idx, item["tag"])
				if t, _ := item["text"].(string); t != "" {
					fmt.Fprintf(writer, " %q", t)
				}
				fmt.Fprintln(writer)
			}
			return nil
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id (env: OBU_TAB_ID)")
	cmd.Flags().IntVar(&limit, "limit", 100, "maximum interactive elements to return")
	return cmd
}

func newClickCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	cmd := &cobra.Command{
		Use:   "click @N",
		Short: "Click an element by its snapshot ref (e.g. click @3)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabID = envTabID(tabID)
			if tabID <= 0 {
				return errors.New("click requires --tab-id or OBU_TAB_ID env var")
			}
			ref := strings.TrimPrefix(args[0], "@")
			_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
			response, err := invokeWithOptions(options, "executeCdp", map[string]any{
				"target": map[string]any{"tabId": tabID},
				"method": "Runtime.evaluate",
				"commandParams": map[string]any{
					"expression":    clickJS(ref),
					"returnByValue": true,
				},
			})
			if err != nil {
				return err
			}
			result, err := interactionResultFromResponse("click", response)
			if options.jsonOutput {
				if writeErr := writeJSONTo(cmd.OutOrStdout(), result); writeErr != nil && err == nil {
					return writeErr
				}
			}
			return err
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id (env: OBU_TAB_ID)")
	return cmd
}

func newFillCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	cmd := &cobra.Command{
		Use:   "fill @N TEXT",
		Short: "Type text into an input by its snapshot ref (e.g. fill @5 hello)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			tabID = envTabID(tabID)
			if tabID <= 0 {
				return errors.New("fill requires --tab-id or OBU_TAB_ID env var")
			}
			ref := strings.TrimPrefix(args[0], "@")
			text := strings.Join(args[1:], " ")
			_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
			response, err := invokeWithOptions(options, "executeCdp", map[string]any{
				"target": map[string]any{"tabId": tabID},
				"method": "Runtime.evaluate",
				"commandParams": map[string]any{
					"expression":    fillJS(ref, text),
					"returnByValue": true,
				},
			})
			if err != nil {
				return err
			}
			result, err := interactionResultFromResponse("fill", response)
			if options.jsonOutput {
				if writeErr := writeJSONTo(cmd.OutOrStdout(), result); writeErr != nil && err == nil {
					return writeErr
				}
			}
			return err
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id (env: OBU_TAB_ID)")
	return cmd
}

func newMoveMouseCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	var x float64
	var y float64
	var wait bool
	cmd := &cobra.Command{
		Use:   "move-mouse",
		Short: "Move the browser cursor overlay",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if tabID <= 0 {
				return errors.New("move-mouse requires --tab-id")
			}
			return invokeAndWriteTo(cmd.OutOrStdout(), options, "moveMouse", map[string]any{
				"tabId":          tabID,
				"x":              x,
				"y":              y,
				"waitForArrival": wait,
			})
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id to move the cursor in")
	cmd.Flags().Float64Var(&x, "x", 0, "viewport x coordinate")
	cmd.Flags().Float64Var(&y, "y", 0, "viewport y coordinate")
	cmd.Flags().BoolVar(&wait, "wait", true, "wait for cursor arrival acknowledgement")
	return cmd
}

func newWaitFileChooserCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	cmd := &cobra.Command{
		Use:   "wait-file-chooser",
		Short: "Wait for a browser session tab to open a file chooser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if tabID <= 0 {
				return errors.New("wait-file-chooser requires --tab-id")
			}
			return invokeAndWriteTo(cmd.OutOrStdout(), options, "waitForFileChooser", map[string]any{
				"tabId":     tabID,
				"timeoutMs": int(options.timeout.Milliseconds()),
			})
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id expected to open a file chooser")
	return cmd
}

func newSetFileChooserFilesCommand() *cobra.Command {
	var options socketOptions
	var fileChooserID string
	var files []string
	cmd := &cobra.Command{
		Use:   "set-file-chooser-files",
		Short: "Set files for an intercepted browser file chooser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if fileChooserID == "" {
				return errors.New("set-file-chooser-files requires --file-chooser-id")
			}
			if len(files) == 0 {
				return errors.New("set-file-chooser-files requires at least one --file")
			}
			return invokeAndWriteTo(cmd.OutOrStdout(), options, "setFileChooserFiles", map[string]any{
				"fileChooserId": fileChooserID,
				"files":         files,
			})
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().StringVar(&fileChooserID, "file-chooser-id", "", "file chooser id returned by wait-file-chooser")
	cmd.Flags().StringSliceVar(&files, "file", nil, "file path to set; repeat or comma-separate for multiple files")
	return cmd
}

func newOpenTabCommand() *cobra.Command {
	var options socketOptions
	var url string
	cmd := &cobra.Command{
		Use:   "open-tab",
		Short: "Open a new browser session tab",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			tabResponse, err := invokeWithOptions(options, "createTab", map[string]any{})
			if err != nil {
				return err
			}
			result, _ := tabResponse["result"].(map[string]any)
			tabID, ok := numberAsInt(result["id"])
			if !ok {
				return errors.New("createTab response did not include numeric tab id")
			}
			output := map[string]any{"tab": result}
			if url != "" {
				_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
				navigateResponse, err := invokeWithOptions(options, "executeCdp", map[string]any{
					"target":        map[string]any{"tabId": tabID},
					"method":        "Page.navigate",
					"commandParams": map[string]any{"url": url},
				})
				if err != nil {
					return err
				}
				output["navigate"] = navigateResponse["result"]
			}
			return writeJSONTo(cmd.OutOrStdout(), output)
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().StringVar(&url, "url", "", "optional URL to navigate after tab creation")
	return cmd
}

func newNavigateCommand() *cobra.Command {
	var options socketOptions
	var tabID int
	var url string
	cmd := &cobra.Command{
		Use:   "navigate",
		Short: "Navigate a browser session tab",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if tabID <= 0 {
				return errors.New("navigate requires --tab-id")
			}
			if url == "" {
				return errors.New("navigate requires --url")
			}
			_, _ = invokeWithOptions(options, "attach", map[string]any{"tabId": tabID})
			response, err := invokeWithOptions(options, "executeCdp", map[string]any{
				"target":        map[string]any{"tabId": tabID},
				"method":        "Page.navigate",
				"commandParams": map[string]any{"url": url},
			})
			if err != nil {
				return err
			}
			if options.jsonOutput {
				result, _ := response["result"].(map[string]any)
				return writeJSONTo(cmd.OutOrStdout(), map[string]any{"navigate": result})
			}
			return writeJSONTo(cmd.OutOrStdout(), response)
		},
	}
	addSocketFlags(cmd, &options)
	cmd.Flags().IntVar(&tabID, "tab-id", 0, "tab id to navigate")
	cmd.Flags().StringVar(&url, "url", "", "URL to navigate to")
	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		},
	}
}

type actionRunner struct {
	options      socketOptions
	sessionID    string
	turnID       string
	currentTabID int
}

type actionRunOutput struct {
	Steps        []actionStepOutput `json:"steps"`
	CurrentTabID int                `json:"currentTabId,omitempty"`
}

type actionStepOutput struct {
	Line     int            `json:"line"`
	Action   string         `json:"action"`
	Risk     string         `json:"risk"`
	TabID    int            `json:"tabId,omitempty"`
	Response map[string]any `json:"response"`
}

func newActionRunner(options socketOptions) *actionRunner {
	sessionID := options.sessionID
	if sessionID == "" {
		sessionID = defaultCLISessionID
	}
	return &actionRunner{
		options:   options,
		sessionID: sessionID,
		turnID:    fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano()),
	}
}

func (runner *actionRunner) run(script string) (actionRunOutput, error) {
	lines, err := parseActionScript(script)
	if err != nil {
		return actionRunOutput{}, err
	}
	output := actionRunOutput{Steps: []actionStepOutput{}}
	for _, line := range lines {
		step, err := runner.runActionLine(line)
		if err != nil {
			return actionRunOutput{}, fmt.Errorf("line %d: %w", line.number, err)
		}
		output.Steps = append(output.Steps, step)
	}
	output.CurrentTabID = runner.currentTabID
	return output, nil
}

func (runner *actionRunner) runActionLine(line actionLine) (actionStepOutput, error) {
	action := line.fields[0]
	args := line.fields[1:]
	startedAt := time.Now()
	response, tabID, err := runner.runAction(action, args)
	if err != nil {
		_ = runner.writeTrace(line.number, action, actionRisk(action), tabID, startedAt, err)
		return actionStepOutput{}, err
	}
	step := actionStepOutput{
		Line:     line.number,
		Action:   action,
		Risk:     actionRisk(action),
		TabID:    tabID,
		Response: response,
	}
	_ = runner.writeTrace(line.number, action, step.Risk, tabID, startedAt, nil)
	return step, nil
}

func (runner *actionRunner) runAction(action string, args []string) (map[string]any, int, error) {
	switch action {
	case "ping":
		return runner.invoke("ping", map[string]any{})
	case "info":
		return runner.invoke("getInfo", map[string]any{})
	case "tabs":
		return runner.invoke("getTabs", map[string]any{})
	case "user-tabs":
		return runner.invoke("getUserTabs", map[string]any{})
	case "turn-ended":
		return runner.invoke("turnEnded", map[string]any{})
	case "name-session":
		if len(args) == 0 {
			return nil, 0, errors.New("name-session requires a name")
		}
		return runner.invoke("nameSession", map[string]any{"name": strings.Join(args, " ")})
	case "open-tab":
		return runner.runOpenTabAction(args)
	case "claim-tab":
		tabID, err := firstIntArg(args, "--tab-id")
		if err != nil {
			return nil, 0, fmt.Errorf("claim-tab requires tab id: %w", err)
		}
		response, _, err := runner.invoke("claimUserTab", map[string]any{"tabId": tabID})
		if err == nil {
			runner.currentTabID = tabID
		}
		return response, tabID, err
	case "navigate":
		return runner.runNavigateAction(args)
	case "wait-load":
		return runner.runWaitLoadAction(args)
	case "page-info":
		return runner.runPageInfoAction(args)
	case "text":
		return runner.runTextAction(args)
	case "snapshot":
		return runner.runSnapshotAction(args)
	case "click":
		return runner.runClickAction(args)
	case "fill":
		return runner.runFillAction(args)
	case "screenshot":
		return runner.runScreenshotAction(args)
	case "cdp":
		return runner.runCDPAction(args)
	case "history":
		return runner.runHistoryAction(args)
	case "move-mouse":
		return runner.runMoveMouseAction(args)
	case "wait-file-chooser":
		return runner.runWaitFileChooserAction(args)
	case "set-file-chooser-files":
		return runner.runSetFileChooserFilesAction(args)
	case "finalize-tabs":
		return runner.runFinalizeTabsAction(args)
	case "call":
		return runner.runCallAction(args)
	default:
		return nil, 0, fmt.Errorf("unsupported action %q", action)
	}
}

func actionRisk(action string) string {
	switch action {
	case "click", "fill", "move-mouse":
		return "interaction"
	case "navigate", "open-tab", "claim-tab":
		return "navigation"
	case "set-file-chooser-files":
		return "file-system"
	case "wait-file-chooser", "screenshot", "snapshot", "text", "page-info", "history", "tabs", "user-tabs", "info", "ping", "wait-load":
		return "read"
	case "finalize-tabs", "turn-ended", "name-session":
		return "session"
	case "cdp", "call":
		return "unrestricted"
	default:
		return "unknown"
	}
}

func (runner *actionRunner) writeTrace(line int, action string, risk string, tabID int, startedAt time.Time, actionErr error) error {
	if runner.options.traceLog == "" {
		return nil
	}
	entry := map[string]any{
		"timestamp":  time.Now().UTC().Format(time.RFC3339Nano),
		"sessionId":  runner.sessionID,
		"turnId":     runner.turnID,
		"line":       line,
		"action":     action,
		"risk":       risk,
		"tabId":      tabID,
		"durationMs": time.Since(startedAt).Milliseconds(),
		"ok":         actionErr == nil,
	}
	if actionErr != nil {
		entry["error"] = actionErr.Error()
	}
	payload, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(runner.options.traceLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(payload, '\n'))
	return err
}

func (runner *actionRunner) runOpenTabAction(args []string) (map[string]any, int, error) {
	url := firstStringArg(args, "--url")
	response, _, err := runner.invoke("createTab", map[string]any{})
	if err != nil {
		return nil, 0, err
	}
	result, _ := response["result"].(map[string]any)
	tabID, ok := numberAsInt(result["id"])
	if !ok {
		return nil, 0, errors.New("createTab response did not include numeric tab id")
	}
	runner.currentTabID = tabID
	output := map[string]any{"tab": result}
	if url != "" {
		if err := runner.attach(tabID); err != nil {
			return nil, 0, err
		}
		navigate, _, err := runner.invoke("executeCdp", map[string]any{
			"target":        map[string]any{"tabId": tabID},
			"method":        "Page.navigate",
			"commandParams": map[string]any{"url": url},
		})
		if err != nil {
			return nil, 0, err
		}
		output["navigate"] = navigate["result"]
	}
	return map[string]any{"result": output}, tabID, nil
}

func (runner *actionRunner) runNavigateAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	url := firstStringArg(args, "--url")
	if url == "" {
		return nil, 0, errors.New("navigate requires a URL")
	}
	if err := runner.attach(tabID); err != nil {
		return nil, 0, err
	}
	return runner.invoke("executeCdp", map[string]any{
		"target":        map[string]any{"tabId": tabID},
		"method":        "Page.navigate",
		"commandParams": map[string]any{"url": url},
	})
}

func (runner *actionRunner) runWaitLoadAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	state := firstStringArg(args, "--state")
	if state == "" {
		state = "load"
	}
	if state != "load" && state != "domcontentloaded" {
		return nil, 0, fmt.Errorf("unsupported load state %q", state)
	}
	if err := runner.attach(tabID); err != nil {
		return nil, 0, err
	}
	if _, _, err := runner.invoke("executeCdp", map[string]any{
		"target":        map[string]any{"tabId": tabID},
		"method":        "Page.enable",
		"commandParams": map[string]any{},
	}); err != nil {
		return nil, 0, err
	}
	deadline := time.Now().Add(runner.options.timeout)
	for {
		response, _, err := runner.invoke("executeCdp", map[string]any{
			"target": map[string]any{"tabId": tabID},
			"method": "Runtime.evaluate",
			"commandParams": map[string]any{
				"expression":    "document.readyState",
				"returnByValue": true,
			},
		})
		if err != nil {
			return nil, 0, err
		}
		readyState := runtimeEvaluateString(response)
		if readyState == "complete" || (state == "domcontentloaded" && readyState == "interactive") {
			return map[string]any{"result": map[string]any{"readyState": readyState}}, tabID, nil
		}
		if time.Now().After(deadline) {
			return nil, 0, fmt.Errorf("timed out waiting for %s in tab %d", state, tabID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (runner *actionRunner) runPageInfoAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	selector := stringFlag(args, "--selector")
	maxChars, _ := intFlag(args, "--max-chars")
	if err := runner.attach(tabID); err != nil {
		return nil, 0, err
	}
	response, _, err := runner.invoke("executeCdp", map[string]any{
		"target": map[string]any{"tabId": tabID},
		"method": "Runtime.evaluate",
		"commandParams": map[string]any{
			"expression":    pageInfoJS(selector, maxChars),
			"returnByValue": true,
		},
	})
	return response, tabID, err
}

func (runner *actionRunner) runTextAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	selector := stringFlag(args, "--selector")
	maxChars, _ := intFlag(args, "--max-chars")
	if err := runner.attach(tabID); err != nil {
		return nil, 0, err
	}
	response, _, err := runner.invoke("executeCdp", map[string]any{
		"target": map[string]any{"tabId": tabID},
		"method": "Runtime.evaluate",
		"commandParams": map[string]any{
			"expression":    pageTextJS(selector, maxChars),
			"returnByValue": true,
		},
	})
	if err != nil {
		return nil, 0, err
	}
	return map[string]any{"result": map[string]any{"text": runtimeEvaluateString(response)}}, tabID, nil
}

func (runner *actionRunner) runSnapshotAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	limit := 100
	if value, ok := intFlag(args, "--limit"); ok {
		limit = value
	}
	if err := runner.attach(tabID); err != nil {
		return nil, 0, err
	}
	response, _, err := runner.invoke("executeCdp", map[string]any{
		"target":        map[string]any{"tabId": tabID},
		"method":        "Runtime.evaluate",
		"commandParams": map[string]any{"expression": snapshotJS(limit), "returnByValue": true},
	})
	if err != nil {
		return nil, 0, err
	}
	items, err := snapshotItemsFromResponse(response)
	if err != nil {
		return nil, 0, err
	}
	_ = saveSnapshot(runner.sessionID, items)
	return map[string]any{"result": map[string]any{"items": items}}, tabID, nil
}

func (runner *actionRunner) runClickAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	ref := firstStringArg(args, "--ref")
	if ref == "" {
		return nil, 0, errors.New("click requires @ref or --ref")
	}
	ref = strings.TrimPrefix(ref, "@")
	if err := runner.attach(tabID); err != nil {
		return nil, 0, err
	}
	response, _, err := runner.invoke("executeCdp", map[string]any{
		"target": map[string]any{"tabId": tabID},
		"method": "Runtime.evaluate",
		"commandParams": map[string]any{
			"expression":    clickJS(ref),
			"returnByValue": true,
		},
	})
	if err != nil {
		return nil, 0, err
	}
	result, err := interactionResultFromResponse("click", response)
	if err != nil {
		return map[string]any{"result": result}, tabID, err
	}
	return map[string]any{"result": result}, tabID, nil
}

func (runner *actionRunner) runFillAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	positionals := positionalArgs(args)
	if len(positionals) < 2 && stringFlag(args, "--text") == "" {
		return nil, 0, errors.New("fill requires @ref and text")
	}
	ref := stringFlag(args, "--ref")
	text := stringFlag(args, "--text")
	if ref == "" && len(positionals) > 0 {
		ref = positionals[0]
	}
	if text == "" && len(positionals) > 1 {
		text = strings.Join(positionals[1:], " ")
	}
	ref = strings.TrimPrefix(ref, "@")
	if err := runner.attach(tabID); err != nil {
		return nil, 0, err
	}
	response, _, err := runner.invoke("executeCdp", map[string]any{
		"target": map[string]any{"tabId": tabID},
		"method": "Runtime.evaluate",
		"commandParams": map[string]any{
			"expression":    fillJS(ref, text),
			"returnByValue": true,
		},
	})
	if err != nil {
		return nil, 0, err
	}
	result, err := interactionResultFromResponse("fill", response)
	if err != nil {
		return map[string]any{"result": result}, tabID, err
	}
	return map[string]any{"result": result}, tabID, nil
}

func (runner *actionRunner) runScreenshotAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	output := stringFlag(args, "--output")
	selector := stringFlag(args, "--selector")
	fullPage := boolFlag(args, "--full-page")
	result, err := captureScreenshot(runner.options, tabID, output, selector, fullPage)
	if err != nil {
		return nil, 0, err
	}
	return map[string]any{"result": result}, tabID, nil
}

func (runner *actionRunner) runCDPAction(args []string) (map[string]any, int, error) {
	method := stringFlagOrPositional(args, "--method", 0)
	if method == "" {
		return nil, 0, errors.New("cdp requires a method")
	}
	paramsJSON := stringFlagOrPositional(args, "--params", 1)
	if paramsJSON == "" {
		paramsJSON = "{}"
	}
	var commandParams map[string]any
	if err := json.Unmarshal([]byte(paramsJSON), &commandParams); err != nil {
		return nil, 0, fmt.Errorf("cdp params must be a JSON object: %w", err)
	}
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	if err := runner.attach(tabID); err != nil {
		return nil, 0, err
	}
	return runner.invoke("executeCdp", map[string]any{
		"target":        map[string]any{"tabId": tabID},
		"method":        method,
		"commandParams": commandParams,
	})
}

func (runner *actionRunner) runHistoryAction(args []string) (map[string]any, int, error) {
	params := map[string]any{"query": firstStringArg(args, "--query"), "limit": 100}
	if limit, ok := intFlag(args, "--limit"); ok {
		params["limit"] = limit
	}
	if from := stringFlag(args, "--from"); from != "" {
		params["from"] = from
	}
	if to := stringFlag(args, "--to"); to != "" {
		params["to"] = to
	}
	return runner.invoke("getUserHistory", params)
}

func (runner *actionRunner) runMoveMouseAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	if len(positionalArgs(args)) < 2 {
		return nil, 0, errors.New("move-mouse requires x and y")
	}
	x, err := strconv.ParseFloat(positionalArgs(args)[0], 64)
	if err != nil {
		return nil, 0, fmt.Errorf("move-mouse x must be numeric: %w", err)
	}
	y, err := strconv.ParseFloat(positionalArgs(args)[1], 64)
	if err != nil {
		return nil, 0, fmt.Errorf("move-mouse y must be numeric: %w", err)
	}
	return runner.invoke("moveMouse", map[string]any{
		"tabId":          tabID,
		"x":              x,
		"y":              y,
		"waitForArrival": true,
	})
}

func (runner *actionRunner) runWaitFileChooserAction(args []string) (map[string]any, int, error) {
	tabID, err := tabIDArgOrCurrent(args, runner.currentTabID)
	if err != nil {
		return nil, 0, err
	}
	return runner.invoke("waitForFileChooser", map[string]any{
		"tabId":     tabID,
		"timeoutMs": int(runner.options.timeout.Milliseconds()),
	})
}

func (runner *actionRunner) runSetFileChooserFilesAction(args []string) (map[string]any, int, error) {
	fileChooserID := stringFlagOrPositional(args, "--file-chooser-id", 0)
	if fileChooserID == "" {
		return nil, 0, errors.New("set-file-chooser-files requires file chooser id")
	}
	files := positionalArgs(args)
	if stringFlag(args, "--file-chooser-id") == "" && len(files) > 0 {
		files = files[1:]
	}
	if len(files) == 0 {
		return nil, 0, errors.New("set-file-chooser-files requires at least one file")
	}
	return runner.invoke("setFileChooserFiles", map[string]any{
		"fileChooserId": fileChooserID,
		"files":         files,
	})
}

func (runner *actionRunner) runFinalizeTabsAction(args []string) (map[string]any, int, error) {
	keepJSON := firstStringArg(args, "--keep")
	if keepJSON == "" {
		keepJSON = "[]"
	}
	var keep []any
	if err := json.Unmarshal([]byte(keepJSON), &keep); err != nil {
		return nil, 0, fmt.Errorf("finalize-tabs keep must be a JSON array: %w", err)
	}
	return runner.invoke("finalizeTabs", map[string]any{"keep": keep})
}

func (runner *actionRunner) runCallAction(args []string) (map[string]any, int, error) {
	method := stringFlagOrPositional(args, "--method", 0)
	if method == "" {
		return nil, 0, errors.New("call requires a method")
	}
	paramsJSON := stringFlagOrPositional(args, "--params", 1)
	if paramsJSON == "" {
		paramsJSON = "{}"
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		return nil, 0, fmt.Errorf("call params must be a JSON object: %w", err)
	}
	return runner.invoke(method, params)
}

func (runner *actionRunner) attach(tabID int) error {
	_, _, err := runner.invoke("attach", map[string]any{"tabId": tabID})
	return err
}

func (runner *actionRunner) invoke(method string, params map[string]any) (map[string]any, int, error) {
	if params == nil {
		params = map[string]any{}
	}
	params["session_id"] = runner.sessionID
	params["turn_id"] = runner.turnID
	response, err := invoke(runner.options.socketPath, runner.options.socketDir, method, params, runner.options.timeout)
	return response, runner.currentTabID, err
}

type actionLine struct {
	number int
	fields []string
}

func parseActionScript(script string) ([]actionLine, error) {
	var lines []actionLine
	for index, rawLine := range strings.Split(script, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields, err := splitActionFields(line)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", index+1, err)
		}
		if len(fields) == 0 {
			continue
		}
		lines = append(lines, actionLine{number: index + 1, fields: fields})
	}
	return lines, nil
}

func splitActionFields(line string) ([]string, error) {
	var fields []string
	var builder strings.Builder
	var quote rune
	escaped := false
	emit := func() {
		if builder.Len() > 0 {
			fields = append(fields, builder.String())
			builder.Reset()
		}
	}
	for _, current := range line {
		if escaped {
			builder.WriteRune(current)
			escaped = false
			continue
		}
		if current == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if current == quote {
				quote = 0
			} else {
				builder.WriteRune(current)
			}
			continue
		}
		if current == '"' || current == '\'' {
			quote = current
			continue
		}
		if current == '#' && builder.Len() == 0 {
			break
		}
		if current == ' ' || current == '\t' {
			emit()
			continue
		}
		builder.WriteRune(current)
	}
	if escaped {
		return nil, errors.New("unterminated escape")
	}
	if quote != 0 {
		return nil, errors.New("unterminated quote")
	}
	emit()
	return fields, nil
}

func firstStringArg(args []string, flag string) string {
	if value := stringFlag(args, flag); value != "" {
		return value
	}
	for _, arg := range positionalArgs(args) {
		return arg
	}
	return ""
}

func stringFlagOrPositional(args []string, flag string, positionalIndex int) string {
	if value := stringFlag(args, flag); value != "" {
		return value
	}
	positionals := positionalArgs(args)
	if positionalIndex >= 0 && positionalIndex < len(positionals) {
		return positionals[positionalIndex]
	}
	return ""
}

func firstIntArg(args []string, flag string) (int, error) {
	if value, ok := intFlag(args, flag); ok {
		return value, nil
	}
	for _, arg := range positionalArgs(args) {
		value, err := strconv.Atoi(arg)
		if err != nil {
			return 0, err
		}
		return value, nil
	}
	return 0, errors.New("missing integer argument")
}

func tabIDArgOrCurrent(args []string, currentTabID int) (int, error) {
	if value, ok := intFlag(args, "--tab-id"); ok {
		return value, nil
	}
	if currentTabID > 0 {
		return currentTabID, nil
	}
	return 0, errors.New("action requires --tab-id or a previous open-tab/claim-tab action")
}

func stringFlag(args []string, name string) string {
	for index, arg := range args {
		if arg == name && index+1 < len(args) {
			return args[index+1]
		}
		prefix := name + "="
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}

func intFlag(args []string, name string) (int, bool) {
	value := stringFlag(args, name)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func boolFlag(args []string, name string) bool {
	for index, arg := range args {
		if arg == name {
			return true
		}
		prefix := name + "="
		if strings.HasPrefix(arg, prefix) {
			value := strings.TrimPrefix(arg, prefix)
			parsed, err := strconv.ParseBool(value)
			return err == nil && parsed
		}
		if strings.HasPrefix(arg, "--") && !strings.Contains(arg, "=") {
			index++
		}
	}
	return false
}

func positionalArgs(args []string) []string {
	var out []string
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if strings.HasPrefix(arg, "--") {
			if !strings.Contains(arg, "=") {
				index++
			}
			continue
		}
		out = append(out, arg)
	}
	return out
}

func runtimeEvaluateString(response map[string]any) string {
	result, _ := response["result"].(map[string]any)
	cdpResult, _ := result["result"].(map[string]any)
	value, _ := cdpResult["value"].(string)
	return value
}

func runtimeEvaluateValue(response map[string]any) any {
	result, _ := response["result"].(map[string]any)
	cdpResult, _ := result["result"].(map[string]any)
	return cdpResult["value"]
}

func runtimeEvaluateMap(response map[string]any) (map[string]any, error) {
	value, ok := runtimeEvaluateValue(response).(map[string]any)
	if !ok {
		return nil, errors.New("runtime evaluation returned no object value")
	}
	return value, nil
}

func invokeAndWrite(options socketOptions, method string, params map[string]any) error {
	return invokeAndWriteTo(os.Stdout, options, method, params)
}

func invokeAndWriteTo(writer io.Writer, options socketOptions, method string, params map[string]any) error {
	response, err := invokeWithOptions(options, method, params)
	if err != nil {
		return err
	}
	if options.jsonOutput {
		result, _ := response["result"]
		return writeJSONTo(writer, normalizeJSONResult(method, result))
	}
	return writeJSONTo(writer, response)
}

func normalizeJSONResult(method string, result any) any {
	switch method {
	case "ping":
		return map[string]any{"status": result}
	case "getTabs", "getUserTabs", "getUserHistory":
		if result == nil {
			return map[string]any{"items": []any{}}
		}
		return map[string]any{"items": result}
	case "claimUserTab":
		return map[string]any{"tab": result}
	case "nameSession", "finalizeTabs", "turnEnded":
		return map[string]any{"ok": true}
	default:
		return result
	}
}

func invokeWithOptions(options socketOptions, method string, params map[string]any) (map[string]any, error) {
	applySessionDefaults(params, options.sessionID)
	return invoke(options.socketPath, options.socketDir, method, params, options.timeout)
}

func invoke(socketPath string, socketDir string, method string, params map[string]any, timeout time.Duration) (map[string]any, error) {
	conn, err := dialBrowserSocket(socketPath, socketDir, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	applySessionDefaults(params, defaultCLISessionID)
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      "cli-1",
		"method":  method,
		"params":  params,
	}
	if err := wire.WriteJSON(conn, request); err != nil {
		return nil, err
	}
	var response map[string]any
	if err := wire.ReadJSON(conn, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func applySessionDefaults(params map[string]any, sessionID string) {
	if sessionID == "" {
		if env := os.Getenv("OBU_SESSION_ID"); env != "" {
			sessionID = env
		} else {
			sessionID = defaultCLISessionID
		}
	}
	if _, ok := params["session_id"]; !ok {
		params["session_id"] = sessionID
	}
	if _, ok := params["turn_id"]; !ok {
		params["turn_id"] = fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano())
	}
	if _, ok := params["request_id"]; !ok {
		params["request_id"] = fmt.Sprintf("%s-%d", sessionID, time.Now().UnixNano())
	}
}

func envTabID(flagTabID int) int {
	if flagTabID > 0 {
		return flagTabID
	}
	if env := os.Getenv("OBU_TAB_ID"); env != "" {
		if id, err := strconv.Atoi(env); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

func dialBrowserSocket(socketPath string, socketDir string, timeout time.Duration) (net.Conn, error) {
	if socketPath != "" {
		return host.DialSocket(socketPath, timeout)
	}

	// Windows uses fixed TCP port - no socket file discovery needed
	if runtime.GOOS == "windows" {
		conn, err := host.DialSocket("", timeout)
		if err != nil {
			return nil, fmt.Errorf("TCP relay on 127.0.0.1:%d is not available; ensure the browser extension is running: %w", host.TCPPort, err)
		}
		return conn, nil
	}

	record, err := host.ReadActiveSocketRecord(socketDir)
	if err == nil {
		conn, dialErr := host.DialSocket(record.SocketPath, timeout)
		if dialErr == nil {
			return conn, nil
		}
		_ = host.RemoveActiveSocketRecord(socketDir, record.SocketPath)
		removeSocketPathIfInDir(socketDir, record.SocketPath)
		conn, scanErr := scanSocketDir(socketDir, record.SocketPath, timeout)
		if scanErr == nil {
			return conn, nil
		}
		return nil, fmt.Errorf("active socket registry points to unavailable socket %q; removed stale registry entry; no connectable socket found by scanning: %w", record.SocketPath, dialErr)
	}

	conn, scanErr := scanSocketDir(socketDir, "", timeout)
	if scanErr == nil {
		return conn, nil
	}
	return nil, fmt.Errorf("socket not provided and active socket registry is unavailable; no connectable socket found by scanning: %w", err)
}

type socketCandidate struct {
	path    string
	modTime time.Time
}

func scanSocketDir(socketDir string, skipPath string, timeout time.Duration) (net.Conn, error) {
	dir := socketDir
	if dir == "" {
		dir = host.DefaultSocketDir()
	}

	// On Windows, connect to the TCP relay directly
	if runtime.GOOS == "windows" {
		conn, err := host.DialSocket("", timeout)
		if err == nil {
			return conn, nil
		}
		return nil, fmt.Errorf("TCP relay not available on 127.0.0.1:%d: %w", host.TCPPort, err)
	}

	// Unix: scan for .sock files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var candidates []socketCandidate
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sock") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		candidates = append(candidates, socketCandidate{
			path:    filepath.Join(dir, entry.Name()),
			modTime: info.ModTime(),
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].modTime.Equal(candidates[j].modTime) {
			return candidates[i].path > candidates[j].path
		}
		return candidates[i].modTime.After(candidates[j].modTime)
	})
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no socket files in %q", dir)
	}

	probeTimeout := timeout
	if probeTimeout > 500*time.Millisecond {
		probeTimeout = 500 * time.Millisecond
	}
	var lastErr error
	for _, candidate := range candidates {
		if candidate.path == skipPath {
			continue
		}
		conn, err := host.DialSocket(candidate.path, probeTimeout)
		if err == nil {
			repairActiveSocketRecord(dir, candidate)
			cleanupStaleSocketCandidates(dir, candidates, candidate.path, probeTimeout)
			return conn, nil
		}
		removeSocketPathIfInDir(dir, candidate.path)
		lastErr = err
	}
	if lastErr == nil {
		return nil, fmt.Errorf("no socket files in %q after filtering", dir)
	}
	return nil, lastErr
}

func cleanupStaleSocketCandidates(socketDir string, candidates []socketCandidate, keepPath string, timeout time.Duration) {
	cleanupTimeout := timeout
	if cleanupTimeout > 100*time.Millisecond {
		cleanupTimeout = 100 * time.Millisecond
	}
	for _, candidate := range candidates {
		if candidate.path == keepPath {
			continue
		}
		conn, err := host.DialSocket(candidate.path, cleanupTimeout)
		if err == nil {
			_ = conn.Close()
			continue
		}
		removeSocketPathIfInDir(socketDir, candidate.path)
	}
}

func repairActiveSocketRecord(socketDir string, candidate socketCandidate) {
	record := host.ActiveSocketRecord{
		SocketPath: candidate.path,
		PID:        0,
		StartedAt:  candidate.modTime.UTC(),
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return
	}
	path := host.ActiveSocketRecordPath(socketDir)
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		return
	}
	_ = os.Chmod(path, 0o600)
}

func removeSocketPathIfInDir(socketDir string, socketPath string) {
	dir := socketDir
	if dir == "" {
		dir = host.DefaultSocketDir()
	}
	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		return
	}
	pathAbs, err := filepath.Abs(socketPath)
	if err != nil {
		return
	}
	rel, err := filepath.Rel(dirAbs, pathAbs)
	if err != nil || rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." {
		return
	}
	if filepath.Dir(rel) != "." || !strings.HasSuffix(rel, ".sock") {
		return
	}
	_ = os.Remove(pathAbs)
}

func writeJSON(value any) error {
	return writeJSONTo(os.Stdout, value)
}

func writeJSONTo(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func nativeManifest(extensionID string, hostPath string) (map[string]any, error) {
	allowedExtensionID := strings.TrimSpace(extensionID)
	if allowedExtensionID == "" {
		allowedExtensionID = defaultChromeExtensionID
	}
	path := hostPath
	if path == "" {
		var err error
		path, err = stableNativeHostPath()
		if err != nil {
			return nil, err
		}
	} else {
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		path = absolutePath
	}
	return map[string]any{
		"name":        host.NativeHostName,
		"description": "Open Browser Use for Windows native messaging host",
		"type":        "stdio",
		"path":        path,
		"allowed_origins": []string{
			fmt.Sprintf("chrome-extension://%s/", allowedExtensionID),
		},
	}, nil
}

func resolveNativeHostTarget(binaryPath string) (string, error) {
	path := binaryPath
	if path == "" {
		executable, err := os.Executable()
		if err != nil {
			return "", err
		}
		path = executable
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("native host binary target is a directory: %s", absolutePath)
	}
	// On Windows, check for .exe extension instead of Unix permission bits
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(strings.ToLower(absolutePath), ".exe") {
			return "", fmt.Errorf("native host binary target is not a Windows executable: %s", absolutePath)
		}
	} else {
		if info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("native host binary target is not executable: %s", absolutePath)
		}
	}
	return absolutePath, nil
}

func stableNativeHostPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library/Application Support/OpenBrowserUse/native-host/open-browser-use"), nil
	case "linux":
		return filepath.Join(home, ".local/share/open-browser-use/native-host/open-browser-use"), nil
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, "OpenBrowserUse", "native-host", "open-browser-use.exe"), nil
	default:
		return "", fmt.Errorf("stable native host link path is not implemented for %s", runtime.GOOS)
	}
}

func defaultChromeExternalExtensionPath(browser string, extensionID string) (string, error) {
	filename := strings.TrimSpace(extensionID) + ".json"
	if filename == ".json" {
		return "", errors.New("Chrome extension id is empty")
	}
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library/Application Support/Google/Chrome/External Extensions", filename), nil
	case "linux":
		return filepath.Join("/opt/google/chrome/extensions", filename), nil
	case "windows":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		if browser == browserEdge {
			return filepath.Join(localAppData, "Microsoft", "Edge", "User Data", "External Extensions", filename), nil
		}
		return filepath.Join(localAppData, "Google", "Chrome", "User Data", "External Extensions", filename), nil
	default:
		return "", fmt.Errorf("Chrome external extension setup is not implemented for %s", runtime.GOOS)
	}
}

func installStableNativeHostLink(targetPath string, linkPath string) error {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o700); err != nil {
		return err
	}
	if err := os.Remove(linkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// On Windows, copy the file instead of creating a symlink
	// (symlinks require admin privileges or developer mode)
	if runtime.GOOS == "windows" {
		return copyFile(targetPath, linkPath)
	}
	return os.Symlink(targetPath, linkPath)
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

func defaultNativeHostManifestPath(browser string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	filename := host.NativeHostName + ".json"
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts", filename), nil
	case "linux":
		return filepath.Join(home, ".config/google-chrome/NativeMessagingHosts", filename), nil
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		if browser == browserEdge {
			return filepath.Join(localAppData, "Microsoft", "Edge", "User Data", "NativeMessagingHosts", filename), nil
		}
		return filepath.Join(localAppData, "Google", "Chrome", "User Data", "NativeMessagingHosts", filename), nil
	default:
		return "", fmt.Errorf("default manifest install path is not implemented for %s; pass --output", runtime.GOOS)
	}
}

func downloadLatestReleaseZIP() (string, error) {
	assetName := fmt.Sprintf("open-browser-use-chrome-extension-%s.zip", version)
	url := fmt.Sprintf(
		"https://github.com/Kiana-ZY/open-browser-use-windows/releases/download/v%s/%s",
		version,
		assetName,
	)
	return downloadReleaseAsset(assetName, url)
}

func downloadReleaseAsset(name string, url string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	targetDir := filepath.Join(cacheDir, "open-browser-use", "extensions")
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, filepath.Base(name))
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("User-Agent", "open-browser-use/"+version)
	client := http.Client{Timeout: 2 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("extension ZIP download failed: %s", response.Status)
	}
	tempPath := targetPath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	_, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tempPath)
		return "", copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tempPath)
		return "", closeErr
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return targetPath, nil
}

func resolveExistingExtensionZIP(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolutePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("extension ZIP path is a directory: %s", absolutePath)
	}
	return absolutePath, nil
}

func installUnpackedExtension(zipPath string) (string, string, error) {
	targetDir, err := defaultUnpackedExtensionDir()
	if err != nil {
		return "", "", err
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return "", "", err
	}
	if err := unzipExtension(zipPath, targetDir); err != nil {
		return "", "", err
	}
	extensionID, err := pinUnpackedExtensionKey(filepath.Join(targetDir, "manifest.json"))
	if err != nil {
		return "", "", err
	}
	return targetDir, extensionID, nil
}

func writeBetaInstallZIP(unpackedPath string, sourceZIPPath string) (string, error) {
	targetPath := sourceZIPPath
	tempPath := targetPath + ".tmp"
	if err := os.Remove(tempPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := packageExtensionDirectory(unpackedPath, tempPath); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return targetPath, nil
}

func packageExtensionDirectory(sourceDir string, targetZIPPath string) error {
	target, err := os.Create(targetZIPPath)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(target)
	walkErr := filepath.WalkDir(sourceDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relativePath)
		header.Method = zip.Deflate
		destination, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		source, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(destination, source)
		closeErr := source.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	closeWriterErr := writer.Close()
	closeTargetErr := target.Close()
	if walkErr != nil {
		return walkErr
	}
	if closeWriterErr != nil {
		return closeWriterErr
	}
	return closeTargetErr
}

func defaultUnpackedExtensionDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library/Application Support/OpenBrowserUse/chrome-extension/release"), nil
	case "linux":
		return filepath.Join(home, ".local/share/open-browser-use/chrome-extension/release"), nil
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, "OpenBrowserUse", "chrome-extension", "release"), nil
	default:
		return "", fmt.Errorf("release extension setup is not implemented for %s", runtime.GOOS)
	}
}

func unzipExtension(zipPath string, targetDir string) error {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	cleanTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return err
	}
	for _, file := range reader.File {
		targetPath := filepath.Join(cleanTargetDir, file.Name)
		if targetPath != cleanTargetDir && !strings.HasPrefix(targetPath, cleanTargetDir+string(os.PathSeparator)) {
			return fmt.Errorf("extension ZIP contains unsafe path: %s", file.Name)
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o700); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
			return err
		}
		source, err := file.Open()
		if err != nil {
			return err
		}
		destination, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			_ = source.Close()
			return err
		}
		_, copyErr := io.Copy(destination, source)
		closeDestinationErr := destination.Close()
		closeSourceErr := source.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeDestinationErr != nil {
			return closeDestinationErr
		}
		if closeSourceErr != nil {
			return closeSourceErr
		}
	}
	return nil
}

func pinUnpackedExtensionKey(manifestPath string) (string, error) {
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		return "", err
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return "", err
	}
	key, _ := manifest["key"].(string)
	if strings.TrimSpace(key) == "" {
		key = betaExtensionPublicKey
		manifest["key"] = key
		updated, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(manifestPath, append(updated, '\n'), 0o600); err != nil {
			return "", err
		}
	}
	return extensionIDFromPublicKey(key)
}

func extensionIDFromPublicKey(key string) (string, error) {
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(key))
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(der)
	return extensionIDFromCRXID(sum[:16]), nil
}

func extensionIDFromCRX(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	prefix := make([]byte, 12)
	if _, err := io.ReadFull(file, prefix); err != nil {
		return "", err
	}
	if string(prefix[0:4]) != "Cr24" {
		return "", fmt.Errorf("CRX %s does not start with Cr24", path)
	}
	version := binary.LittleEndian.Uint32(prefix[4:8])
	if version != 3 {
		return "", fmt.Errorf("CRX %s uses unsupported version %d", path, version)
	}
	headerLength := binary.LittleEndian.Uint32(prefix[8:12])
	header := make([]byte, headerLength)
	if _, err := io.ReadFull(file, header); err != nil {
		return "", err
	}
	signedHeaderData, err := protobufBytesField(header, 10000)
	if err != nil {
		return "", err
	}
	crxID, err := protobufBytesField(signedHeaderData, 1)
	if err != nil {
		return "", err
	}
	if len(crxID) != 16 {
		return "", fmt.Errorf("CRX %s has invalid id length %d", path, len(crxID))
	}
	return extensionIDFromCRXID(crxID), nil
}

func protobufBytesField(payload []byte, wantedField uint64) ([]byte, error) {
	for offset := 0; offset < len(payload); {
		key, n, err := readVarint(payload[offset:])
		if err != nil {
			return nil, err
		}
		offset += n
		fieldNumber := key >> 3
		wireType := key & 0x7
		switch wireType {
		case 0:
			_, n, err := readVarint(payload[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
		case 1:
			offset += 8
		case 2:
			length, n, err := readVarint(payload[offset:])
			if err != nil {
				return nil, err
			}
			offset += n
			if length > uint64(len(payload)-offset) {
				return nil, errors.New("protobuf bytes field exceeds payload")
			}
			end := offset + int(length)
			if fieldNumber == wantedField {
				return payload[offset:end], nil
			}
			offset = end
		case 5:
			offset += 4
		default:
			return nil, fmt.Errorf("unsupported protobuf wire type %d", wireType)
		}
		if offset > len(payload) {
			return nil, errors.New("protobuf field exceeds payload")
		}
	}
	return nil, fmt.Errorf("protobuf bytes field %d not found", wantedField)
}

func readVarint(payload []byte) (uint64, int, error) {
	var value uint64
	for i, b := range payload {
		if i == 10 {
			return 0, 0, errors.New("protobuf varint is too long")
		}
		value |= uint64(b&0x7f) << (7 * i)
		if b < 0x80 {
			return value, i + 1, nil
		}
	}
	return 0, 0, io.ErrUnexpectedEOF
}

func extensionIDFromCRXID(crxID []byte) string {
	const alphabet = "abcdefghijklmnop"
	var builder strings.Builder
	builder.Grow(len(crxID) * 2)
	for _, b := range crxID {
		builder.WriteByte(alphabet[(b>>4)&0x0f])
		builder.WriteByte(alphabet[b&0x0f])
	}
	return builder.String()
}

func openFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-a", "Google Chrome", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		return fmt.Errorf("opening files is not implemented for %s", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func revealFile(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "linux":
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	case "windows":
		cmd = exec.Command("explorer", "/select,", path)
	default:
		return fmt.Errorf("revealing files is not implemented for %s", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func openBrowserExtensionsPage(browser string) error {
	var cmd *exec.Cmd
	extURL := "chrome://extensions/"
	if browser == browserEdge {
		extURL = "edge://extensions/"
	}
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-a", "Google Chrome", extURL)
	case "linux":
		cmd = exec.Command("xdg-open", extURL)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", extURL)
	default:
		return fmt.Errorf("opening extensions page is not implemented for %s", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func numberAsInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case float64:
		if typed == float64(int(typed)) {
			return int(typed), true
		}
	}
	return 0, false
}
