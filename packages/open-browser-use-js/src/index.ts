import { Socket, createConnection } from "node:net";
import { endianness } from "node:os";

export type JsonValue =
  | null
  | boolean
  | number
  | string
  | JsonValue[]
  | { [key: string]: JsonValue };

export type BrowserUseRequestParams = { [key: string]: JsonValue };

export type OpenBrowserUseClientOptions = {
  socketPath?: string;
  sessionId?: string;
  turnId?: string;
  timeoutMs?: number;
};

type PendingRequest = {
  resolve: (value: JsonValue) => void;
  reject: (error: Error) => void;
  timeout: NodeJS.Timeout;
};

export type OpenBrowserUseNotification = {
  method: string;
  params?: JsonValue;
};

export type NotificationHandler = (notification: OpenBrowserUseNotification) => void;

const headerBytes = 4;
const defaultRelayPath = "127.0.0.1:19832";
const defaultNavigationTimeoutMs = 10_000;

export type OpenBrowserUseLoadState = "domcontentloaded" | "load";

export type OpenBrowserUseGotoOptions = {
  waitUntil?: OpenBrowserUseLoadState;
  timeoutMs?: number;
};

export type OpenBrowserUseWaitForLoadStateOptions = {
  state?: OpenBrowserUseLoadState;
  timeoutMs?: number;
};

export type OpenBrowserUseBrowserOptions = OpenBrowserUseClientOptions | { client: OpenBrowserUseClient };

export type OpenBrowserUseTextOptions = {
  selector?: string;
  maxChars?: number;
};

export type OpenBrowserUsePageInfo = {
  title: string;
  url: string;
  readyState: string;
  text: string;
};

export type OpenBrowserUseTextResult = {
  text: string;
};

export type OpenBrowserUseSnapshotItem = {
  index: number;
  tag: string;
  text: string;
  type?: string;
  href?: string;
  selector?: string;
};

export type OpenBrowserUseSnapshotResult = {
  items: OpenBrowserUseSnapshotItem[];
};

export type OpenBrowserUseScreenshotOptions = {
  selector?: string;
  fullPage?: boolean;
};

export type OpenBrowserUseScreenshotResult = {
  data: string;
  bytes: number;
  format: "png";
  tabId: number;
  selector?: string;
  clip?: JsonValue;
};

export type OpenBrowserUseInteractionResult = {
  ok: boolean;
  ref: string;
  action?: "click" | "fill";
  reason?: string;
  tag?: string;
  text?: string;
  rect?: JsonValue;
  valueLength?: number;
};

export class OpenBrowserUseClient {
  readonly socketPath: string;
  readonly sessionId: string;
  readonly turnId: string;
  readonly timeoutMs: number;
  #socket: Socket | null = null;
  #pendingData = Buffer.alloc(0);
  #nextId = 1;
  #pending = new Map<string, PendingRequest>();
  #notificationHandlers = new Set<NotificationHandler>();

  constructor(options: OpenBrowserUseClientOptions = {}) {
    this.socketPath = options.socketPath ?? defaultRelayPath;
    this.sessionId = options.sessionId ?? "open-browser-use-js";
    this.turnId = options.turnId ?? `turn-${Date.now()}`;
    this.timeoutMs = options.timeoutMs ?? 10_000;
  }

  async connect(): Promise<this> {
    if (this.#socket) {
      return this;
    }
    const tcpTarget = tcpSocketTarget(this.socketPath);
    const socket = tcpTarget ? createConnection(tcpTarget.port, tcpTarget.host) : createConnection(this.socketPath);
    this.#socket = socket;
    socket.on("data", (chunk) => this.#handleData(Buffer.from(chunk)));
    socket.on("close", () => this.#rejectAll(new Error("Open Browser Use socket closed")));
    socket.on("error", (error) => this.#rejectAll(error));
    await new Promise<void>((resolve, reject) => {
      socket.once("connect", resolve);
      socket.once("error", reject);
    });
    return this;
  }

  close(): void {
    this.#socket?.end();
    this.#socket = null;
  }

  onNotification(handler: NotificationHandler): () => void {
    this.#notificationHandlers.add(handler);
    return () => {
      this.#notificationHandlers.delete(handler);
    };
  }

  async request(method: string, params: BrowserUseRequestParams = {}): Promise<JsonValue> {
    await this.connect();
    const socket = this.#socket;
    if (!socket) {
      throw new Error("Open Browser Use socket is not connected");
    }
    const id = String(this.#nextId++);
    const mergedParams = {
      session_id: this.sessionId,
      turn_id: this.turnId,
      ...params
    };
    const message = {
      jsonrpc: "2.0",
      id,
      method,
      params: mergedParams
    };
    const payload = encodeFrame(message);
    const promise = new Promise<JsonValue>((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.#pending.delete(id);
        reject(new Error(`Open Browser Use request timed out: ${method}`));
      }, this.timeoutMs);
      this.#pending.set(id, { resolve, reject, timeout });
    });
    socket.write(payload);
    return await promise;
  }

  getInfo(): Promise<JsonValue> {
    return this.request("getInfo");
  }

  createTab(): Promise<JsonValue> {
    return this.request("createTab");
  }

  getTabs(): Promise<JsonValue> {
    return this.request("getTabs");
  }

  getUserTabs(): Promise<JsonValue> {
    return this.request("getUserTabs");
  }

  getUserHistory(params: BrowserUseRequestParams = {}): Promise<JsonValue> {
    return this.request("getUserHistory", params);
  }

  claimUserTab(tabId: number): Promise<JsonValue> {
    return this.request("claimUserTab", { tabId });
  }

  finalizeTabs(keep: JsonValue[]): Promise<JsonValue> {
    return this.request("finalizeTabs", { keep });
  }

  nameSession(name: string): Promise<JsonValue> {
    return this.request("nameSession", { name });
  }

  attach(tabId: number): Promise<JsonValue> {
    return this.request("attach", { tabId });
  }

  detach(tabId: number): Promise<JsonValue> {
    return this.request("detach", { tabId });
  }

  executeCdp(tabId: number, method: string, commandParams: BrowserUseRequestParams = {}): Promise<JsonValue> {
    return this.request("executeCdp", {
      target: { tabId },
      method,
      commandParams
    });
  }

  moveMouse(tabId: number, x: number, y: number, waitForArrival = true): Promise<JsonValue> {
    return this.request("moveMouse", { tabId, x, y, waitForArrival });
  }

  waitForFileChooser(tabId: number, timeoutMs?: number): Promise<JsonValue> {
    return this.request("waitForFileChooser", {
      tabId,
      ...(timeoutMs === undefined ? {} : { timeoutMs })
    });
  }

  setFileChooserFiles(fileChooserId: string, files: string[]): Promise<JsonValue> {
    return this.request("setFileChooserFiles", { fileChooserId, files });
  }

  waitForDownload(tabId: number, timeoutMs?: number): Promise<JsonValue> {
    return this.request("waitForDownload", {
      tabId,
      ...(timeoutMs === undefined ? {} : { timeoutMs })
    });
  }

  downloadPath(downloadId: string, timeoutMs?: number): Promise<JsonValue> {
    return this.request("downloadPath", {
      downloadId,
      ...(timeoutMs === undefined ? {} : { timeoutMs })
    });
  }

  browserUserHistory(params: BrowserUseRequestParams = {}): Promise<JsonValue> {
    return this.getUserHistory(params);
  }

  readClipboardText(tabId: number): Promise<JsonValue> {
    return this.request("readClipboardText", { tabId });
  }

  writeClipboardText(tabId: number, text: string): Promise<JsonValue> {
    return this.request("writeClipboardText", { tabId, text });
  }

  readClipboard(tabId: number): Promise<JsonValue> {
    return this.request("readClipboard", { tabId });
  }

  writeClipboard(tabId: number, items: JsonValue[]): Promise<JsonValue> {
    return this.request("writeClipboard", { tabId, items });
  }

  turnEnded(): Promise<JsonValue> {
    return this.request("turnEnded");
  }

  #handleData(chunk: Buffer): void {
    this.#pendingData = Buffer.concat([this.#pendingData, chunk]);
    while (this.#pendingData.length >= headerBytes) {
      const length =
        endianness() === "LE"
          ? this.#pendingData.readUInt32LE(0)
          : this.#pendingData.readUInt32BE(0);
      const total = headerBytes + length;
      if (this.#pendingData.length < total) {
        return;
      }
      const payload = this.#pendingData.subarray(headerBytes, total);
      this.#pendingData = this.#pendingData.subarray(total);
      this.#handleMessage(JSON.parse(payload.toString("utf8")) as JsonValue);
    }
  }

  #handleMessage(message: JsonValue): void {
    if (!isObject(message)) {
      return;
    }
    const id = typeof message.id === "string" || typeof message.id === "number" ? String(message.id) : null;
    if (!id && typeof message.method === "string") {
      const notification = {
        method: message.method,
        params: message.params
      };
      for (const handler of this.#notificationHandlers) {
        handler(notification);
      }
      return;
    }
    if (!id) {
      return;
    }
    const pending = this.#pending.get(id);
    if (!pending) {
      return;
    }
    this.#pending.delete(id);
    clearTimeout(pending.timeout);
    if (isObject(message.error)) {
      pending.reject(new Error(String(message.error.message ?? "Open Browser Use request failed")));
      return;
    }
    pending.resolve((message.result ?? null) as JsonValue);
  }

  #rejectAll(error: Error): void {
    for (const [id, pending] of this.#pending) {
      this.#pending.delete(id);
      clearTimeout(pending.timeout);
      pending.reject(error);
    }
  }
}

export async function connectOpenBrowserUse(options: OpenBrowserUseBrowserOptions = {}): Promise<OpenBrowserUseBrowser> {
  const browser = new OpenBrowserUseBrowser(options);
  await browser.connect();
  return browser;
}

export class OpenBrowserUseBrowser {
  readonly client: OpenBrowserUseClient;
  readonly cdp: OpenBrowserUseCdp;

  constructor(options: OpenBrowserUseBrowserOptions = {}) {
    this.client = "client" in options ? options.client : new OpenBrowserUseClient(options);
    this.cdp = new OpenBrowserUseCdp(this.client);
  }

  async connect(): Promise<this> {
    await this.client.connect();
    return this;
  }

  close(): void {
    this.client.close();
  }

  async newTab(options: OpenBrowserUseGotoOptions & { url?: string } = {}): Promise<OpenBrowserUseTab> {
    const created = await this.client.createTab();
    const tabId = tabIdFromValue(created, "createTab response");
    const tab = this.tab(tabId);
    if (options.url) {
      await tab.goto(options.url, options);
    }
    return tab;
  }

  tab(tabId: number): OpenBrowserUseTab {
    return new OpenBrowserUseTab(this, tabId);
  }

  getTabs(): Promise<JsonValue> {
    return this.client.getTabs();
  }
}

export class OpenBrowserUseTab {
  readonly browser: OpenBrowserUseBrowser;
  readonly id: number;
  readonly playwright: OpenBrowserUseTabPlaywright;

  constructor(browser: OpenBrowserUseBrowser, id: number) {
    this.browser = browser;
    this.id = id;
    this.playwright = new OpenBrowserUseTabPlaywright(this);
  }

  goto(url: string, options: OpenBrowserUseGotoOptions = {}): Promise<JsonValue> {
    return this.browser.cdp.navigate(this.id, url, options);
  }

  waitForLoadState(options: OpenBrowserUseWaitForLoadStateOptions = {}): Promise<void> {
    return this.browser.cdp.waitForLoadState(this.id, options);
  }

  domSnapshot(): Promise<string> {
    return this.browser.cdp.evaluate(this.id, "document.body?.innerText ?? ''").then((value) => String(value ?? ""));
  }

  async pageInfo(options: OpenBrowserUseTextOptions = {}): Promise<OpenBrowserUsePageInfo> {
    const value = await this.browser.cdp.evaluate(
      this.id,
      pageInfoExpression(options.selector ?? "", options.maxChars ?? 0)
    );
    if (!isObject(value)) {
      throw new Error("pageInfo returned no object value");
    }
    return {
      title: String(value.title ?? ""),
      url: String(value.url ?? ""),
      readyState: String(value.readyState ?? ""),
      text: String(value.text ?? "")
    };
  }

  async text(options: OpenBrowserUseTextOptions = {}): Promise<OpenBrowserUseTextResult> {
    const value = await this.browser.cdp.evaluate(
      this.id,
      pageTextExpression(options.selector ?? "", options.maxChars ?? 0)
    );
    return { text: String(value ?? "") };
  }

  async snapshot(options: { limit?: number } = {}): Promise<OpenBrowserUseSnapshotResult> {
    const value = await this.browser.cdp.evaluate(this.id, snapshotExpression(options.limit ?? 100));
    if (!Array.isArray(value)) {
      throw new Error("snapshot returned no element list");
    }
    return { items: value.map(snapshotItemFromValue) };
  }

  async screenshot(options: OpenBrowserUseScreenshotOptions = {}): Promise<OpenBrowserUseScreenshotResult> {
    const commandParams: BrowserUseRequestParams = { format: "png" };
    let clip: JsonValue | undefined;
    if (options.selector) {
      clip = await this.resolveScreenshotClip(selectorClipExpression(options.selector), "selector");
      commandParams.clip = clip;
    } else if (options.fullPage) {
      clip = await this.resolveScreenshotClip(fullPageClipExpression(), "full-page");
      commandParams.clip = clip;
    }
    const result = await this.browser.cdp.call(this.id, "Page.captureScreenshot", commandParams);
    if (!isObject(result) || typeof result.data !== "string" || !result.data) {
      throw new Error("screenshot returned no data");
    }
    return {
      data: result.data,
      bytes: Buffer.from(result.data, "base64").length,
      format: "png",
      tabId: this.id,
      ...(options.selector ? { selector: options.selector } : {}),
      ...(clip === undefined ? {} : { clip })
    };
  }

  async click(ref: string | number): Promise<OpenBrowserUseInteractionResult> {
    const result = await this.browser.cdp.evaluate(this.id, clickExpression(String(ref).replace(/^@/, "")));
    return interactionResultFromValue("click", result);
  }

  async fill(ref: string | number, text: string): Promise<OpenBrowserUseInteractionResult> {
    const result = await this.browser.cdp.evaluate(this.id, fillExpression(String(ref).replace(/^@/, ""), text));
    return interactionResultFromValue("fill", result);
  }

  evaluate(expression: string, options: { awaitPromise?: boolean } = {}): Promise<JsonValue> {
    return this.browser.cdp.evaluate(this.id, expression, options);
  }

  close(): Promise<JsonValue> {
    return this.browser.cdp.call(this.id, "Page.close");
  }

  private async resolveScreenshotClip(expression: string, label: string): Promise<JsonValue> {
    const value = await this.browser.cdp.evaluate(this.id, expression);
    if (!isObject(value) || value.ok !== true) {
      const reason = isObject(value) && typeof value.reason === "string" ? value.reason : "unknown";
      throw new Error(`${label} screenshot clip failed: ${reason}`);
    }
    if (!isObject(value.clip)) {
      throw new Error(`${label} screenshot clip returned no clip`);
    }
    return value.clip;
  }
}

export class OpenBrowserUseTabPlaywright {
  readonly tab: OpenBrowserUseTab;

  constructor(tab: OpenBrowserUseTab) {
    this.tab = tab;
  }

  waitForLoadState(options: OpenBrowserUseWaitForLoadStateOptions = {}): Promise<void> {
    return this.tab.waitForLoadState(options);
  }

  domSnapshot(): Promise<string> {
    return this.tab.domSnapshot();
  }

  pageInfo(options: OpenBrowserUseTextOptions = {}): Promise<OpenBrowserUsePageInfo> {
    return this.tab.pageInfo(options);
  }

  text(options: OpenBrowserUseTextOptions = {}): Promise<OpenBrowserUseTextResult> {
    return this.tab.text(options);
  }

  snapshot(options: { limit?: number } = {}): Promise<OpenBrowserUseSnapshotResult> {
    return this.tab.snapshot(options);
  }

  screenshot(options: OpenBrowserUseScreenshotOptions = {}): Promise<OpenBrowserUseScreenshotResult> {
    return this.tab.screenshot(options);
  }

  click(ref: string | number): Promise<OpenBrowserUseInteractionResult> {
    return this.tab.click(ref);
  }

  fill(ref: string | number, text: string): Promise<OpenBrowserUseInteractionResult> {
    return this.tab.fill(ref, text);
  }
}

export class OpenBrowserUseCdp {
  readonly client: OpenBrowserUseClient;
  #attachedTabIds = new Set<number>();

  constructor(client: OpenBrowserUseClient) {
    this.client = client;
  }

  async call(
    tabId: number,
    method: string,
    commandParams: BrowserUseRequestParams = {},
    options: { timeoutMs?: number } = {}
  ): Promise<JsonValue> {
    await this.ensureAttached(tabId);
    return this.client.request("executeCdp", {
      target: { tabId },
      method,
      commandParams,
      ...(options.timeoutMs === undefined ? {} : { timeoutMs: options.timeoutMs })
    });
  }

  async evaluate(tabId: number, expression: string, options: { awaitPromise?: boolean } = {}): Promise<JsonValue> {
    const result = await this.call(tabId, "Runtime.evaluate", {
      expression,
      returnByValue: true,
      ...(options.awaitPromise === undefined ? {} : { awaitPromise: options.awaitPromise })
    });
    if (isObject(result) && isObject(result.exceptionDetails)) {
      throw new Error(String(result.exceptionDetails.text ?? "Open Browser Use evaluation failed"));
    }
    return isObject(result) && isObject(result.result) ? (result.result.value ?? null) : null;
  }

  async navigate(tabId: number, url: string, options: OpenBrowserUseGotoOptions = {}): Promise<JsonValue> {
    if (!url) {
      throw new Error("goto requires a URL");
    }
    const waitUntil = options.waitUntil ?? "load";
    assertSupportedLoadState(waitUntil);
    const timeoutMs = options.timeoutMs ?? defaultNavigationTimeoutMs;
    await this.call(tabId, "Page.enable");
    const wait = this.waitForLoadEvent(tabId, waitUntil, timeoutMs);
    const result = await this.call(tabId, "Page.navigate", { url }, { timeoutMs });
    if (isObject(result) && typeof result.errorText === "string" && result.errorText) {
      throw new Error(`Browser failed to navigate tab ${tabId}: ${result.errorText}`);
    }
    await wait.catch(async (error) => {
      const state = await this.readDocumentState(tabId);
      if (documentStateMatches(state, waitUntil)) {
        return;
      }
      throw error;
    });
    return result;
  }

  async waitForLoadState(tabId: number, options: OpenBrowserUseWaitForLoadStateOptions = {}): Promise<void> {
    const state = options.state ?? "load";
    assertSupportedLoadState(state);
    await this.call(tabId, "Page.enable");
    const documentState = await this.readDocumentState(tabId);
    if (documentStateMatches(documentState, state)) {
      return;
    }
    await this.waitForLoadEvent(tabId, state, options.timeoutMs ?? defaultNavigationTimeoutMs);
  }

  async readDocumentState(tabId: number): Promise<{ href?: string; readyState?: string } | undefined> {
    try {
      const value = await this.evaluate(tabId, "({ href: window.location.href, readyState: document.readyState })");
      return isObject(value) ? { href: stringValue(value.href), readyState: stringValue(value.readyState) } : undefined;
    } catch {
      return undefined;
    }
  }

  waitForEvent(
    tabId: number,
    predicate: (notification: OpenBrowserUseNotification) => boolean,
    options: { timeoutMs?: number; timeoutMessage?: string } = {}
  ): Promise<OpenBrowserUseNotification> {
    const timeoutMs = options.timeoutMs ?? defaultNavigationTimeoutMs;
    return new Promise((resolve, reject) => {
      let settled = false;
      let removeHandler: (() => void) | null = null;
      let timer: NodeJS.Timeout;
      const finish = (): boolean => {
        if (settled) {
          return false;
        }
        settled = true;
        clearTimeout(timer);
        removeHandler?.();
        return true;
      };
      timer = setTimeout(() => {
        if (finish()) {
          reject(new Error(options.timeoutMessage ?? `Timed out waiting for tab ${tabId} event`));
        }
      }, timeoutMs);
      removeHandler = this.client.onNotification((notification) => {
        try {
          if (predicate(notification)) {
            if (finish()) {
              resolve(notification);
            }
          }
        } catch (error) {
          if (finish()) {
            reject(error instanceof Error ? error : new Error(String(error)));
          }
        }
      });
    });
  }

  waitForLoadEvent(tabId: number, state: OpenBrowserUseLoadState, timeoutMs: number): Promise<void> {
    return this.waitForEvent(
      tabId,
      (notification) => {
        const event = cdpEventForTab(notification, tabId);
        if (!event) {
          return false;
        }
        if (event.method === "Page.navigationBlocked") {
          throw new Error(`Navigation was blocked in tab ${tabId}`);
        }
        return state === "domcontentloaded"
          ? event.method === "Page.domContentEventFired"
          : event.method === "Page.loadEventFired";
      },
      {
        timeoutMs,
        timeoutMessage: `Timed out waiting for ${state} in tab ${tabId}`
      }
    ).then(() => undefined);
  }

  async ensureAttached(tabId: number): Promise<void> {
    if (this.#attachedTabIds.has(tabId)) {
      return;
    }
    await this.client.attach(tabId);
    this.#attachedTabIds.add(tabId);
  }
}

export function encodeFrame(value: JsonValue | Record<string, unknown>): Buffer {
  const payload = Buffer.from(JSON.stringify(value), "utf8");
  const frame = Buffer.alloc(headerBytes + payload.length);
  if (endianness() === "LE") {
    frame.writeUInt32LE(payload.length, 0);
  } else {
    frame.writeUInt32BE(payload.length, 0);
  }
  payload.copy(frame, headerBytes);
  return frame;
}

function isObject(value: JsonValue | unknown): value is { [key: string]: JsonValue } {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function tabIdFromValue(value: JsonValue, label: string): number {
  if (!isObject(value)) {
    throw new Error(`${label} did not include a tab object`);
  }
  const id = value.id;
  if (typeof id === "number" && Number.isInteger(id) && id > 0) {
    return id;
  }
  if (typeof id === "string") {
    const parsed = Number(id);
    if (Number.isInteger(parsed) && parsed > 0) {
      return parsed;
    }
  }
  throw new Error(`${label} did not include a numeric tab id`);
}

function assertSupportedLoadState(state: string): asserts state is OpenBrowserUseLoadState {
  if (state !== "domcontentloaded" && state !== "load") {
    throw new Error(`Unsupported load state "${state}". Use "domcontentloaded" or "load".`);
  }
}

function documentStateMatches(documentState: { readyState?: string } | undefined, state: OpenBrowserUseLoadState): boolean {
  if (documentState?.readyState === "complete") {
    return true;
  }
  return state === "domcontentloaded" && documentState?.readyState === "interactive";
}

function cdpEventForTab(
  notification: OpenBrowserUseNotification,
  tabId: number
): { method?: string } | null {
  if (notification.method !== "onCDPEvent" || !isObject(notification.params)) {
    return null;
  }
  const source = notification.params.source;
  if (!isObject(source) || source.tabId !== tabId) {
    return null;
  }
  return {
    method: stringValue(notification.params.method)
  };
}

function stringValue(value: JsonValue | undefined): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function tcpSocketTarget(socketPath: string): { host: string; port: number } | null {
  const match = /^([^:]+):(\d+)$/.exec(socketPath);
  if (!match) {
    return null;
  }
  return { host: match[1], port: Number(match[2]) };
}

function pageTextExpression(selector: string, maxChars: number): string {
  return `(() => {
  const selector = ${JSON.stringify(selector)};
  const root = selector ? document.querySelector(selector) : document.body;
  let text = root?.innerText ?? "";
  if (${maxChars} > 0 && text.length > ${maxChars}) text = text.slice(0, ${maxChars});
  return text;
})()`;
}

function pageInfoExpression(selector: string, maxChars: number): string {
  return `(() => {
  const selector = ${JSON.stringify(selector)};
  const root = selector ? document.querySelector(selector) : document.body;
  let text = root?.innerText ?? "";
  if (${maxChars} > 0 && text.length > ${maxChars}) text = text.slice(0, ${maxChars});
  return { title: document.title ?? "", url: location.href, readyState: document.readyState, text };
})()`;
}

function snapshotExpression(limit: number): string {
  const normalizedLimit = limit > 0 ? limit : 100;
  return `(() => {
  if (typeof window.__obu_refs !== 'undefined' && window.__obu_refs_limit === ${normalizedLimit}) return window.__obu_refs;
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
    if (idx >= ${normalizedLimit}) break;
  }
  window.__obu_refs = results;
  window.__obu_refs_limit = ${normalizedLimit};
  return results;
})()`;
}

function selectorClipExpression(selector: string): string {
  return `(() => {
  const selector = ${JSON.stringify(selector)};
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
})()`;
}

function fullPageClipExpression(): string {
  return `(() => {
  const doc = document.documentElement;
  const body = document.body;
  const width = Math.max(doc?.scrollWidth ?? 0, body?.scrollWidth ?? 0, doc?.clientWidth ?? 0, window.innerWidth);
  const height = Math.max(doc?.scrollHeight ?? 0, body?.scrollHeight ?? 0, doc?.clientHeight ?? 0, window.innerHeight);
  return { ok: true, clip: { x: 0, y: 0, width, height, scale: 1 } };
})()`;
}

function interactionPreludeExpression(ref: string): string {
  return `const ref = ${JSON.stringify(ref)};
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
  const y = rect.top + rect.height / 2;`;
}

function clickExpression(ref: string): string {
  return `(() => {
  ${interactionPreludeExpression(ref)}
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
})()`;
}

function fillExpression(ref: string, text: string): string {
  return `(() => {
  ${interactionPreludeExpression(ref)}
  const text = ${JSON.stringify(text)};
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
})()`;
}

function snapshotItemFromValue(value: JsonValue): OpenBrowserUseSnapshotItem {
  if (!isObject(value)) {
    throw new Error("snapshot returned a non-object element");
  }
  return {
    index: Number(value.index ?? 0),
    tag: String(value.tag ?? ""),
    text: String(value.text ?? ""),
    ...(typeof value.type === "string" ? { type: value.type } : {}),
    ...(typeof value.href === "string" ? { href: value.href } : {}),
    ...(typeof value.selector === "string" ? { selector: value.selector } : {})
  };
}

function interactionResultFromValue(action: "click" | "fill", value: JsonValue): OpenBrowserUseInteractionResult {
  if (!isObject(value)) {
    throw new Error(`${action} returned no object value`);
  }
  const result: OpenBrowserUseInteractionResult = {
    ok: value.ok === true,
    ref: String(value.ref ?? ""),
    ...(value.action === action ? { action } : {}),
    ...(typeof value.reason === "string" ? { reason: value.reason } : {}),
    ...(typeof value.tag === "string" ? { tag: value.tag } : {}),
    ...(typeof value.text === "string" ? { text: value.text } : {}),
    ...(value.rect === undefined ? {} : { rect: value.rect }),
    ...(typeof value.valueLength === "number" ? { valueLength: value.valueLength } : {})
  };
  if (!result.ok) {
    throw new Error(`${action} failed: ${result.reason ?? "unknown"}`);
  }
  return result;
}
