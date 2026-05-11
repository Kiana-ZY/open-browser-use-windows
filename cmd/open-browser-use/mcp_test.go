package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Kiana-ZY/open-browser-use-windows/internal/wire"
)

func TestMCPInitializeAndListTools(t *testing.T) {
	server := newMCPServer(socketOptions{socketDir: t.TempDir(), timeout: time.Second})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		"",
	}, "\n")
	var output bytes.Buffer

	if err := server.serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}

	responses := decodeMCPResponses(t, output.Bytes())
	if len(responses) != 2 {
		t.Fatalf("expected initialize and tools/list responses, got %#v", responses)
	}
	initializeResult, _ := responses[0]["result"].(map[string]any)
	if initializeResult["protocolVersion"] != mcpProtocolVersion {
		t.Fatalf("expected protocol version %q, got %#v", mcpProtocolVersion, initializeResult["protocolVersion"])
	}
	capabilities, _ := initializeResult["capabilities"].(map[string]any)
	if _, ok := capabilities["tools"].(map[string]any); !ok {
		t.Fatalf("expected tools capability, got %#v", capabilities)
	}

	listResult, _ := responses[1]["result"].(map[string]any)
	tools, _ := listResult["tools"].([]any)
	if len(tools) == 0 {
		t.Fatalf("expected tools/list to return tools, got %#v", listResult)
	}
	names := map[string]bool{}
	for _, rawTool := range tools {
		tool, _ := rawTool.(map[string]any)
		name, _ := tool["name"].(string)
		names[name] = true
		if _, ok := tool["inputSchema"].(map[string]any); !ok {
			t.Fatalf("expected tool %q to include inputSchema, got %#v", name, tool)
		}
	}
	for _, name := range []string{"user_tabs", "open_tab", "text", "snapshot", "click", "fill", "screenshot", "cdp", "run_action_plan"} {
		if !names[name] {
			t.Fatalf("expected MCP tools to include %q, got %#v", name, names)
		}
	}
}

func TestMCPToolCallInvokesBrowserSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-mcp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socketPath := filepath.Join(socketDir, "browser.sock")
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
		response := map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result": []any{
				map[string]any{"id": 321, "title": "Example", "url": "https://example.com"},
			},
		}
		if err := wire.WriteJSON(conn, response); err != nil {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	server := newMCPServer(socketOptions{socketPath: socketPath, timeout: time.Second})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"user_tabs","arguments":{}}}`,
		"",
	}, "\n")
	var output bytes.Buffer

	if err := server.serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
	browserRequest := <-requests
	if browserRequest["method"] != "getUserTabs" {
		t.Fatalf("expected getUserTabs browser request, got %#v", browserRequest["method"])
	}
	params, _ := browserRequest["params"].(map[string]any)
	if params["session_id"] == "" || params["turn_id"] == "" {
		t.Fatalf("expected MCP tool call to include session and turn ids, got %#v", params)
	}

	responses := decodeMCPResponses(t, output.Bytes())
	if len(responses) != 2 {
		t.Fatalf("expected initialize and tools/call responses, got %#v", responses)
	}
	callResult, _ := responses[1]["result"].(map[string]any)
	if callResult["isError"] != false {
		t.Fatalf("expected successful tool result, got %#v", callResult)
	}
	structured, ok := callResult["structuredContent"].(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent with browser response, got %#v", callResult)
	}
	items, _ := structured["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected normalized items payload, got %#v", structured)
	}
	content, _ := callResult["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected text content mirror, got %#v", callResult["content"])
	}
}

func TestMCPToolCallNormalizesPingResult(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-mcp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socketPath := filepath.Join(socketDir, "browser.sock")
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
		if request["method"] != "ping" {
			serverDone <- fmt.Errorf("expected ping browser request, got %#v", request["method"])
			return
		}
		response := map[string]any{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result":  "pong",
		}
		if err := wire.WriteJSON(conn, response); err != nil {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	server := newMCPServer(socketOptions{socketPath: socketPath, timeout: time.Second})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"ping","arguments":{}}}`,
		"",
	}, "\n")
	var output bytes.Buffer

	if err := server.serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	responses := decodeMCPResponses(t, output.Bytes())
	callResult, _ := responses[1]["result"].(map[string]any)
	structured, _ := callResult["structuredContent"].(map[string]any)
	if structured["status"] != "pong" {
		t.Fatalf("expected normalized ping payload, got %#v", structured)
	}
}

func TestMCPToolCallTraceLog(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-mcp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socketPath := filepath.Join(socketDir, "browser.sock")
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
		if request["method"] != "getUserTabs" {
			serverDone <- fmt.Errorf("expected getUserTabs browser request, got %#v", request["method"])
			return
		}
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

	tracePath := filepath.Join(t.TempDir(), "mcp-trace.jsonl")
	server := newMCPServer(socketOptions{socketPath: socketPath, timeout: time.Second, traceLog: tracePath})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"user_tabs","arguments":{}}}`,
		"",
	}, "\n")
	var output bytes.Buffer

	if err := server.serve(strings.NewReader(input), &output); err != nil {
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
	if entry["action"] != "user-tabs" || entry["risk"] != "read" || entry["ok"] != true {
		t.Fatalf("unexpected trace entry %#v", entry)
	}
	if entry["sessionId"] == "" || entry["turnId"] == "" {
		t.Fatalf("expected session and turn ids in trace entry %#v", entry)
	}
}

func TestMCPToolCallNormalizesPageInfoResult(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-mcp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socketPath := filepath.Join(socketDir, "browser.sock")
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
							"text":       "Example text",
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

	server := newMCPServer(socketOptions{socketPath: socketPath, timeout: time.Second})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"call-1","method":"tools/call","params":{"name":"page_info","arguments":{"tab_id":123}}}`,
		"",
	}, "\n")
	var output bytes.Buffer

	if err := server.serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	responses := decodeMCPResponses(t, output.Bytes())
	callResult, _ := responses[1]["result"].(map[string]any)
	structured, _ := callResult["structuredContent"].(map[string]any)
	if structured["title"] != "Example Domain" {
		t.Fatalf("expected flattened page_info payload, got %#v", structured)
	}
	if structured["readyState"] != "complete" {
		t.Fatalf("expected readyState %q, got %#v", "complete", structured["readyState"])
	}
}

func TestMCPToolCallTextAndSnapshotResults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-mcp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socketPath := filepath.Join(socketDir, "browser.sock")
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

	server := newMCPServer(socketOptions{socketPath: socketPath, timeout: time.Second})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"text","method":"tools/call","params":{"name":"text","arguments":{"tab_id":123,"max_chars":20}}}`,
		`{"jsonrpc":"2.0","id":"snapshot","method":"tools/call","params":{"name":"snapshot","arguments":{"tab_id":123,"limit":1}}}`,
		"",
	}, "\n")
	var output bytes.Buffer

	if err := server.serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	responses := decodeMCPResponses(t, output.Bytes())
	textResult, _ := responses[1]["result"].(map[string]any)
	textStructured, _ := textResult["structuredContent"].(map[string]any)
	if textStructured["text"] != "Example text" {
		t.Fatalf("expected text payload, got %#v", textStructured)
	}
	snapshotResult, _ := responses[2]["result"].(map[string]any)
	snapshotStructured, _ := snapshotResult["structuredContent"].(map[string]any)
	items, _ := snapshotStructured["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one snapshot item, got %#v", snapshotStructured)
	}
}

func TestMCPToolCallClickAndFillResults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-mcp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socketPath := filepath.Join(socketDir, "browser.sock")
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

	server := newMCPServer(socketOptions{socketPath: socketPath, timeout: time.Second})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"click","method":"tools/call","params":{"name":"click","arguments":{"tab_id":123,"ref":"@1"}}}`,
		`{"jsonrpc":"2.0","id":"fill","method":"tools/call","params":{"name":"fill","arguments":{"tab_id":123,"ref":"@1","text":"hello"}}}`,
		"",
	}, "\n")
	var output bytes.Buffer

	if err := server.serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	responses := decodeMCPResponses(t, output.Bytes())
	clickResult, _ := responses[1]["result"].(map[string]any)
	clickStructured, _ := clickResult["structuredContent"].(map[string]any)
	if clickStructured["action"] != "click" || clickStructured["ok"] != true {
		t.Fatalf("expected click payload, got %#v", clickStructured)
	}
	fillResult, _ := responses[2]["result"].(map[string]any)
	fillStructured, _ := fillResult["structuredContent"].(map[string]any)
	if fillStructured["action"] != "fill" || fillStructured["ok"] != true {
		t.Fatalf("expected fill payload, got %#v", fillStructured)
	}
}

func TestMCPToolCallScreenshotResult(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix socket test not compatible with Windows")
	}
	socketDir, err := os.MkdirTemp("/tmp", "obu-mcp-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(socketDir)
	socketPath := filepath.Join(socketDir, "browser.sock")
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
		for i := 0; i < 3; i++ {
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
				if cdpMethod == "Page.captureScreenshot" {
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

	server := newMCPServer(socketOptions{socketPath: socketPath, timeout: time.Second})
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		fmt.Sprintf(`{"jsonrpc":"2.0","id":"shot","method":"tools/call","params":{"name":"screenshot","arguments":{"tab_id":123,"output":%q}}}`, outputPath),
		"",
	}, "\n")
	var output bytes.Buffer

	if err := server.serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}

	responses := decodeMCPResponses(t, output.Bytes())
	shotResult, _ := responses[1]["result"].(map[string]any)
	structured, _ := shotResult["structuredContent"].(map[string]any)
	if structured["path"] != outputPath {
		t.Fatalf("expected screenshot path %q, got %#v", outputPath, structured)
	}
	if structured["bytes"].(float64) != 3 {
		t.Fatalf("expected byte count 3, got %#v", structured["bytes"])
	}
}

func decodeMCPResponses(t *testing.T, payload []byte) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(string(payload)), "\n")
	responses := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("failed to decode MCP response %q: %v", line, err)
		}
		responses = append(responses, response)
	}
	return responses
}
