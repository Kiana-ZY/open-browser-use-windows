from __future__ import annotations

import base64
import json
import os
import socket
import struct
import sys
import time
from dataclasses import dataclass
from typing import Any, Callable, Literal


JsonObject = dict[str, Any]
LoadState = Literal["domcontentloaded", "load"]
NotificationHandler = Callable[[JsonObject], None]
DEFAULT_NAVIGATION_TIMEOUT = 10.0
TCP_PORT = 19832


@dataclass
class OpenBrowserUseClient:
    socket_path: str
    session_id: str = "open-browser-use-python"
    turn_id: str | None = None
    timeout: float = 10.0

    def __post_init__(self) -> None:
        if self.turn_id is None:
            self.turn_id = f"turn-{time.time_ns()}"
        self._next_id = 1
        self._socket: socket.socket | None = None
        self._notification_handlers: list[NotificationHandler] = []

    def connect(self) -> "OpenBrowserUseClient":
        if self._socket is None:
            if self._connect_tcp_socket_path():
                return self
            if sys.platform == "win32":
                self._connect_windows()
            else:
                self._connect_unix()
        return self

    def _connect_tcp_socket_path(self) -> bool:
        if ":" not in self.socket_path:
            return False
        host, port_text = self.socket_path.rsplit(":", 1)
        if not host or not port_text.isdigit():
            return False
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(self.timeout)
        sock.connect((host, int(port_text)))
        self._socket = sock
        return True

    def _connect_unix(self) -> None:
        sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        sock.settimeout(self.timeout)
        sock.connect(self.socket_path)
        self._socket = sock

    def _connect_windows(self) -> None:
        """Connect to the relay TCP listener on localhost."""
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(self.timeout)
        sock.connect(("127.0.0.1", TCP_PORT))
        self._socket = sock

    def close(self) -> None:
        if self._socket is not None:
            self._socket.close()
            self._socket = None

    def on_notification(self, handler: NotificationHandler) -> Callable[[], None]:
        self._notification_handlers.append(handler)

        def remove() -> None:
            if handler in self._notification_handlers:
                self._notification_handlers.remove(handler)

        return remove

    def request(self, method: str, params: JsonObject | None = None) -> Any:
        self.connect()
        if self._socket is None:
            raise RuntimeError("Open Browser Use socket is not connected")
        request_id = self._next_id
        self._next_id += 1
        merged_params: JsonObject = {
            "session_id": self.session_id,
            "turn_id": self.turn_id,
        }
        if params:
            merged_params.update(params)
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": merged_params,
        }
        self._socket.sendall(encode_frame(request))
        while True:
            response = read_frame(self._socket)
            if response.get("id") == request_id:
                if "error" in response:
                    message = response["error"].get("message", "Open Browser Use request failed")
                    raise RuntimeError(message)
                return response.get("result")
            if "id" not in response and isinstance(response.get("method"), str):
                self._dispatch_notification(response)
                continue
            raise RuntimeError(f"unexpected response id: {response.get('id')!r}")

    def _dispatch_notification(self, notification: JsonObject) -> None:
        for handler in list(self._notification_handlers):
            handler(notification)

    def get_info(self) -> Any:
        return self.request("getInfo")

    def create_tab(self) -> Any:
        return self.request("createTab")

    def get_tabs(self) -> Any:
        return self.request("getTabs")

    def get_user_tabs(self) -> Any:
        return self.request("getUserTabs")

    def get_user_history(self, **params: Any) -> Any:
        return self.request("getUserHistory", params)

    def claim_user_tab(self, tab_id: int) -> Any:
        return self.request("claimUserTab", {"tabId": tab_id})

    def finalize_tabs(self, keep: list[JsonObject]) -> Any:
        return self.request("finalizeTabs", {"keep": keep})

    def name_session(self, name: str) -> Any:
        return self.request("nameSession", {"name": name})

    def attach(self, tab_id: int) -> Any:
        return self.request("attach", {"tabId": tab_id})

    def detach(self, tab_id: int) -> Any:
        return self.request("detach", {"tabId": tab_id})

    def execute_cdp(self, tab_id: int, method: str, command_params: JsonObject | None = None) -> Any:
        return self.request(
            "executeCdp",
            {
                "target": {"tabId": tab_id},
                "method": method,
                "commandParams": command_params or {},
            },
        )

    def move_mouse(self, tab_id: int, x: float, y: float, wait_for_arrival: bool = True) -> Any:
        return self.request(
            "moveMouse",
            {
                "tabId": tab_id,
                "x": x,
                "y": y,
                "waitForArrival": wait_for_arrival,
            },
        )

    def wait_for_file_chooser(self, tab_id: int, timeout_ms: int | None = None) -> Any:
        params: JsonObject = {"tabId": tab_id}
        if timeout_ms is not None:
            params["timeoutMs"] = timeout_ms
        return self.request("waitForFileChooser", params)

    def set_file_chooser_files(self, file_chooser_id: str, files: list[str]) -> Any:
        return self.request(
            "setFileChooserFiles",
            {
                "fileChooserId": file_chooser_id,
                "files": files,
            },
        )

    def wait_for_download(self, tab_id: int, timeout_ms: int | None = None) -> Any:
        params: JsonObject = {"tabId": tab_id}
        if timeout_ms is not None:
            params["timeoutMs"] = timeout_ms
        return self.request("waitForDownload", params)

    def download_path(self, download_id: str, timeout_ms: int | None = None) -> Any:
        params: JsonObject = {"downloadId": download_id}
        if timeout_ms is not None:
            params["timeoutMs"] = timeout_ms
        return self.request("downloadPath", params)

    def browser_user_history(self, **params: Any) -> Any:
        return self.get_user_history(**params)

    def read_clipboard_text(self, tab_id: int) -> Any:
        return self.request("readClipboardText", {"tabId": tab_id})

    def write_clipboard_text(self, tab_id: int, text: str) -> Any:
        return self.request("writeClipboardText", {"tabId": tab_id, "text": text})

    def read_clipboard(self, tab_id: int) -> Any:
        return self.request("readClipboard", {"tabId": tab_id})

    def write_clipboard(self, tab_id: int, items: list[JsonObject]) -> Any:
        return self.request("writeClipboard", {"tabId": tab_id, "items": items})

    def turn_ended(self) -> Any:
        return self.request("turnEnded")


def connect_open_browser_use(
    socket_path: str = "",
    session_id: str = "open-browser-use-python",
    turn_id: str | None = None,
    timeout: float = 10.0,
) -> "OpenBrowserUseBrowser":
    """Connect to the OBU relay. On Windows, socket_path is ignored (auto TCP)."""
    browser = OpenBrowserUseBrowser(
        OpenBrowserUseClient(
            socket_path=socket_path,
            session_id=session_id,
            turn_id=turn_id,
            timeout=timeout,
        )
    )
    browser.connect()
    return browser


def connect(
    session_id: str = "open-browser-use-python",
    timeout: float = 10.0,
) -> "OpenBrowserUseBrowser":
    """Shorthand: connect to the OBU relay with auto-discovery."""
    return connect_open_browser_use("", session_id=session_id, timeout=timeout)


class OpenBrowserUseBrowser:
    def __init__(self, client: OpenBrowserUseClient) -> None:
        self.client = client
        self.cdp = OpenBrowserUseCdp(client)

    def connect(self) -> "OpenBrowserUseBrowser":
        self.client.connect()
        return self

    def close(self) -> None:
        self.client.close()

    def __enter__(self) -> "OpenBrowserUseBrowser":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()

    def new_tab(
        self,
        url: str | None = None,
        wait_until: LoadState = "load",
        timeout: float = DEFAULT_NAVIGATION_TIMEOUT,
    ) -> "OpenBrowserUseTab":
        result = self.client.create_tab()
        tab = self.tab(_tab_id_from_value(result, "create_tab response"))
        if url:
            tab.goto(url, wait_until=wait_until, timeout=timeout)
        return tab

    def tab(self, tab_id: int) -> "OpenBrowserUseTab":
        return OpenBrowserUseTab(self, tab_id)

    def get_tabs(self) -> Any:
        return self.client.get_tabs()


class OpenBrowserUseTab:
    def __init__(self, browser: OpenBrowserUseBrowser, tab_id: int) -> None:
        self.browser = browser
        self.id = tab_id
        self.playwright = OpenBrowserUseTabPlaywright(self)

    def goto(
        self,
        url: str,
        wait_until: LoadState = "load",
        timeout: float = DEFAULT_NAVIGATION_TIMEOUT,
    ) -> Any:
        return self.browser.cdp.navigate(self.id, url, wait_until=wait_until, timeout=timeout)

    def wait_for_load_state(
        self,
        state: LoadState = "load",
        timeout: float = DEFAULT_NAVIGATION_TIMEOUT,
    ) -> None:
        self.browser.cdp.wait_for_load_state(self.id, state=state, timeout=timeout)

    def dom_snapshot(self) -> str:
        value = self.browser.cdp.evaluate(self.id, "document.body?.innerText ?? ''")
        return "" if value is None else str(value)

    def page_info(self, selector: str = "", max_chars: int = 0) -> JsonObject:
        value = self.browser.cdp.evaluate(self.id, _page_info_expression(selector, max_chars))
        if not isinstance(value, dict):
            raise RuntimeError("page_info returned no object value")
        return {
            "title": str(value.get("title") or ""),
            "url": str(value.get("url") or ""),
            "readyState": str(value.get("readyState") or ""),
            "text": str(value.get("text") or ""),
        }

    def text_result(self, selector: str = "", max_chars: int = 0) -> JsonObject:
        return {"text": self.text(selector=selector, max_chars=max_chars)}

    def snapshot(self, limit: int = 100) -> JsonObject:
        value = self.browser.cdp.evaluate(self.id, _snapshot_expression(limit))
        if not isinstance(value, list):
            raise RuntimeError("snapshot returned no element list")
        return {"items": [_snapshot_item_from_value(item) for item in value]}

    def screenshot_result(self, selector: str = "", full_page: bool = False) -> JsonObject:
        command_params: JsonObject = {"format": "png"}
        clip: Any = None
        if selector:
            clip = self._resolve_screenshot_clip(_selector_clip_expression(selector), "selector")
            command_params["clip"] = clip
        elif full_page:
            clip = self._resolve_screenshot_clip(_full_page_clip_expression(), "full-page")
            command_params["clip"] = clip
        result = self.browser.cdp.call(self.id, "Page.captureScreenshot", command_params)
        if not isinstance(result, dict) or not isinstance(result.get("data"), str) or not result["data"]:
            raise RuntimeError("screenshot returned no data")
        data = result["data"]
        output: JsonObject = {
            "data": data,
            "bytes": len(base64.b64decode(data)),
            "format": "png",
            "tabId": self.id,
        }
        if selector:
            output["selector"] = selector
        if clip is not None:
            output["clip"] = clip
        return output

    def click(self, ref: str | int) -> JsonObject:
        result = self.browser.cdp.evaluate(self.id, _click_expression(str(ref).removeprefix("@")))
        return _interaction_result_from_value("click", result)

    def fill(self, ref: str | int, text: str) -> JsonObject:
        result = self.browser.cdp.evaluate(self.id, _fill_expression(str(ref).removeprefix("@"), text))
        return _interaction_result_from_value("fill", result)

    def evaluate(self, expression: str, await_promise: bool | None = None) -> Any:
        return self.browser.cdp.evaluate(self.id, expression, await_promise=await_promise)

    def title(self) -> str:
        value = self.evaluate("document.title ?? ''")
        return "" if value is None else str(value)

    def url(self) -> str:
        value = self.evaluate("location.href")
        return "" if value is None else str(value)

    def wait_for_timeout(self, timeout_ms: float) -> None:
        if timeout_ms < 0:
            raise ValueError("timeout_ms must be non-negative")
        time.sleep(timeout_ms / 1000)

    def screenshot(self, path: str | None = None) -> Any:
        """Take a page screenshot. If path is provided, saves to file."""
        result = self.browser.cdp.call(self.id, "Page.captureScreenshot")
        if isinstance(result, dict) and "data" in result:
            if path:
                with open(path, "wb") as f:
                    f.write(base64.b64decode(result["data"]))
            return result["data"]
        return result

    def text(self, selector: str = "body", max_chars: int = 0) -> str:
        """Get innerText of an element. Defaults to full page body text."""
        normalized_selector = "" if selector == "body" else selector
        value = self.browser.cdp.evaluate(self.id, _page_text_expression(normalized_selector, max_chars))
        return "" if value is None else str(value)

    def locator(self, selector: str) -> "OpenBrowserUseLocator":
        return OpenBrowserUseLocator(self, selector)

    def close(self) -> Any:
        return self.browser.cdp.call(self.id, "Page.close")

    def _resolve_screenshot_clip(self, expression: str, label: str) -> Any:
        value = self.browser.cdp.evaluate(self.id, expression)
        if not isinstance(value, dict) or value.get("ok") is not True:
            reason = value.get("reason") if isinstance(value, dict) else "unknown"
            raise RuntimeError(f"{label} screenshot clip failed: {reason or 'unknown'}")
        if not isinstance(value.get("clip"), dict):
            raise RuntimeError(f"{label} screenshot clip returned no clip")
        return value["clip"]


class OpenBrowserUseTabPlaywright:
    def __init__(self, tab: OpenBrowserUseTab) -> None:
        self.tab = tab

    def wait_for_load_state(
        self,
        state: LoadState = "load",
        timeout: float = DEFAULT_NAVIGATION_TIMEOUT,
    ) -> None:
        self.tab.wait_for_load_state(state=state, timeout=timeout)

    def dom_snapshot(self) -> str:
        return self.tab.dom_snapshot()

    def page_info(self, selector: str = "", max_chars: int = 0) -> JsonObject:
        return self.tab.page_info(selector=selector, max_chars=max_chars)

    def text_result(self, selector: str = "", max_chars: int = 0) -> JsonObject:
        return self.tab.text_result(selector=selector, max_chars=max_chars)

    def snapshot(self, limit: int = 100) -> JsonObject:
        return self.tab.snapshot(limit=limit)

    def screenshot_result(self, selector: str = "", full_page: bool = False) -> JsonObject:
        return self.tab.screenshot_result(selector=selector, full_page=full_page)

    def click(self, ref: str | int) -> JsonObject:
        return self.tab.click(ref)

    def fill(self, ref: str | int, text: str) -> JsonObject:
        return self.tab.fill(ref, text)

    def title(self) -> str:
        return self.tab.title()

    def url(self) -> str:
        return self.tab.url()

    def wait_for_timeout(self, timeout_ms: float) -> None:
        self.tab.wait_for_timeout(timeout_ms)

    def locator(self, selector: str) -> "OpenBrowserUseLocator":
        return self.tab.locator(selector)


class OpenBrowserUseLocator:
    def __init__(self, tab: OpenBrowserUseTab, selector: str) -> None:
        if not selector:
            raise ValueError("locator requires a selector")
        self.tab = tab
        self.selector = selector

    def inner_text(self, timeout_ms: int | None = None) -> str:
        value = self.tab.evaluate(
            _locator_inner_text_expression(self.selector, timeout_ms),
            await_promise=True,
        )
        return "" if value is None else str(value)


class OpenBrowserUseCdp:
    def __init__(self, client: OpenBrowserUseClient) -> None:
        self.client = client
        self._attached_tab_ids: set[int] = set()

    def call(
        self,
        tab_id: int,
        method: str,
        command_params: JsonObject | None = None,
        timeout_ms: int | None = None,
    ) -> Any:
        self.ensure_attached(tab_id)
        params: JsonObject = {
            "target": {"tabId": tab_id},
            "method": method,
            "commandParams": command_params or {},
        }
        if timeout_ms is not None:
            params["timeoutMs"] = timeout_ms
        return self.client.request("executeCdp", params)

    def evaluate(self, tab_id: int, expression: str, await_promise: bool | None = None) -> Any:
        command_params: JsonObject = {
            "expression": expression,
            "returnByValue": True,
        }
        if await_promise is not None:
            command_params["awaitPromise"] = await_promise
        result = self.call(tab_id, "Runtime.evaluate", command_params)
        if isinstance(result, dict) and isinstance(result.get("exceptionDetails"), dict):
            raise RuntimeError(str(result["exceptionDetails"].get("text", "Open Browser Use evaluation failed")))
        if isinstance(result, dict) and isinstance(result.get("result"), dict):
            return result["result"].get("value")
        return None

    def navigate(
        self,
        tab_id: int,
        url: str,
        wait_until: LoadState = "load",
        timeout: float = DEFAULT_NAVIGATION_TIMEOUT,
    ) -> Any:
        if not url:
            raise ValueError("goto requires a URL")
        _assert_supported_load_state(wait_until)
        self.call(tab_id, "Page.enable")
        result = self.call(tab_id, "Page.navigate", {"url": url}, timeout_ms=int(timeout * 1000))
        if isinstance(result, dict) and result.get("errorText"):
            raise RuntimeError(f"Browser failed to navigate tab {tab_id}: {result['errorText']}")
        self.wait_for_load_state(tab_id, state=wait_until, timeout=timeout)
        return result

    def wait_for_load_state(
        self,
        tab_id: int,
        state: LoadState = "load",
        timeout: float = DEFAULT_NAVIGATION_TIMEOUT,
    ) -> None:
        _assert_supported_load_state(state)
        self.call(tab_id, "Page.enable")
        deadline = time.monotonic() + timeout
        while True:
            if _document_state_matches(self.read_document_state(tab_id), state):
                return
            if time.monotonic() >= deadline:
                raise TimeoutError(f"Timed out waiting for {state} in tab {tab_id}")
            time.sleep(0.1)

    def read_document_state(self, tab_id: int) -> JsonObject | None:
        try:
            value = self.evaluate(tab_id, "({ href: window.location.href, readyState: document.readyState })")
        except Exception:
            return None
        return value if isinstance(value, dict) else None

    def ensure_attached(self, tab_id: int) -> None:
        if tab_id in self._attached_tab_ids:
            return
        self.client.attach(tab_id)
        self._attached_tab_ids.add(tab_id)


def encode_frame(value: JsonObject) -> bytes:
    payload = json.dumps(value, separators=(",", ":")).encode("utf-8")
    return struct.pack(_native_u32(), len(payload)) + payload


def read_frame(sock: socket.socket) -> JsonObject:
    header = _read_exact(sock, 4)
    (length,) = struct.unpack(_native_u32(), header)
    payload = _read_exact(sock, length)
    value = json.loads(payload.decode("utf-8"))
    if not isinstance(value, dict):
        raise RuntimeError("Open Browser Use response must be a JSON object")
    return value


def _read_exact(sock: socket.socket, length: int) -> bytes:
    chunks: list[bytes] = []
    remaining = length
    while remaining > 0:
        chunk = sock.recv(remaining)
        if not chunk:
            raise EOFError("Open Browser Use socket closed")
        chunks.append(chunk)
        remaining -= len(chunk)
    return b"".join(chunks)


def _native_u32() -> str:
    return "=I"


def _tab_id_from_value(value: Any, label: str) -> int:
    if not isinstance(value, dict):
        raise RuntimeError(f"{label} did not include a tab object")
    tab_id = value.get("id")
    if isinstance(tab_id, int) and tab_id > 0:
        return tab_id
    if isinstance(tab_id, str) and tab_id.isdigit() and int(tab_id) > 0:
        return int(tab_id)
    raise RuntimeError(f"{label} did not include a numeric tab id")


def _assert_supported_load_state(state: str) -> None:
    if state not in ("domcontentloaded", "load"):
        raise ValueError(f'Unsupported load state "{state}". Use "domcontentloaded" or "load".')


def _document_state_matches(document_state: JsonObject | None, state: LoadState) -> bool:
    ready_state = document_state.get("readyState") if isinstance(document_state, dict) else None
    return ready_state == "complete" or (state == "domcontentloaded" and ready_state == "interactive")


def _page_text_expression(selector: str, max_chars: int) -> str:
    selector_json = json.dumps(selector)
    return f"""(() => {{
  const selector = {selector_json};
  const root = selector ? document.querySelector(selector) : document.body;
  let text = root?.innerText ?? "";
  if ({max_chars} > 0 && text.length > {max_chars}) text = text.slice(0, {max_chars});
  return text;
}})()"""


def _page_info_expression(selector: str, max_chars: int) -> str:
    selector_json = json.dumps(selector)
    return f"""(() => {{
  const selector = {selector_json};
  const root = selector ? document.querySelector(selector) : document.body;
  let text = root?.innerText ?? "";
  if ({max_chars} > 0 && text.length > {max_chars}) text = text.slice(0, {max_chars});
  return {{ title: document.title ?? "", url: location.href, readyState: document.readyState, text }};
}})()"""


def _snapshot_expression(limit: int) -> str:
    normalized_limit = limit if limit > 0 else 100
    return f"""(() => {{
  if (typeof window.__obu_refs !== 'undefined' && window.__obu_refs_limit === {normalized_limit}) return window.__obu_refs;
  const els = document.querySelectorAll('button, a, input, select, textarea, [role=button], [role=textbox], [role=combobox]');
  const results = [];
  let idx = 0;
  for (const el of els) {{
    if (el.offsetParent === null && el.tagName !== 'SELECT') continue;
    const tag = el.tagName.toLowerCase();
    const text = (el.textContent || el.value || el.placeholder || el.getAttribute('aria-label') || '').trim().slice(0, 60);
    const id = el.id || '';
    const type = (el.type || '').toLowerCase();
    const href = el.href || '';
    const selector = id ? '#' + CSS.escape(id) : '';
    el.setAttribute('data-obu-ref', String(idx + 1));
    results.push({{ index: idx + 1, tag, text, type, href, selector }});
    idx++;
    if (idx >= {normalized_limit}) break;
  }}
  window.__obu_refs = results;
  window.__obu_refs_limit = {normalized_limit};
  return results;
}})()"""


def _selector_clip_expression(selector: str) -> str:
    selector_json = json.dumps(selector)
    return f"""(() => {{
  const selector = {selector_json};
  const el = document.querySelector(selector);
  if (!el) return {{ ok: false, reason: "not-found", selector }};
  el.scrollIntoView({{ block: "center", inline: "center", behavior: "instant" }});
  const rect = el.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) {{
    return {{ ok: false, reason: "not-visible", selector, rect: {{ x: rect.x, y: rect.y, width: rect.width, height: rect.height }} }};
  }}
  return {{
    ok: true,
    selector,
    clip: {{
      x: Math.max(0, rect.left + window.scrollX),
      y: Math.max(0, rect.top + window.scrollY),
      width: rect.width,
      height: rect.height,
      scale: 1
    }}
  }};
}})()"""


def _full_page_clip_expression() -> str:
    return """(() => {
  const doc = document.documentElement;
  const body = document.body;
  const width = Math.max(doc?.scrollWidth ?? 0, body?.scrollWidth ?? 0, doc?.clientWidth ?? 0, window.innerWidth);
  const height = Math.max(doc?.scrollHeight ?? 0, body?.scrollHeight ?? 0, doc?.clientHeight ?? 0, window.innerHeight);
  return { ok: true, clip: { x: 0, y: 0, width, height, scale: 1 } };
})()"""


def _interaction_prelude_expression(ref: str) -> str:
    ref_json = json.dumps(ref)
    return f"""const ref = {ref_json};
  const el = [...document.querySelectorAll("[data-obu-ref]")].find((candidate) => candidate.getAttribute("data-obu-ref") === ref);
  const describe = (reason, extra = {{}}) => ({{
    ok: false,
    ref,
    reason,
    tag: el?.tagName?.toLowerCase?.() ?? "",
    text: (el?.innerText || el?.value || el?.placeholder || el?.getAttribute?.("aria-label") || "").trim().slice(0, 120),
    ...extra,
  }});
  if (!el) return {{ ok: false, ref, reason: "not-found" }};
  if (el.disabled || el.getAttribute("aria-disabled") === "true") return describe("disabled");
  el.scrollIntoView({{ block: "center", inline: "center", behavior: "instant" }});
  const rect = el.getBoundingClientRect();
  const style = getComputedStyle(el);
  if (rect.width <= 0 || rect.height <= 0 || style.visibility === "hidden" || style.display === "none") {{
    return describe("not-visible", {{ rect: {{ x: rect.x, y: rect.y, width: rect.width, height: rect.height }} }});
  }}
  const x = rect.left + rect.width / 2;
  const y = rect.top + rect.height / 2;"""


def _click_expression(ref: str) -> str:
    return f"""(() => {{
  {_interaction_prelude_expression(ref)}
  const eventInit = {{ bubbles: true, cancelable: true, view: window, clientX: x, clientY: y }};
  for (const type of ["pointerover", "mouseover", "pointermove", "mousemove", "pointerdown", "mousedown", "pointerup", "mouseup"]) {{
    const EventClass = type.startsWith("pointer") && window.PointerEvent ? PointerEvent : MouseEvent;
    el.dispatchEvent(new EventClass(type, eventInit));
  }}
  if (typeof el.click === "function") el.click();
  return {{
    ok: true,
    ref,
    action: "click",
    tag: el.tagName.toLowerCase(),
    text: (el.innerText || el.value || el.placeholder || el.getAttribute("aria-label") || "").trim().slice(0, 120),
    rect: {{ x: rect.x, y: rect.y, width: rect.width, height: rect.height }}
  }};
}})()"""


def _fill_expression(ref: str, text: str) -> str:
    text_json = json.dumps(text)
    return f"""(() => {{
  {_interaction_prelude_expression(ref)}
  const text = {text_json};
  el.focus();
  if (el.isContentEditable) {{
    el.textContent = text;
  }} else if ("value" in el) {{
    const proto = el instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype :
      el instanceof HTMLSelectElement ? HTMLSelectElement.prototype : HTMLInputElement.prototype;
    const descriptor = Object.getOwnPropertyDescriptor(proto, "value");
    if (descriptor?.set) {{
      descriptor.set.call(el, text);
    }} else {{
      el.value = text;
    }}
  }} else {{
    return describe("not-fillable");
  }}
  el.dispatchEvent(new InputEvent("beforeinput", {{ bubbles: true, cancelable: true, inputType: "insertText", data: text }}));
  el.dispatchEvent(new InputEvent("input", {{ bubbles: true, inputType: "insertText", data: text }}));
  el.dispatchEvent(new Event("change", {{ bubbles: true }}));
  return {{
    ok: true,
    ref,
    action: "fill",
    tag: el.tagName.toLowerCase(),
    text: (el.innerText || el.value || el.placeholder || el.getAttribute("aria-label") || "").trim().slice(0, 120),
    valueLength: String(("value" in el ? el.value : el.textContent) ?? "").length,
    rect: {{ x: rect.x, y: rect.y, width: rect.width, height: rect.height }}
  }};
}})()"""


def _snapshot_item_from_value(value: Any) -> JsonObject:
    if not isinstance(value, dict):
        raise RuntimeError("snapshot returned a non-object element")
    return {
        "index": int(value.get("index") or 0),
        "tag": str(value.get("tag") or ""),
        "text": str(value.get("text") or ""),
        "type": str(value.get("type") or ""),
        "href": str(value.get("href") or ""),
        "selector": str(value.get("selector") or ""),
    }


def _interaction_result_from_value(action: str, value: Any) -> JsonObject:
    if not isinstance(value, dict):
        raise RuntimeError(f"{action} returned no object value")
    result = dict(value)
    if result.get("ok") is not True:
        raise RuntimeError(f"{action} failed: {result.get('reason') or 'unknown'}")
    return result


def _locator_inner_text_expression(selector: str, timeout_ms: int | None) -> str:
    timeout = DEFAULT_NAVIGATION_TIMEOUT * 1000 if timeout_ms is None else timeout_ms
    if timeout < 0:
        raise ValueError("timeout_ms must be non-negative")
    selector_json = json.dumps(selector)
    return f"""(async () => {{
  const selector = {selector_json};
  const deadline = performance.now() + {timeout};
  while (true) {{
    const element = document.querySelector(selector);
    if (element) {{
      return element.innerText ?? element.textContent ?? "";
    }}
    if (performance.now() >= deadline) {{
      throw new Error(`Timed out waiting for locator ${{selector}}`);
    }}
    await new Promise((resolve) => setTimeout(resolve, 100));
  }}
}})()"""
