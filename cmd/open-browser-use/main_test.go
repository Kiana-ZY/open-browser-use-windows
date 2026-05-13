package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Kiana-ZY/open-browser-use-windows/internal/host"
	"github.com/Kiana-ZY/open-browser-use-windows/internal/wire"
)

func TestNativeHostNameIsChromeCompatible(t *testing.T) {
	valid := regexp.MustCompile(`^[a-z0-9_]+(\.[a-z0-9_]+)*$`)
	if !valid.MatchString(host.NativeHostName) {
		t.Fatalf("native host name is not Chrome-compatible: %q", host.NativeHostName)
	}
}

func TestNativeMessagingLaunchArg(t *testing.T) {
	if !isNativeMessagingLaunch("chrome-extension://nfjjgckfgejeofdcmaepbapclmldcflf/") {
		t.Fatal("expected Chrome extension origin to launch host mode")
	}
	if isNativeMessagingLaunch("host") {
		t.Fatal("expected CLI subcommand not to be treated as native messaging launch")
	}
}

func TestCobraVersionCommand(t *testing.T) {
	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != version {
		t.Fatalf("expected version %q, got %q", version, got)
	}
}

func TestCobraNoArgsPrintsVersionAndExtensionStatus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "📦 CLI version: "+version) {
		t.Fatalf("expected no-arg output to include CLI version, got %q", got)
	}
	if !strings.Contains(got, "🧩 Browser extension:") {
		t.Fatalf("expected no-arg output to include browser extension status, got %q", got)
	}
	if !strings.Contains(got, "open-browser-use setup") && !strings.Contains(got, "open-browser-use user-tabs") {
		t.Fatalf("expected no-arg output to include a setup or ready next step, got %q", got)
	}
}

func TestCobraVersionFlag(t *testing.T) {
	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"-v"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(output.String()); got != version {
		t.Fatalf("expected version %q, got %q", version, got)
	}
}

func TestNativeManifestDefaultsToStoreExtensionAndStableHostPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	manifest, err := nativeManifest("", "")
	if err != nil {
		t.Fatal(err)
	}
	origins, ok := manifest["allowed_origins"].([]string)
	if !ok || len(origins) != 1 {
		t.Fatalf("expected one allowed origin, got %#v", manifest["allowed_origins"])
	}
	if origins[0] != "chrome-extension://"+defaultChromeExtensionID+"/" {
		t.Fatalf("expected default extension origin, got %q", origins[0])
	}
	hostPath, err := stableNativeHostPath()
	if err != nil {
		t.Fatal(err)
	}
	if manifest["path"] != hostPath {
		t.Fatalf("expected manifest path %q, got %#v", hostPath, manifest["path"])
	}
}

func TestInstallNativeManifestCreatesStableLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stable native host link is not implemented on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	targetPath := filepath.Join(t.TempDir(), "open-browser-use")
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath, err := installNativeManifest("chrome", "", targetPath, "")
	if err != nil {
		t.Fatal(err)
	}
	hostPath, err := stableNativeHostPath()
	if err != nil {
		t.Fatal(err)
	}
	linkTarget, err := os.Readlink(hostPath)
	if err != nil {
		t.Fatal(err)
	}
	if linkTarget != targetPath {
		t.Fatalf("expected stable link to point at %q, got %q", targetPath, linkTarget)
	}
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["path"] != hostPath {
		t.Fatalf("expected manifest path %q, got %#v", hostPath, manifest["path"])
	}
}

func TestCobraInstallManifestDefaultsToStoreExtension(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stable native host link is not implemented on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	targetPath := filepath.Join(t.TempDir(), "open-browser-use")
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"install-manifest", "--path", targetPath})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	manifestPath := strings.TrimSpace(output.String())
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	origins, ok := manifest["allowed_origins"].([]any)
	if !ok || len(origins) != 1 {
		t.Fatalf("expected one allowed origin, got %#v", manifest["allowed_origins"])
	}
	if origins[0] != "chrome-extension://"+defaultChromeExtensionID+"/" {
		t.Fatalf("expected default extension origin, got %#v", origins[0])
	}
}

func TestInstallChromeExternalExtensionWritesWebStoreHint(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), defaultChromeExtensionID+".json")
	path, err := installChromeExternalExtension("chrome", "", outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if path != outputPath {
		t.Fatalf("expected output path %q, got %q", outputPath, path)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["external_update_url"] != chromeWebStoreUpdateURL {
		t.Fatalf("expected Chrome Web Store update URL, got %#v", manifest["external_update_url"])
	}
}

func TestCobraSetupWritesNativeAndExternalManifests(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stable native host link is not implemented on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	targetPath := filepath.Join(t.TempDir(), "open-browser-use")
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	externalPath := filepath.Join(t.TempDir(), defaultChromeExtensionID+".json")

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"setup", "--path", targetPath, "--external-extension-output", externalPath})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "✅ Open Browser Use setup") {
		t.Fatalf("expected setup output to mention native manifest, got %q", output.String())
	}
	if !strings.Contains(output.String(), "Registered native host") {
		t.Fatalf("expected setup output to mention native host registration, got %q", output.String())
	}
	if !strings.Contains(output.String(), "Browser extension") {
		t.Fatalf("expected setup output to mention browser extension status, got %q", output.String())
	}
	if _, err := os.Stat(filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts", host.NativeHostName+".json")); runtime.GOOS == "darwin" && err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(externalPath); err != nil {
		t.Fatal(err)
	}
}

func TestCobraSetupBetaUsesProvidedZIP(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stable native host link is not implemented on windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	targetPath := filepath.Join(t.TempDir(), "open-browser-use")
	if err := os.WriteFile(targetPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	zipPath := filepath.Join(t.TempDir(), "open-browser-use-chrome-extension.zip")
	expectedExtensionID, err := extensionIDFromPublicKey(betaExtensionPublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTestExtensionZIP(zipPath); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"setup", "beta", "--path", targetPath, "--zip", zipPath, "--no-open"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "ZIP:") || !strings.Contains(got, zipPath) {
		t.Fatalf("expected setup beta output to mention install ZIP path, got %q", got)
	}
	if strings.Contains(got, "-manual.zip") {
		t.Fatalf("expected setup beta output to avoid a separate manual ZIP path, got %q", got)
	}
	if !strings.Contains(got, "Extension id: "+expectedExtensionID) {
		t.Fatalf("expected setup beta output to mention unpacked extension id, got %q", got)
	}
	if !strings.Contains(got, "Drag the ZIP file into the Chrome extensions page") && !strings.Contains(got, "Drag the ZIP file into the Edge extensions page") && !strings.Contains(got, "All set.") {
		t.Fatalf("expected setup beta output to mention manual install or connected status, got %q", got)
	}
	manifestPath := filepath.Join(home, "Library/Application Support/Google/Chrome/NativeMessagingHosts", host.NativeHostName+".json")
	if runtime.GOOS == "linux" {
		manifestPath = filepath.Join(home, ".config/google-chrome/NativeMessagingHosts", host.NativeHostName+".json")
	}
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), "chrome-extension://"+expectedExtensionID+"/") {
		t.Fatalf("expected native manifest to allow unpacked extension id, got %s", payload)
	}
	unpackedPath, err := defaultUnpackedExtensionDir()
	if err != nil {
		t.Fatal(err)
	}
	unpackedManifest, err := os.ReadFile(filepath.Join(unpackedPath, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(unpackedManifest), betaExtensionPublicKey) {
		t.Fatalf("expected unpacked manifest to include stable key, got %s", unpackedManifest)
	}
	installManifest, err := readManifestFromZIP(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installManifest), betaExtensionPublicKey) {
		t.Fatalf("expected install ZIP manifest to include stable key, got %s", installManifest)
	}
}

func TestDetectInstalledChromeExtensionFromProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	profileRoot := filepath.Join(home, "Library/Application Support/Google/Chrome/Default/Extensions", defaultChromeExtensionID, "0.1.10")
	if runtime.GOOS == "linux" {
		profileRoot = filepath.Join(home, ".config/google-chrome/Default/Extensions", defaultChromeExtensionID, "0.1.10")
	}
	if runtime.GOOS == "windows" {
		t.Skip("Chrome profile detection is not implemented on windows")
	}
	if err := os.MkdirAll(profileRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileRoot, "manifest.json"), []byte(`{"version":"0.1.10"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	detected, ok := detectInstalledChromeExtension("chrome")
	if !ok {
		t.Fatal("expected installed extension to be detected")
	}
	if detected.ExtensionID != defaultChromeExtensionID {
		t.Fatalf("expected extension id %q, got %q", defaultChromeExtensionID, detected.ExtensionID)
	}
	if detected.Version != "0.1.10" {
		t.Fatalf("expected version 0.1.10, got %q", detected.Version)
	}
}

func TestCompareChromeVersions(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{left: "0.1.15", right: "0.1.11", want: 1},
		{left: "0.1.15", right: "0.1.15", want: 0},
		{left: "0.1.11", right: "0.1.15", want: -1},
		{left: "0.1", right: "0.1.0", want: 0},
	}
	for _, test := range tests {
		got := compareChromeVersions(test.left, test.right)
		if got != test.want {
			t.Fatalf("compareChromeVersions(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

func TestBrowserExtensionStatusSummaries(t *testing.T) {
	ready := browserExtensionStatus{
		Installed:       true,
		Reachable:       true,
		Version:         version,
		ExpectedVersion: version,
	}
	if got := ready.summary(); !strings.Contains(got, "Ready") || !strings.Contains(got, version) {
		t.Fatalf("expected ready summary with version, got %q", got)
	}

	outdated := browserExtensionStatus{
		Installed:       true,
		Version:         "0.1.0",
		ExpectedVersion: version,
		UpgradeCommand:  "open-browser-use setup",
	}
	if got := outdated.summary(); !strings.Contains(got, "CLI expects v"+version) || !strings.Contains(got, "open-browser-use setup") {
		t.Fatalf("expected upgrade summary with command, got %q", got)
	}
}

func TestInspectNativeManifestPayloadAcceptsExpectedManifest(t *testing.T) {
	stablePath := filepath.Join(t.TempDir(), "open-browser-use")
	payload, err := json.Marshal(map[string]any{
		"name": host.NativeHostName,
		"path": stablePath,
		"allowed_origins": []string{
			"chrome-extension://" + defaultChromeExtensionID + "/",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := doctorReport{
		NativeHost: doctorNativeHostReport{Name: host.NativeHostName},
	}

	ok, detail := inspectNativeManifestPayload(payload, stablePath, &report)
	if !ok {
		t.Fatalf("expected manifest to pass, detail=%q", detail)
	}
	if report.NativeHost.ManifestPathInFile != stablePath {
		t.Fatalf("expected manifest path %q, got %q", stablePath, report.NativeHost.ManifestPathInFile)
	}
	if len(report.NativeHost.ManifestAllowedOrigins) != 1 {
		t.Fatalf("expected one allowed origin, got %#v", report.NativeHost.ManifestAllowedOrigins)
	}
}

func TestInspectNativeManifestPayloadRejectsWrongHostPath(t *testing.T) {
	stablePath := filepath.Join(t.TempDir(), "open-browser-use")
	payload, err := json.Marshal(map[string]any{
		"name": host.NativeHostName,
		"path": filepath.Join(t.TempDir(), "other-open-browser-use"),
		"allowed_origins": []string{
			"chrome-extension://" + defaultChromeExtensionID + "/",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := doctorReport{
		NativeHost: doctorNativeHostReport{Name: host.NativeHostName},
	}

	ok, detail := inspectNativeManifestPayload(payload, stablePath, &report)
	if ok {
		t.Fatal("expected manifest to fail")
	}
	if !strings.Contains(detail, "expected") {
		t.Fatalf("expected detail to explain expected path, got %q", detail)
	}
}

func TestCobraDoctorJSONReportsRelayAndExtension(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	stablePath, err := stableNativeHostPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(stablePath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stablePath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath, err := defaultNativeHostManifestPath(browserChrome)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := nativeManifest(defaultChromeExtensionID, stablePath)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, append(payload, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			result := any(map[string]any{})
			switch method {
			case "ping":
				result = "pong"
			case "getInfo":
				result = map[string]any{
					"version": version,
					"metadata": map[string]any{
						"extensionId": defaultChromeExtensionID,
					},
				}
			default:
				_ = conn.Close()
				serverDone <- fmt.Errorf("unexpected method %q", method)
				return
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"doctor", "--socket", socketPath, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got doctorReport
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.OK {
		t.Fatalf("expected doctor report to be ok, got %#v", got.Checks)
	}
	if !got.Socket.Reachable || got.Socket.PingStatus != "pong" {
		t.Fatalf("expected reachable socket with pong, got %#v", got.Socket)
	}
	if !got.NativeHost.ManifestOK {
		t.Fatalf("expected manifest ok, got %#v", got.NativeHost)
	}
	if !got.BrowserExtension.Reachable || got.BrowserExtension.Version != version {
		t.Fatalf("expected connected extension version, got %#v", got.BrowserExtension)
	}
}

func TestCobraDoctorTextShowsNextStepsWhenMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"doctor", "--timeout", "1ms"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "Open Browser Use doctor") {
		t.Fatalf("expected doctor heading, got %q", got)
	}
	if !strings.Contains(got, "Status: needs attention") {
		t.Fatalf("expected needs attention status, got %q", got)
	}
	if !strings.Contains(got, "Next steps:") {
		t.Fatalf("expected next steps, got %q", got)
	}
}

func TestCobraDoctorAllJSONReportsBothBrowsers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"doctor", "--browser", "all", "--socket", "127.0.0.1:1", "--timeout", "1ms", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var got doctorSuiteReport
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.OK {
		t.Fatalf("expected not-ready suite report, got %#v", got)
	}
	if len(got.Browsers) != 2 {
		t.Fatalf("expected chrome and edge reports, got %#v", got.Browsers)
	}
	if got.Browsers[0].Browser != browserChrome || got.Browsers[1].Browser != browserEdge {
		t.Fatalf("expected chrome then edge reports, got %#v", got.Browsers)
	}
	if len(got.NextSteps) == 0 {
		t.Fatalf("expected suite next steps, got %#v", got)
	}
}

func TestCobraDoctorAllTextShowsOverallStatus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", filepath.Join(home, "AppData", "Local"))

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"doctor", "--browser", "all", "--socket", "127.0.0.1:1", "--timeout", "1ms"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "Browser: chrome") || !strings.Contains(got, "Browser: edge") {
		t.Fatalf("expected both browser sections, got %q", got)
	}
	if !strings.Contains(got, "Overall status: needs attention") {
		t.Fatalf("expected overall status, got %q", got)
	}
}

func TestCobraUnknownCommand(t *testing.T) {
	cmd := newRootCommand()
	cmd.SetArgs([]string{"does-not-exist"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected unknown command to fail")
	}
}

func TestInvokeRemovesStaleActiveSocketRecord(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket cleanup logic not used on Windows")
	}
	socketDir := t.TempDir()
	socketPath := filepath.Join(socketDir, "missing.sock")
	if err := host.WriteActiveSocketRecord(socketDir, socketPath); err != nil {
		t.Fatal(err)
	}

	_, err := invoke("", socketDir, "getInfo", map[string]any{}, 10*time.Millisecond)
	if err == nil {
		t.Fatal("expected stale active socket to fail")
	}
	if !strings.Contains(err.Error(), "removed stale registry entry") {
		t.Fatalf("expected stale registry cleanup error, got %v", err)
	}
	if _, err := host.ReadActiveSocketRecord(socketDir); err == nil {
		t.Fatal("expected stale active socket record to be removed")
	}
}

func TestInvokeScansSocketDirWhenActiveRecordMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket scan is not used on Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-socket-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	stalePath := filepath.Join(socketDir, "stale.sock")
	staleListener, err := net.Listen("unix", stalePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := staleListener.Close(); err != nil {
		t.Fatal(err)
	}
	socketPath := filepath.Join(socketDir, "live.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			serverDone <- err
			return
		}
		if request["method"] != "getInfo" {
			serverDone <- fmt.Errorf("expected getInfo request, got %#v", request["method"])
			return
		}
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result":  map[string]any{"version": version},
		}); err != nil {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	response, err := invoke("", socketDir, "getInfo", map[string]any{}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	result, _ := response["result"].(map[string]any)
	if result["version"] != version {
		t.Fatalf("expected scanned socket response, got %#v", response)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	record, err := host.ReadActiveSocketRecord(socketDir)
	if err != nil {
		t.Fatal(err)
	}
	if record.SocketPath != socketPath {
		t.Fatalf("expected active socket record to be repaired to %q, got %#v", socketPath, record)
	}
	if record.PID != 0 {
		t.Fatalf("expected repaired active socket pid to be unknown, got %d", record.PID)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale socket file to be removed after fallback success, got %v", err)
	}
}

func TestInvokeCleansStaleSocketFilesDuringScan(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket scan is not used on Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-socket-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)

	stalePath := filepath.Join(socketDir, "stale.sock")
	staleListener, err := net.Listen("unix", stalePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := staleListener.Close(); err != nil {
		t.Fatal(err)
	}
	if err := host.WriteActiveSocketRecord(socketDir, stalePath); err != nil {
		t.Fatal(err)
	}

	livePath := filepath.Join(socketDir, "live.sock")
	listener, err := net.Listen("unix", livePath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			serverDone <- err
			return
		}
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result":  map[string]any{"version": version},
		}); err != nil {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	if _, err := invoke("", socketDir, "getInfo", map[string]any{}, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale socket file to be removed, got %v", err)
	}
	record, err := host.ReadActiveSocketRecord(socketDir)
	if err != nil {
		t.Fatal(err)
	}
	if record.SocketPath != livePath {
		t.Fatalf("expected active socket record to point to live socket %q, got %#v", livePath, record)
	}
}

func TestCobraRunActionPlan(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket action plan test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(t.TempDir(), "obu.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	requests := make(chan map[string]any, 16)
	serverDone := make(chan error, 1)
	go func() {
		defer close(requests)
		var sessionID string
		var turnID string
		for count := 0; count < 10; count++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			params, _ := request["params"].(map[string]any)
			if sessionID == "" {
				sessionID, _ = params["session_id"].(string)
				turnID, _ = params["turn_id"].(string)
			}
			if params["session_id"] != sessionID || params["turn_id"] != turnID {
				_ = conn.Close()
				serverDone <- errors.New("run action requests did not share session and turn")
				return
			}
			if params["request_id"] == "" {
				_ = conn.Close()
				serverDone <- errors.New("run action request did not include request_id")
				return
			}
			requests <- request

			method, _ := request["method"].(string)
			cdpMethod, _ := params["method"].(string)
			result := map[string]any{}
			switch {
			case method == "nameSession":
				result = map[string]any{}
			case method == "finalizeTabs":
				result = map[string]any{}
			case method == "createTab":
				result = map[string]any{"id": 123}
			case method == "executeCdp" && cdpMethod == "Runtime.evaluate":
				commandParams, _ := params["commandParams"].(map[string]any)
				expression, _ := commandParams["expression"].(string)
				if expression == "document.readyState" {
					result = map[string]any{"result": map[string]any{"value": "interactive"}}
				} else {
					result = map[string]any{"result": map[string]any{"value": map[string]any{
						"title":      "Browser Use Docs",
						"url":        "https://docs.browser-use.com",
						"readyState": "interactive",
						"text":       "Docs",
					}}}
				}
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"--timeout", "100ms",
		"-c", `
name-session "Docs scan - OBU"
open-tab https://docs.browser-use.com
wait-load domcontentloaded
page-info
finalize-tabs []
`,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got actionRunOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.CurrentTabID != 123 {
		t.Fatalf("expected current tab id 123, got %d", got.CurrentTabID)
	}
	if len(got.Steps) != 5 {
		t.Fatalf("expected 5 action steps, got %d", len(got.Steps))
	}
	if got.Steps[1].Action != "open-tab" || got.Steps[1].TabID != 123 {
		t.Fatalf("expected open-tab step to capture tab id, got %#v", got.Steps[1])
	}
	if got.Steps[1].Risk != "navigation" {
		t.Fatalf("expected open-tab risk navigation, got %#v", got.Steps[1].Risk)
	}
	if got.Steps[3].Risk != "read" {
		t.Fatalf("expected page-info risk read, got %#v", got.Steps[3].Risk)
	}
	nameResult, _ := got.Steps[0].Response["result"].(map[string]any)
	if nameResult["ok"] != true {
		t.Fatalf("expected name-session ok result, got %#v", got.Steps[0].Response)
	}
	pageInfoResult, _ := got.Steps[3].Response["result"].(map[string]any)
	if pageInfoResult["title"] != "Browser Use Docs" {
		t.Fatalf("expected page-info title, got %#v", got.Steps[3].Response)
	}
	finalizeResult, _ := got.Steps[4].Response["result"].(map[string]any)
	if finalizeResult["ok"] != true {
		t.Fatalf("expected finalize-tabs ok result, got %#v", got.Steps[4].Response)
	}

	var methods []string
	var sessions []string
	for request := range requests {
		method, _ := request["method"].(string)
		params, _ := request["params"].(map[string]any)
		sessionID, _ := params["session_id"].(string)
		sessions = append(sessions, sessionID)
		if cdpMethod, _ := params["method"].(string); cdpMethod != "" {
			method += ":" + cdpMethod
		}
		methods = append(methods, method)
	}
	want := []string{
		"nameSession",
		"createTab",
		"attach",
		"executeCdp:Page.navigate",
		"attach",
		"executeCdp:Page.enable",
		"executeCdp:Runtime.evaluate",
		"attach",
		"executeCdp:Runtime.evaluate",
		"finalizeTabs",
	}
	if strings.Join(methods, ",") != strings.Join(want, ",") {
		t.Fatalf("expected methods %v, got %v", want, methods)
	}
	for _, sessionID := range sessions {
		if sessionID != defaultCLISessionID {
			t.Fatalf("expected run action plan to use CLI session %q, got %q", defaultCLISessionID, sessionID)
		}
	}
}

func TestCobraRunActionPlanTraceLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket action plan test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(t.TempDir(), "obu.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			serverDone <- err
			return
		}
		params, _ := request["params"].(map[string]any)
		if params["request_id"] == "" {
			serverDone <- errors.New("expected request_id")
			return
		}
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result":  "pong",
		}); err != nil {
			serverDone <- err
			return
		}
	}()

	tracePath := filepath.Join(t.TempDir(), "trace.jsonl")
	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"--trace-log", tracePath,
		"-c", "ping",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	payload, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one trace line, got %q", string(payload))
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["action"] != "ping" || entry["risk"] != "read" || entry["ok"] != true {
		t.Fatalf("unexpected trace entry %#v", entry)
	}
}

func TestRunActionPlanNormalizesCommonInvokeResults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket action plan test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(t.TempDir(), "obu.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 4; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result any
			switch method {
			case "ping":
				result = "pong"
			case "getTabs", "getUserTabs", "getUserHistory":
				result = []any{map[string]any{"id": float64(123), "title": "Example Domain"}}
			default:
				_ = conn.Close()
				serverDone <- fmt.Errorf("unexpected method %q", method)
				return
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"-c", `
ping
tabs
user-tabs
history --query example --limit 1
`,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got actionRunOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Steps) != 4 {
		t.Fatalf("expected 4 action steps, got %d", len(got.Steps))
	}
	pingResult, _ := got.Steps[0].Response["result"].(map[string]any)
	if pingResult["status"] != "pong" {
		t.Fatalf("expected normalized ping status, got %#v", got.Steps[0].Response)
	}
	for index, action := range []string{"tabs", "user-tabs", "history"} {
		result, _ := got.Steps[index+1].Response["result"].(map[string]any)
		items, _ := result["items"].([]any)
		if len(items) != 1 {
			t.Fatalf("expected normalized %s items, got %#v", action, got.Steps[index+1].Response)
		}
	}
}

func TestRunActionPlanWritesPartialOutputOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket action plan test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(t.TempDir(), "obu.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			serverDone <- err
			return
		}
		if method, _ := request["method"].(string); method != "ping" {
			serverDone <- fmt.Errorf("unexpected method %q", method)
			return
		}
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result":  "pong",
		}); err != nil {
			serverDone <- err
			return
		}
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"-c", `
ping
unsupported-action
`,
	})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected run to fail")
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got actionRunOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.OK {
		t.Fatalf("expected failed run output, got %#v", got)
	}
	if len(got.Steps) != 2 {
		t.Fatalf("expected successful and failed steps, got %#v", got.Steps)
	}
	if !got.Steps[0].OK {
		t.Fatalf("expected first step ok, got %#v", got.Steps[0])
	}
	if got.Steps[1].OK || got.Steps[1].Error == "" {
		t.Fatalf("expected second step error, got %#v", got.Steps[1])
	}
}

func TestRunActionPlanContinueOnError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket action plan test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(t.TempDir(), "obu.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			if method, _ := request["method"].(string); method != "ping" {
				_ = conn.Close()
				serverDone <- fmt.Errorf("unexpected method %q", method)
				return
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  "pong",
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"--continue-on-error",
		"-c", `
ping
unsupported-action
ping
`,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got actionRunOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.OK {
		t.Fatalf("expected run output to record failure, got %#v", got)
	}
	if len(got.Steps) != 3 {
		t.Fatalf("expected all steps with continue-on-error, got %#v", got.Steps)
	}
	if !got.Steps[0].OK || got.Steps[1].OK || !got.Steps[2].OK {
		t.Fatalf("unexpected step statuses: %#v", got.Steps)
	}
}

func TestCobraPageInfoJSON(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result map[string]any
			switch method {
			case "attach":
				result = map[string]any{}
			case "executeCdp":
				result = map[string]any{
					"result": map[string]any{
						"value": map[string]any{
							"title":      "Example Domain",
							"url":        "https://example.com",
							"readyState": "complete",
							"text":       "Example Domain",
						},
					},
				}
			default:
				_ = conn.Close()
				serverDone <- fmt.Errorf("unexpected method %q", method)
				return
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"page-info",
		"--socket", socketPath,
		"--tab-id", "123",
		"--json",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["title"] != "Example Domain" {
		t.Fatalf("expected title %q, got %#v", "Example Domain", got["title"])
	}
	if got["readyState"] != "complete" {
		t.Fatalf("expected readyState %q, got %#v", "complete", got["readyState"])
	}
}

func TestCobraTextJSONWrapsTextField(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result map[string]any
			switch method {
			case "attach":
				result = map[string]any{}
			case "executeCdp":
				result = map[string]any{
					"result": map[string]any{
						"value": "Example Domain",
					},
				}
			default:
				_ = conn.Close()
				serverDone <- fmt.Errorf("unexpected method %q", method)
				return
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"text",
		"--socket", socketPath,
		"--tab-id", "123",
		"--json",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["text"] != "Example Domain" {
		t.Fatalf("expected text %q, got %#v", "Example Domain", got["text"])
	}
}

func TestCobraSnapshotJSONWrapsItemsField(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result map[string]any
			switch method {
			case "attach":
				result = map[string]any{}
			case "executeCdp":
				result = map[string]any{
					"result": map[string]any{
						"value": []any{
							map[string]any{
								"index":    1,
								"tag":      "a",
								"text":     "Learn more",
								"type":     "",
								"href":     "https://iana.org/domains/example",
								"selector": "",
							},
						},
					},
				}
			default:
				_ = conn.Close()
				serverDone <- fmt.Errorf("unexpected method %q", method)
				return
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"snapshot",
		"--socket", socketPath,
		"--tab-id", "123",
		"--json",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 snapshot item, got %d", len(got.Items))
	}
	if got.Items[0]["text"] != "Learn more" {
		t.Fatalf("expected snapshot text %q, got %#v", "Learn more", got.Items[0]["text"])
	}
}

func TestRunActionPlanSupportsTextAndSnapshot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	requests := make(chan map[string]any, 10)
	serverDone := make(chan error, 1)
	go func() {
		defer close(requests)
		defer close(serverDone)
		for i := 0; i < 5; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			requests <- request
			method, _ := request["method"].(string)
			var result any = map[string]any{}
			if method == "executeCdp" {
				params, _ := request["params"].(map[string]any)
				commandParams, _ := params["commandParams"].(map[string]any)
				expression, _ := commandParams["expression"].(string)
				if strings.Contains(expression, "querySelectorAll") {
					result = map[string]any{
						"result": map[string]any{
							"value": []any{
								map[string]any{"index": 1, "tag": "a", "text": "Docs", "href": "https://example.com/docs"},
							},
						},
					}
				} else {
					result = map[string]any{
						"result": map[string]any{
							"value": "Example text",
						},
					}
				}
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"-c", `
claim-tab 123
text --max-chars 20
snapshot --limit 1
`,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got actionRunOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Steps) != 3 {
		t.Fatalf("expected 3 action steps, got %d", len(got.Steps))
	}
	textResult, _ := got.Steps[1].Response["result"].(map[string]any)
	if textResult["text"] != "Example text" {
		t.Fatalf("expected text result, got %#v", got.Steps[1].Response)
	}
	snapshotResult, _ := got.Steps[2].Response["result"].(map[string]any)
	items, _ := snapshotResult["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one snapshot item, got %#v", snapshotResult)
	}
}

func TestRunActionPlanSupportsRobustClickAndFill(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 5; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result any = map[string]any{}
			if method == "executeCdp" {
				params, _ := request["params"].(map[string]any)
				commandParams, _ := params["commandParams"].(map[string]any)
				expression, _ := commandParams["expression"].(string)
				action := "click"
				if strings.Contains(expression, `action: "fill"`) {
					action = "fill"
				}
				result = map[string]any{
					"result": map[string]any{
						"value": map[string]any{
							"ok":     true,
							"ref":    "1",
							"action": action,
							"tag":    "input",
							"text":   "Hello",
						},
					},
				}
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"-c", `
claim-tab 123
click @1
fill @1 hello
`,
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got actionRunOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Steps) != 3 {
		t.Fatalf("expected 3 action steps, got %d", len(got.Steps))
	}
	clickResult, _ := got.Steps[1].Response["result"].(map[string]any)
	if clickResult["action"] != "click" || clickResult["ok"] != true {
		t.Fatalf("expected click result, got %#v", clickResult)
	}
	fillResult, _ := got.Steps[2].Response["result"].(map[string]any)
	if fillResult["action"] != "fill" || fillResult["ok"] != true {
		t.Fatalf("expected fill result, got %#v", fillResult)
	}
}

func TestRunActionPlanSupportsSelectorScreenshot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	outputPath := filepath.Join(t.TempDir(), "shot.png")
	pngData := base64.StdEncoding.EncodeToString([]byte("png"))
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 4; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result any = map[string]any{}
			if method == "executeCdp" {
				params, _ := request["params"].(map[string]any)
				cdpMethod, _ := params["method"].(string)
				if cdpMethod == "Runtime.evaluate" {
					result = map[string]any{
						"result": map[string]any{
							"value": map[string]any{
								"ok": true,
								"clip": map[string]any{
									"x": 1, "y": 2, "width": 30, "height": 40, "scale": 1,
								},
							},
						},
					}
				} else if cdpMethod == "Page.captureScreenshot" {
					result = map[string]any{"data": pngData}
				}
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"-c", fmt.Sprintf(`
claim-tab 123
screenshot --selector main --output %q
`, outputPath),
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got actionRunOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Steps) != 2 {
		t.Fatalf("expected 2 action steps, got %d", len(got.Steps))
	}
	result, _ := got.Steps[1].Response["result"].(map[string]any)
	if result["path"] != outputPath {
		t.Fatalf("expected screenshot path %q, got %#v", outputPath, result)
	}
	if result["bytes"].(float64) != 3 {
		t.Fatalf("expected byte count 3, got %#v", result["bytes"])
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected screenshot file: %v", err)
	}
}

func TestRunActionPlanRetriesEmptyScreenshotData(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	outputPath := filepath.Join(t.TempDir(), "shot.png")
	pngData := base64.StdEncoding.EncodeToString([]byte("png"))
	var captureAttempts int
	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 4; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result any = map[string]any{}
			if method == "executeCdp" {
				params, _ := request["params"].(map[string]any)
				if cdpMethod, _ := params["method"].(string); cdpMethod == "Page.captureScreenshot" {
					captureAttempts++
					if captureAttempts == 2 {
						result = map[string]any{"data": pngData}
					}
				}
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"run",
		"--socket", socketPath,
		"-c", fmt.Sprintf(`
claim-tab 123
screenshot --output %q
`, outputPath),
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	if captureAttempts != 2 {
		t.Fatalf("expected screenshot retry, got %d attempts", captureAttempts)
	}
	var got actionRunOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	result, _ := got.Steps[1].Response["result"].(map[string]any)
	if result["bytes"].(float64) != 3 {
		t.Fatalf("expected screenshot result after retry, got %#v", result)
	}
}

func TestCobraHistoryJSONWrapsItemsField(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			_ = conn.Close()
			serverDone <- err
			return
		}
		method, _ := request["method"].(string)
		if method != "getUserHistory" {
			_ = conn.Close()
			serverDone <- fmt.Errorf("unexpected method %q", method)
			return
		}
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result": []any{
				map[string]any{
					"title": "Example Domain",
					"url":   "https://example.com/",
				},
			},
		}); err != nil {
			_ = conn.Close()
			serverDone <- err
			return
		}
		_ = conn.Close()
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"history",
		"--socket", socketPath,
		"--query", "example",
		"--limit", "1",
		"--json",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(got.Items))
	}
	if got.Items[0]["title"] != "Example Domain" {
		t.Fatalf("expected history title %q, got %#v", "Example Domain", got.Items[0]["title"])
	}
}

func TestCobraWaitJSONWrapsReadyStateField(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result map[string]any
			switch method {
			case "attach":
				result = map[string]any{}
			case "executeCdp":
				result = map[string]any{
					"result": map[string]any{
						"value": "complete",
					},
				}
			default:
				_ = conn.Close()
				serverDone <- fmt.Errorf("unexpected method %q", method)
				return
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{
		"wait",
		"--socket", socketPath,
		"--tab-id", "123",
		"--json",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["readyState"] != "complete" {
		t.Fatalf("expected readyState %q, got %#v", "complete", got["readyState"])
	}
}

func TestCobraPingJSONWrapsStatusField(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			_ = conn.Close()
			serverDone <- err
			return
		}
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result":  "pong",
		}); err != nil {
			_ = conn.Close()
			serverDone <- err
			return
		}
		_ = conn.Close()
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"ping", "--socket", socketPath, "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "pong" {
		t.Fatalf("expected status %q, got %#v", "pong", got["status"])
	}
}

func TestCobraClaimTabJSONWrapsTabField(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			_ = conn.Close()
			serverDone <- err
			return
		}
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result": map[string]any{
				"id":    123,
				"title": "Example Domain",
			},
		}); err != nil {
			_ = conn.Close()
			serverDone <- err
			return
		}
		_ = conn.Close()
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"claim-tab", "--socket", socketPath, "--tab-id", "123", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got struct {
		Tab map[string]any `json:"tab"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Tab["id"] != float64(123) {
		t.Fatalf("expected claimed tab id 123, got %#v", got.Tab["id"])
	}
}

func TestCobraNavigateJSONWrapsNavigateField(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		for i := 0; i < 2; i++ {
			conn, err := listener.Accept()
			if err != nil {
				serverDone <- err
				return
			}
			var request map[string]any
			if err := wire.ReadJSON(conn, &request); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			method, _ := request["method"].(string)
			var result map[string]any
			switch method {
			case "attach":
				result = map[string]any{}
			case "executeCdp":
				result = map[string]any{
					"frameId":    "FRAME-1",
					"isDownload": false,
				}
			default:
				_ = conn.Close()
				serverDone <- fmt.Errorf("unexpected method %q", method)
				return
			}
			if err := wire.WriteJSON(conn, map[string]any{
				"jsonrpc": "2.0",
				"id":      request["id"],
				"result":  result,
			}); err != nil {
				_ = conn.Close()
				serverDone <- err
				return
			}
			_ = conn.Close()
		}
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"navigate", "--socket", socketPath, "--tab-id", "123", "--url", "https://example.com", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got struct {
		Navigate map[string]any `json:"navigate"`
	}
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Navigate["frameId"] != "FRAME-1" {
		t.Fatalf("expected navigate frame id %q, got %#v", "FRAME-1", got.Navigate["frameId"])
	}
}

func TestCobraFinalizeTabsJSONWrapsOKField(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		defer close(serverDone)
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			_ = conn.Close()
			serverDone <- err
			return
		}
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result":  map[string]any{},
		}); err != nil {
			_ = conn.Close()
			serverDone <- err
			return
		}
		_ = conn.Close()
		serverDone <- nil
	}()

	cmd := newRootCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"finalize-tabs", "--socket", socketPath, "--keep", "[]", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	var got map[string]any
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["ok"] != true {
		t.Fatalf("expected ok true, got %#v", got["ok"])
	}
}

func TestInvokeWithOptionsUsesSessionID(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows TCP relay")
	}
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("obu-test-%d.sock", time.Now().UnixNano()))
	defer os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	requests := make(chan map[string]any, 1)
	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		var request map[string]any
		if err := wire.ReadJSON(conn, &request); err != nil {
			serverDone <- err
			return
		}
		requests <- request
		if err := wire.WriteJSON(conn, map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result":  []any{},
		}); err != nil {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	if _, err := invokeWithOptions(socketOptions{
		socketPath: socketPath,
		timeout:    time.Second,
		sessionID:  "custom-cli-session",
	}, "getTabs", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	request := <-requests
	params, _ := request["params"].(map[string]any)
	if params["session_id"] != "custom-cli-session" {
		t.Fatalf("expected custom session id, got %#v", params["session_id"])
	}
	if params["request_id"] == "" {
		t.Fatalf("expected request_id, got %#v", params)
	}
}

func testCRX(crxID []byte) []byte {
	signedHeaderData := testProtoBytes(1, crxID)
	header := testProtoBytes(10000, signedHeaderData)
	prefix := make([]byte, 12)
	copy(prefix[0:4], "Cr24")
	prefix[4] = 3
	prefix[8] = byte(len(header))
	prefix[9] = byte(len(header) >> 8)
	prefix[10] = byte(len(header) >> 16)
	prefix[11] = byte(len(header) >> 24)
	return append(append(prefix, header...), []byte("zip")...)
}

func testProtoBytes(fieldNumber uint64, value []byte) []byte {
	key := testVarint((fieldNumber << 3) | 2)
	length := testVarint(uint64(len(value)))
	payload := append(key, length...)
	return append(payload, value...)
}

func testVarint(value uint64) []byte {
	var out []byte
	for value > 0x7f {
		out = append(out, byte(value&0x7f)|0x80)
		value >>= 7
	}
	return append(out, byte(value))
}

func writeTestExtensionZIP(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	files := map[string]string{
		"manifest.json": `{"manifest_version":3,"name":"Open Browser Use","version":"0.1.0","background":{"service_worker":"background.js"},"permissions":["nativeMessaging"]}`,
		"background.js": "chrome.runtime.onInstalled.addListener(() => {});\n",
	}
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			_ = writer.Close()
			return err
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			_ = writer.Close()
			return err
		}
	}
	return writer.Close()
}

func readManifestFromZIP(path string) ([]byte, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != "manifest.json" {
			continue
		}
		source, err := file.Open()
		if err != nil {
			return nil, err
		}
		payload, readErr := io.ReadAll(source)
		closeErr := source.Close()
		if readErr != nil {
			return nil, readErr
		}
		return payload, closeErr
	}
	return nil, errors.New("manifest.json not found in ZIP")
}
