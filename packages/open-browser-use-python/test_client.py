import json
import socket
import struct
import threading
import unittest
from contextlib import contextmanager
from typing import Callable

from open_browser_use.client import OpenBrowserUseClient, connect_open_browser_use, encode_frame


class OpenBrowserUseClientTest(unittest.TestCase):
    def test_encode_frame(self) -> None:
        frame = encode_frame({"id": 1, "method": "getInfo"})
        (length,) = struct.unpack("=I", frame[:4])
        self.assertEqual(length, len(frame) - 4)
        self.assertEqual(json.loads(frame[4:]), {"id": 1, "method": "getInfo"})

    def test_request_round_trip(self) -> None:
        with start_test_server(serve_request_round_trip) as (socket_path, thread):
            client = OpenBrowserUseClient(socket_path=socket_path)
            try:
                self.assertEqual(client.get_info(), {"name": "Open Browser Use"})
            finally:
                client.close()
            thread.join(timeout=1)

    def test_file_chooser_wrapper(self) -> None:
        def serve_once(conn: socket.socket) -> None:
            request = read_request(conn)
            self.assertEqual(request["method"], "waitForFileChooser")
            self.assertEqual(request["params"]["tabId"], 123)
            conn.sendall(
                encode_frame(
                    {
                        "jsonrpc": "2.0",
                        "id": request["id"],
                        "result": {"fileChooserId": "chooser-1", "isMultiple": False},
                    }
                )
            )

        with start_test_server(serve_once) as (socket_path, thread):
            client = OpenBrowserUseClient(socket_path=socket_path)
            try:
                self.assertEqual(
                    client.wait_for_file_chooser(123),
                    {"fileChooserId": "chooser-1", "isMultiple": False},
                )
            finally:
                client.close()
            thread.join(timeout=1)

    def test_download_and_clipboard_wrappers(self) -> None:
        expected = [
            ("waitForDownload", {"tabId": 123, "timeoutMs": 5000}),
            ("downloadPath", {"downloadId": "download-1"}),
            ("readClipboardText", {"tabId": 123}),
            ("writeClipboardText", {"tabId": 123, "text": "hello"}),
        ]

        def serve(conn: socket.socket) -> None:
            for method, params in expected:
                request = read_request(conn)
                self.assertEqual(request["method"], method)
                for key, value in params.items():
                    self.assertEqual(request["params"][key], value)
                conn.sendall(encode_frame({"jsonrpc": "2.0", "id": request["id"], "result": {}}))

        with start_test_server(serve) as (socket_path, thread):
            client = OpenBrowserUseClient(socket_path=socket_path)
            try:
                client.wait_for_download(123, timeout_ms=5000)
                client.download_path("download-1")
                client.read_clipboard_text(123)
                client.write_clipboard_text(123, "hello")
            finally:
                client.close()
            thread.join(timeout=1)

    def test_high_level_browser_tab_helpers(self) -> None:
        calls: list[tuple[str, str | None]] = []

        def serve(conn: socket.socket) -> None:
            while True:
                try:
                    request = read_request(conn)
                except EOFError:
                    break
                method = request["method"]
                cdp_method = request["params"].get("method")
                calls.append((method, cdp_method))
                result = {}
                if method == "createTab":
                    result = {"id": 123}
                elif method == "executeCdp" and cdp_method == "Page.navigate":
                    conn.sendall(
                        encode_frame(
                            {
                                "jsonrpc": "2.0",
                                "method": "onCDPEvent",
                                "params": {
                                    "source": {"tabId": 123},
                                    "method": "Page.domContentEventFired",
                                    "params": {},
                                },
                            }
                        )
                    )
                    result = {"frameId": "frame-1"}
                elif method == "executeCdp" and cdp_method == "Runtime.evaluate":
                    expression = request["params"]["commandParams"]["expression"]
                    if "readyState" in expression:
                        result = {
                            "result": {
                                "value": {
                                    "href": "https://example.test/issues",
                                    "readyState": "interactive",
                                }
                            }
                        }
                    elif "document.title" in expression:
                        result = {"result": {"value": "Issues - open-codex-computer-use"}}
                    elif "location.href" in expression:
                        result = {"result": {"value": "https://example.test/issues"}}
                    elif "document.querySelector" in expression:
                        result = {"result": {"value": "Open\nClosed\nIssues\nStarred"}}
                    else:
                        result = {"result": {"value": "Open\nClosed\nIssues\nStarred"}}
                conn.sendall(encode_frame({"jsonrpc": "2.0", "id": request["id"], "result": result}))

        with start_test_server(serve) as (socket_path, thread):
            browser = connect_open_browser_use(socket_path=socket_path)
            notifications = []
            browser.client.on_notification(notifications.append)
            try:
                tab = browser.new_tab()
                tab.goto("https://example.test/issues", wait_until="domcontentloaded", timeout=1)
                tab.playwright.wait_for_load_state(state="domcontentloaded", timeout=1)
                tab.playwright.wait_for_timeout(1)
                self.assertEqual(tab.title(), "Issues - open-codex-computer-use")
                self.assertEqual(tab.playwright.url(), "https://example.test/issues")
                self.assertEqual(tab.playwright.dom_snapshot(), "Open\nClosed\nIssues\nStarred")
                self.assertEqual(tab.playwright.locator("body").inner_text(timeout_ms=1000), "Open\nClosed\nIssues\nStarred")
                self.assertEqual(notifications[0]["method"], "onCDPEvent")
                self.assertEqual(
                    calls,
                    [
                        ("createTab", None),
                        ("attach", None),
                        ("executeCdp", "Page.enable"),
                        ("executeCdp", "Page.navigate"),
                        ("executeCdp", "Page.enable"),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Page.enable"),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Runtime.evaluate"),
                    ],
                )
            finally:
                browser.close()
            thread.join(timeout=1)

    def test_structured_read_screenshot_and_interaction_helpers(self) -> None:
        calls: list[tuple[str, str | None]] = []
        png_data = "cG5n"

        def serve(conn: socket.socket) -> None:
            for _ in range(7):
                request = read_request(conn)
                method = request["method"]
                cdp_method = request["params"].get("method")
                expression = request["params"].get("commandParams", {}).get("expression", "")
                calls.append((method, cdp_method))
                result = {}
                if method == "executeCdp" and cdp_method == "Runtime.evaluate":
                    if "document.title" in expression:
                        result = {
                            "result": {
                                "value": {
                                    "title": "Example",
                                    "url": "https://example.test",
                                    "readyState": "complete",
                                    "text": "Hello",
                                }
                            }
                        }
                    elif "__obu_refs" in expression:
                        result = {
                            "result": {
                                "value": [
                                    {
                                        "index": 1,
                                        "tag": "button",
                                        "text": "Save",
                                        "type": "",
                                        "href": "",
                                        "selector": "#save",
                                    }
                                ]
                            }
                        }
                    elif 'action: "click"' in expression:
                        result = {
                            "result": {
                                "value": {
                                    "ok": True,
                                    "ref": "1",
                                    "action": "click",
                                    "tag": "button",
                                    "text": "Save",
                                }
                            }
                        }
                    elif 'action: "fill"' in expression:
                        result = {
                            "result": {
                                "value": {
                                    "ok": True,
                                    "ref": "2",
                                    "action": "fill",
                                    "tag": "input",
                                    "text": "hello",
                                    "valueLength": 5,
                                }
                            }
                        }
                    else:
                        result = {"result": {"value": "Hello"}}
                elif method == "executeCdp" and cdp_method == "Page.captureScreenshot":
                    result = {"data": png_data}
                conn.sendall(encode_frame({"jsonrpc": "2.0", "id": request["id"], "result": result}))

        with start_test_server(serve) as (socket_path, thread):
            browser = connect_open_browser_use(socket_path=socket_path)
            try:
                tab = browser.tab(123)
                self.assertEqual(
                    tab.page_info(max_chars=5),
                    {
                        "title": "Example",
                        "url": "https://example.test",
                        "readyState": "complete",
                        "text": "Hello",
                    },
                )
                self.assertEqual(tab.text_result(selector="main", max_chars=5), {"text": "Hello"})
                self.assertEqual(
                    tab.snapshot(limit=1),
                    {
                        "items": [
                            {
                                "index": 1,
                                "tag": "button",
                                "text": "Save",
                                "type": "",
                                "href": "",
                                "selector": "#save",
                            }
                        ]
                    },
                )
                self.assertEqual(
                    tab.screenshot_result(),
                    {"data": png_data, "bytes": 3, "format": "png", "tabId": 123},
                )
                self.assertEqual(tab.click("@1")["action"], "click")
                self.assertEqual(tab.fill("@2", "hello")["valueLength"], 5)
                self.assertEqual(
                    calls,
                    [
                        ("attach", None),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Page.captureScreenshot"),
                        ("executeCdp", "Runtime.evaluate"),
                        ("executeCdp", "Runtime.evaluate"),
                    ],
                )
            finally:
                browser.close()
            thread.join(timeout=1)


def serve_request_round_trip(conn: socket.socket) -> None:
    request = read_request(conn)
    conn.sendall(
        encode_frame(
            {
                "jsonrpc": "2.0",
                "id": request["id"],
                "result": {"name": "Open Browser Use"},
            }
        )
    )


@contextmanager
def start_test_server(handler: Callable[[socket.socket], None]):
    server = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    server.bind(("127.0.0.1", 0))
    server.listen(1)
    socket_path = f"127.0.0.1:{server.getsockname()[1]}"

    def run() -> None:
        try:
            conn, _ = server.accept()
            with conn:
                handler(conn)
        finally:
            server.close()

    thread = threading.Thread(target=run)
    thread.start()
    try:
        yield socket_path, thread
    finally:
        server.close()


def read_request(conn: socket.socket) -> dict:
    header = conn.recv(4)
    if not header:
        raise EOFError("socket closed")
    (length,) = struct.unpack("=I", header)
    payload = b""
    while len(payload) < length:
        chunk = conn.recv(length - len(payload))
        if not chunk:
            raise EOFError("socket closed")
        payload += chunk
    value = json.loads(payload)
    if not isinstance(value, dict):
        raise RuntimeError("request must be an object")
    return value


if __name__ == "__main__":
    unittest.main()
