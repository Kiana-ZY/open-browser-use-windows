export type JsonValue = null | boolean | number | string | JsonValue[] | {
    [key: string]: JsonValue;
};
export type BrowserUseRequestParams = {
    [key: string]: JsonValue;
};
export type OpenBrowserUseClientOptions = {
    socketPath?: string;
    sessionId?: string;
    turnId?: string;
    timeoutMs?: number;
};
export type OpenBrowserUseNotification = {
    method: string;
    params?: JsonValue;
};
export type NotificationHandler = (notification: OpenBrowserUseNotification) => void;
export type OpenBrowserUseLoadState = "domcontentloaded" | "load";
export type OpenBrowserUseGotoOptions = {
    waitUntil?: OpenBrowserUseLoadState;
    timeoutMs?: number;
};
export type OpenBrowserUseWaitForLoadStateOptions = {
    state?: OpenBrowserUseLoadState;
    timeoutMs?: number;
};
export type OpenBrowserUseBrowserOptions = OpenBrowserUseClientOptions | {
    client: OpenBrowserUseClient;
};
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
export declare class OpenBrowserUseClient {
    #private;
    readonly socketPath: string;
    readonly sessionId: string;
    readonly turnId: string;
    readonly timeoutMs: number;
    constructor(options?: OpenBrowserUseClientOptions);
    connect(): Promise<this>;
    close(): void;
    onNotification(handler: NotificationHandler): () => void;
    request(method: string, params?: BrowserUseRequestParams): Promise<JsonValue>;
    getInfo(): Promise<JsonValue>;
    createTab(): Promise<JsonValue>;
    getTabs(): Promise<JsonValue>;
    getUserTabs(): Promise<JsonValue>;
    getUserHistory(params?: BrowserUseRequestParams): Promise<JsonValue>;
    claimUserTab(tabId: number): Promise<JsonValue>;
    finalizeTabs(keep: JsonValue[]): Promise<JsonValue>;
    nameSession(name: string): Promise<JsonValue>;
    attach(tabId: number): Promise<JsonValue>;
    detach(tabId: number): Promise<JsonValue>;
    executeCdp(tabId: number, method: string, commandParams?: BrowserUseRequestParams): Promise<JsonValue>;
    moveMouse(tabId: number, x: number, y: number, waitForArrival?: boolean): Promise<JsonValue>;
    waitForFileChooser(tabId: number, timeoutMs?: number): Promise<JsonValue>;
    setFileChooserFiles(fileChooserId: string, files: string[]): Promise<JsonValue>;
    waitForDownload(tabId: number, timeoutMs?: number): Promise<JsonValue>;
    downloadPath(downloadId: string, timeoutMs?: number): Promise<JsonValue>;
    browserUserHistory(params?: BrowserUseRequestParams): Promise<JsonValue>;
    readClipboardText(tabId: number): Promise<JsonValue>;
    writeClipboardText(tabId: number, text: string): Promise<JsonValue>;
    readClipboard(tabId: number): Promise<JsonValue>;
    writeClipboard(tabId: number, items: JsonValue[]): Promise<JsonValue>;
    turnEnded(): Promise<JsonValue>;
}
export declare function connectOpenBrowserUse(options?: OpenBrowserUseBrowserOptions): Promise<OpenBrowserUseBrowser>;
export declare class OpenBrowserUseBrowser {
    readonly client: OpenBrowserUseClient;
    readonly cdp: OpenBrowserUseCdp;
    constructor(options?: OpenBrowserUseBrowserOptions);
    connect(): Promise<this>;
    close(): void;
    newTab(options?: OpenBrowserUseGotoOptions & {
        url?: string;
    }): Promise<OpenBrowserUseTab>;
    tab(tabId: number): OpenBrowserUseTab;
    getTabs(): Promise<JsonValue>;
}
export declare class OpenBrowserUseTab {
    readonly browser: OpenBrowserUseBrowser;
    readonly id: number;
    readonly playwright: OpenBrowserUseTabPlaywright;
    constructor(browser: OpenBrowserUseBrowser, id: number);
    goto(url: string, options?: OpenBrowserUseGotoOptions): Promise<JsonValue>;
    waitForLoadState(options?: OpenBrowserUseWaitForLoadStateOptions): Promise<void>;
    domSnapshot(): Promise<string>;
    pageInfo(options?: OpenBrowserUseTextOptions): Promise<OpenBrowserUsePageInfo>;
    text(options?: OpenBrowserUseTextOptions): Promise<OpenBrowserUseTextResult>;
    snapshot(options?: {
        limit?: number;
    }): Promise<OpenBrowserUseSnapshotResult>;
    screenshot(options?: OpenBrowserUseScreenshotOptions): Promise<OpenBrowserUseScreenshotResult>;
    click(ref: string | number): Promise<OpenBrowserUseInteractionResult>;
    fill(ref: string | number, text: string): Promise<OpenBrowserUseInteractionResult>;
    evaluate(expression: string, options?: {
        awaitPromise?: boolean;
    }): Promise<JsonValue>;
    close(): Promise<JsonValue>;
    private resolveScreenshotClip;
}
export declare class OpenBrowserUseTabPlaywright {
    readonly tab: OpenBrowserUseTab;
    constructor(tab: OpenBrowserUseTab);
    waitForLoadState(options?: OpenBrowserUseWaitForLoadStateOptions): Promise<void>;
    domSnapshot(): Promise<string>;
    pageInfo(options?: OpenBrowserUseTextOptions): Promise<OpenBrowserUsePageInfo>;
    text(options?: OpenBrowserUseTextOptions): Promise<OpenBrowserUseTextResult>;
    snapshot(options?: {
        limit?: number;
    }): Promise<OpenBrowserUseSnapshotResult>;
    screenshot(options?: OpenBrowserUseScreenshotOptions): Promise<OpenBrowserUseScreenshotResult>;
    click(ref: string | number): Promise<OpenBrowserUseInteractionResult>;
    fill(ref: string | number, text: string): Promise<OpenBrowserUseInteractionResult>;
}
export declare class OpenBrowserUseCdp {
    #private;
    readonly client: OpenBrowserUseClient;
    constructor(client: OpenBrowserUseClient);
    call(tabId: number, method: string, commandParams?: BrowserUseRequestParams, options?: {
        timeoutMs?: number;
    }): Promise<JsonValue>;
    evaluate(tabId: number, expression: string, options?: {
        awaitPromise?: boolean;
    }): Promise<JsonValue>;
    navigate(tabId: number, url: string, options?: OpenBrowserUseGotoOptions): Promise<JsonValue>;
    waitForLoadState(tabId: number, options?: OpenBrowserUseWaitForLoadStateOptions): Promise<void>;
    readDocumentState(tabId: number): Promise<{
        href?: string;
        readyState?: string;
    } | undefined>;
    waitForEvent(tabId: number, predicate: (notification: OpenBrowserUseNotification) => boolean, options?: {
        timeoutMs?: number;
        timeoutMessage?: string;
    }): Promise<OpenBrowserUseNotification>;
    waitForLoadEvent(tabId: number, state: OpenBrowserUseLoadState, timeoutMs: number): Promise<void>;
    ensureAttached(tabId: number): Promise<void>;
}
export declare function encodeFrame(value: JsonValue | Record<string, unknown>): Buffer;
