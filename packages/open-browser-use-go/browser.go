package obu

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Browser struct {
	Client *Client
	CDP    *CDP
}

func NewBrowser(client *Client) *Browser {
	return &Browser{Client: client, CDP: NewCDP(client)}
}

func (b *Browser) Connect() error {
	return b.Client.Connect()
}

func (b *Browser) Close() error {
	return b.Client.Close()
}

func (b *Browser) NewTab(options ...GotoOptions) (*Tab, error) {
	result, err := b.Client.CreateTab()
	if err != nil {
		return nil, err
	}
	tabID, err := tabIDFromValue(result, "createTab response")
	if err != nil {
		return nil, err
	}
	tab := b.Tab(tabID)
	if len(options) > 0 && options[0].URL != "" {
		if _, err := tab.Goto(options[0].URL, options[0]); err != nil {
			return nil, err
		}
	}
	return tab, nil
}

func (b *Browser) Tab(tabID int) *Tab {
	tab := &Tab{Browser: b, ID: tabID}
	tab.initPlaywright()
	return tab
}

func (b *Browser) GetTabs() (any, error) {
	return b.Client.GetTabs()
}

type Tab struct {
	Browser    *Browser
	ID         int
	Playwright *TabPlaywright
}

func (t *Tab) Goto(url string, options ...GotoOptions) (any, error) {
	option := GotoOptions{}
	if len(options) > 0 {
		option = options[0]
	}
	return t.Browser.CDP.Navigate(t.ID, url, option)
}

func (t *Tab) WaitForLoadState(options ...WaitForLoadStateOptions) error {
	option := WaitForLoadStateOptions{}
	if len(options) > 0 {
		option = options[0]
	}
	return t.Browser.CDP.WaitForLoadState(t.ID, option)
}

func (t *Tab) DOMSnapshot() (string, error) {
	value, err := t.Evaluate("document.body?.innerText ?? ''")
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", nil
	}
	return fmt.Sprint(value), nil
}

func (t *Tab) PageInfo(options ...TextOptions) (PageInfo, error) {
	option := TextOptions{}
	if len(options) > 0 {
		option = options[0]
	}
	value, err := t.Evaluate(pageInfoExpression(option.Selector, option.MaxChars))
	if err != nil {
		return PageInfo{}, err
	}
	result, ok := value.(map[string]any)
	if !ok {
		return PageInfo{}, errors.New("page info returned no object value")
	}
	return PageInfo{
		Title:      fmt.Sprint(result["title"]),
		URL:        fmt.Sprint(result["url"]),
		ReadyState: fmt.Sprint(result["readyState"]),
		Text:       fmt.Sprint(result["text"]),
	}, nil
}

func (t *Tab) Text(options ...TextOptions) (TextResult, error) {
	option := TextOptions{}
	if len(options) > 0 {
		option = options[0]
	}
	value, err := t.Evaluate(pageTextExpression(option.Selector, option.MaxChars))
	if err != nil {
		return TextResult{}, err
	}
	if value == nil {
		return TextResult{}, nil
	}
	return TextResult{Text: fmt.Sprint(value)}, nil
}

func (t *Tab) Snapshot(limit int) (SnapshotResult, error) {
	value, err := t.Evaluate(snapshotExpression(limit))
	if err != nil {
		return SnapshotResult{}, err
	}
	rawItems, ok := value.([]any)
	if !ok {
		return SnapshotResult{}, errors.New("snapshot returned no element list")
	}
	items := make([]SnapshotItem, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, err := snapshotItemFromValue(rawItem)
		if err != nil {
			return SnapshotResult{}, err
		}
		items = append(items, item)
	}
	return SnapshotResult{Items: items}, nil
}

func (t *Tab) Screenshot(options ...ScreenshotOptions) (ScreenshotResult, error) {
	option := ScreenshotOptions{}
	if len(options) > 0 {
		option = options[0]
	}
	commandParams := Params{"format": "png"}
	var clip map[string]any
	var err error
	if option.Selector != "" {
		clip, err = t.resolveScreenshotClip(selectorClipExpression(option.Selector), "selector")
		if err != nil {
			return ScreenshotResult{}, err
		}
		commandParams["clip"] = clip
	} else if option.FullPage {
		clip, err = t.resolveScreenshotClip(fullPageClipExpression(), "full-page")
		if err != nil {
			return ScreenshotResult{}, err
		}
		commandParams["clip"] = clip
	}
	value, err := t.Browser.CDP.Call(t.ID, "Page.captureScreenshot", commandParams, CDPCallOptions{})
	if err != nil {
		return ScreenshotResult{}, err
	}
	result, ok := value.(map[string]any)
	if !ok {
		return ScreenshotResult{}, errors.New("screenshot returned no object")
	}
	data, _ := result["data"].(string)
	if data == "" {
		return ScreenshotResult{}, errors.New("screenshot returned no data")
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return ScreenshotResult{}, err
	}
	return ScreenshotResult{
		Data:     data,
		Bytes:    len(decoded),
		Format:   "png",
		TabID:    t.ID,
		Selector: option.Selector,
		Clip:     clip,
	}, nil
}

func (t *Tab) Click(ref any) (InteractionResult, error) {
	value, err := t.Evaluate(clickExpression(strings.TrimPrefix(fmt.Sprint(ref), "@")))
	if err != nil {
		return InteractionResult{}, err
	}
	return interactionResultFromValue("click", value)
}

func (t *Tab) Fill(ref any, text string) (InteractionResult, error) {
	value, err := t.Evaluate(fillExpression(strings.TrimPrefix(fmt.Sprint(ref), "@"), text))
	if err != nil {
		return InteractionResult{}, err
	}
	return interactionResultFromValue("fill", value)
}

func (t *Tab) Evaluate(expression string, awaitPromise ...bool) (any, error) {
	option := EvaluateOptions{}
	if len(awaitPromise) > 0 {
		option.AwaitPromise = &awaitPromise[0]
	}
	return t.Browser.CDP.Evaluate(t.ID, expression, option)
}

func (t *Tab) Title() (string, error) {
	value, err := t.Evaluate("document.title ?? ''")
	if err != nil {
		return "", err
	}
	return fmt.Sprint(value), nil
}

func (t *Tab) URL() (string, error) {
	value, err := t.Evaluate("location.href")
	if err != nil {
		return "", err
	}
	return fmt.Sprint(value), nil
}

func (t *Tab) WaitForTimeout(timeout time.Duration) error {
	if timeout < 0 {
		return errors.New("timeout must be non-negative")
	}
	time.Sleep(timeout)
	return nil
}

func (t *Tab) Locator(selector string) *Locator {
	return &Locator{Tab: t, Selector: selector}
}

func (t *Tab) Close() (any, error) {
	return t.Browser.CDP.Call(t.ID, "Page.close", nil, CDPCallOptions{})
}

func (t *Tab) resolveScreenshotClip(expression string, label string) (map[string]any, error) {
	value, err := t.Evaluate(expression)
	if err != nil {
		return nil, err
	}
	result, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s screenshot clip returned no object", label)
	}
	if ok, _ := result["ok"].(bool); !ok {
		reason, _ := result["reason"].(string)
		if reason == "" {
			reason = "unknown"
		}
		return nil, fmt.Errorf("%s screenshot clip failed: %s", label, reason)
	}
	clip, ok := result["clip"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s screenshot clip returned no clip", label)
	}
	return clip, nil
}

type TabPlaywright struct {
	tab *Tab
}

func (t *Tab) initPlaywright() {
	if t.Playwright == nil {
		t.Playwright = &TabPlaywright{}
	}
	t.Playwright.tab = t
}

func (p *TabPlaywright) WaitForLoadState(options ...WaitForLoadStateOptions) error {
	return p.tab.WaitForLoadState(options...)
}

func (p *TabPlaywright) DOMSnapshot() (string, error) {
	return p.tab.DOMSnapshot()
}

func (p *TabPlaywright) PageInfo(options ...TextOptions) (PageInfo, error) {
	return p.tab.PageInfo(options...)
}

func (p *TabPlaywright) Text(options ...TextOptions) (TextResult, error) {
	return p.tab.Text(options...)
}

func (p *TabPlaywright) Snapshot(limit int) (SnapshotResult, error) {
	return p.tab.Snapshot(limit)
}

func (p *TabPlaywright) Screenshot(options ...ScreenshotOptions) (ScreenshotResult, error) {
	return p.tab.Screenshot(options...)
}

func (p *TabPlaywright) Click(ref any) (InteractionResult, error) {
	return p.tab.Click(ref)
}

func (p *TabPlaywright) Fill(ref any, text string) (InteractionResult, error) {
	return p.tab.Fill(ref, text)
}

func (p *TabPlaywright) Title() (string, error) {
	return p.tab.Title()
}

func (p *TabPlaywright) URL() (string, error) {
	return p.tab.URL()
}

func (p *TabPlaywright) WaitForTimeout(timeout time.Duration) error {
	return p.tab.WaitForTimeout(timeout)
}

func (p *TabPlaywright) Locator(selector string) *Locator {
	return p.tab.Locator(selector)
}

type Locator struct {
	Tab      *Tab
	Selector string
}

func (l *Locator) InnerText(timeout time.Duration) (string, error) {
	if l.Selector == "" {
		return "", errors.New("locator requires a selector")
	}
	if timeout < 0 {
		return "", errors.New("timeout must be non-negative")
	}
	if timeout == 0 {
		timeout = DefaultNavigationTimeout
	}
	value, err := l.Tab.Evaluate(locatorInnerTextExpression(l.Selector, timeout), true)
	if err != nil {
		return "", err
	}
	if value == nil {
		return "", nil
	}
	return fmt.Sprint(value), nil
}

type CDP struct {
	client        *Client
	attachedTabID map[int]bool
}

func NewCDP(client *Client) *CDP {
	return &CDP{client: client, attachedTabID: map[int]bool{}}
}

type CDPCallOptions struct {
	Timeout time.Duration
}

type EvaluateOptions struct {
	AwaitPromise *bool
}

type GotoOptions struct {
	URL       string
	WaitUntil string
	Timeout   time.Duration
}

type WaitForLoadStateOptions struct {
	State   string
	Timeout time.Duration
}

type TextOptions struct {
	Selector string
	MaxChars int
}

type PageInfo struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	ReadyState string `json:"readyState"`
	Text       string `json:"text"`
}

type TextResult struct {
	Text string `json:"text"`
}

type SnapshotItem struct {
	Index    int    `json:"index"`
	Tag      string `json:"tag"`
	Text     string `json:"text"`
	Type     string `json:"type,omitempty"`
	Href     string `json:"href,omitempty"`
	Selector string `json:"selector,omitempty"`
}

type SnapshotResult struct {
	Items []SnapshotItem `json:"items"`
}

type ScreenshotOptions struct {
	Selector string
	FullPage bool
}

type ScreenshotResult struct {
	Data     string         `json:"data"`
	Bytes    int            `json:"bytes"`
	Format   string         `json:"format"`
	TabID    int            `json:"tabId"`
	Selector string         `json:"selector,omitempty"`
	Clip     map[string]any `json:"clip,omitempty"`
}

type InteractionResult struct {
	OK          bool           `json:"ok"`
	Ref         string         `json:"ref"`
	Action      string         `json:"action,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	Tag         string         `json:"tag,omitempty"`
	Text        string         `json:"text,omitempty"`
	Rect        map[string]any `json:"rect,omitempty"`
	ValueLength int            `json:"valueLength,omitempty"`
}

func (c *CDP) Call(tabID int, method string, commandParams Params, options CDPCallOptions) (any, error) {
	if err := c.EnsureAttached(tabID); err != nil {
		return nil, err
	}
	params := Params{
		"target":        Params{"tabId": tabID},
		"method":        method,
		"commandParams": commandParams,
	}
	if options.Timeout > 0 {
		params["timeoutMs"] = int(options.Timeout / time.Millisecond)
	}
	return c.client.Request("executeCdp", params)
}

func (c *CDP) Evaluate(tabID int, expression string, options EvaluateOptions) (any, error) {
	commandParams := Params{
		"expression":    expression,
		"returnByValue": true,
	}
	if options.AwaitPromise != nil {
		commandParams["awaitPromise"] = *options.AwaitPromise
	}
	result, err := c.Call(tabID, "Runtime.evaluate", commandParams, CDPCallOptions{})
	if err != nil {
		return nil, err
	}
	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, nil
	}
	if exception, ok := resultMap["exceptionDetails"].(map[string]any); ok {
		if text, ok := exception["text"].(string); ok && text != "" {
			return nil, errors.New(text)
		}
		return nil, errors.New("Open Browser Use evaluation failed")
	}
	remoteObject, ok := resultMap["result"].(map[string]any)
	if !ok {
		return nil, nil
	}
	return remoteObject["value"], nil
}

func (c *CDP) Navigate(tabID int, url string, options GotoOptions) (any, error) {
	if url == "" {
		return nil, errors.New("goto requires a URL")
	}
	waitUntil := options.WaitUntil
	if waitUntil == "" {
		waitUntil = LoadStateLoad
	}
	if err := assertSupportedLoadState(waitUntil); err != nil {
		return nil, err
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = DefaultNavigationTimeout
	}
	if _, err := c.Call(tabID, "Page.enable", nil, CDPCallOptions{}); err != nil {
		return nil, err
	}
	result, err := c.Call(tabID, "Page.navigate", Params{"url": url}, CDPCallOptions{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	if resultMap, ok := result.(map[string]any); ok {
		if errorText, ok := resultMap["errorText"].(string); ok && errorText != "" {
			return nil, fmt.Errorf("browser failed to navigate tab %d: %s", tabID, errorText)
		}
	}
	if err := c.WaitForLoadState(tabID, WaitForLoadStateOptions{State: waitUntil, Timeout: timeout}); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CDP) WaitForLoadState(tabID int, options WaitForLoadStateOptions) error {
	state := options.State
	if state == "" {
		state = LoadStateLoad
	}
	if err := assertSupportedLoadState(state); err != nil {
		return err
	}
	timeout := options.Timeout
	if timeout == 0 {
		timeout = DefaultNavigationTimeout
	}
	if _, err := c.Call(tabID, "Page.enable", nil, CDPCallOptions{}); err != nil {
		return err
	}
	deadline := time.Now().Add(timeout)
	for {
		documentState, _ := c.ReadDocumentState(tabID)
		if documentStateMatches(documentState, state) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s in tab %d", state, tabID)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *CDP) ReadDocumentState(tabID int) (map[string]any, error) {
	value, err := c.Evaluate(tabID, "({ href: window.location.href, readyState: document.readyState })", EvaluateOptions{})
	if err != nil {
		return nil, err
	}
	if result, ok := value.(map[string]any); ok {
		return result, nil
	}
	return nil, nil
}

func (c *CDP) EnsureAttached(tabID int) error {
	if c.attachedTabID[tabID] {
		return nil
	}
	if _, err := c.client.Attach(tabID); err != nil {
		return err
	}
	c.attachedTabID[tabID] = true
	return nil
}

func tabIDFromValue(value any, label string) (int, error) {
	result, ok := value.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("%s did not include a tab object", label)
	}
	switch id := result["id"].(type) {
	case float64:
		if id > 0 && id == float64(int(id)) {
			return int(id), nil
		}
	case int:
		if id > 0 {
			return id, nil
		}
	case string:
		parsed, err := strconv.Atoi(id)
		if err == nil && parsed > 0 {
			return parsed, nil
		}
	}
	return 0, fmt.Errorf("%s did not include a numeric tab id", label)
}

func assertSupportedLoadState(state string) error {
	if state != LoadStateDOMContentLoaded && state != LoadStateLoad {
		return fmt.Errorf("unsupported load state %q. Use %q or %q", state, LoadStateDOMContentLoaded, LoadStateLoad)
	}
	return nil
}

func documentStateMatches(documentState map[string]any, state string) bool {
	readyState, _ := documentState["readyState"].(string)
	return readyState == "complete" || (state == LoadStateDOMContentLoaded && readyState == "interactive")
}

func pageTextExpression(selector string, maxChars int) string {
	selectorJSON, _ := json.Marshal(selector)
	return fmt.Sprintf(`(() => {
  const selector = %s;
  const root = selector ? document.querySelector(selector) : document.body;
  let text = root?.innerText ?? "";
  if (%d > 0 && text.length > %d) text = text.slice(0, %d);
  return text;
})()`, selectorJSON, maxChars, maxChars, maxChars)
}

func pageInfoExpression(selector string, maxChars int) string {
	selectorJSON, _ := json.Marshal(selector)
	return fmt.Sprintf(`(() => {
  const selector = %s;
  const root = selector ? document.querySelector(selector) : document.body;
  let text = root?.innerText ?? "";
  if (%d > 0 && text.length > %d) text = text.slice(0, %d);
  return { title: document.title ?? "", url: location.href, readyState: document.readyState, text };
})()`, selectorJSON, maxChars, maxChars, maxChars)
}

func snapshotExpression(limit int) string {
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

func selectorClipExpression(selector string) string {
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
})()`, selectorJSON)
}

func fullPageClipExpression() string {
	return `(() => {
  const doc = document.documentElement;
  const body = document.body;
  const width = Math.max(doc?.scrollWidth ?? 0, body?.scrollWidth ?? 0, doc?.clientWidth ?? 0, window.innerWidth);
  const height = Math.max(doc?.scrollHeight ?? 0, body?.scrollHeight ?? 0, doc?.clientHeight ?? 0, window.innerHeight);
  return { ok: true, clip: { x: 0, y: 0, width, height, scale: 1 } };
})()`
}

func interactionPreludeExpression(ref string) string {
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
  const y = rect.top + rect.height / 2;`, refJSON)
}

func clickExpression(ref string) string {
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
})()`, interactionPreludeExpression(ref))
}

func fillExpression(ref string, text string) string {
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
})()`, interactionPreludeExpression(ref), textJSON)
}

func snapshotItemFromValue(value any) (SnapshotItem, error) {
	item, ok := value.(map[string]any)
	if !ok {
		return SnapshotItem{}, errors.New("snapshot returned a non-object element")
	}
	index := 0
	switch rawIndex := item["index"].(type) {
	case float64:
		index = int(rawIndex)
	case int:
		index = rawIndex
	}
	return SnapshotItem{
		Index:    index,
		Tag:      fmt.Sprint(item["tag"]),
		Text:     fmt.Sprint(item["text"]),
		Type:     fmt.Sprint(item["type"]),
		Href:     fmt.Sprint(item["href"]),
		Selector: fmt.Sprint(item["selector"]),
	}, nil
}

func interactionResultFromValue(action string, value any) (InteractionResult, error) {
	result, ok := value.(map[string]any)
	if !ok {
		return InteractionResult{}, fmt.Errorf("%s returned no object value", action)
	}
	output := InteractionResult{
		OK:     boolValue(result["ok"]),
		Ref:    fmt.Sprint(result["ref"]),
		Action: stringValue(result["action"]),
		Reason: stringValue(result["reason"]),
		Tag:    stringValue(result["tag"]),
		Text:   stringValue(result["text"]),
	}
	if rect, ok := result["rect"].(map[string]any); ok {
		output.Rect = rect
	}
	switch valueLength := result["valueLength"].(type) {
	case float64:
		output.ValueLength = int(valueLength)
	case int:
		output.ValueLength = valueLength
	}
	if !output.OK {
		reason := output.Reason
		if reason == "" {
			reason = "unknown"
		}
		return output, fmt.Errorf("%s failed: %s", action, reason)
	}
	return output, nil
}

func boolValue(value any) bool {
	result, _ := value.(bool)
	return result
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}

func locatorInnerTextExpression(selector string, timeout time.Duration) string {
	selectorJSON, _ := json.Marshal(selector)
	return fmt.Sprintf(`(async () => {
  const selector = %s;
  const deadline = performance.now() + %d;
  while (true) {
    const element = document.querySelector(selector);
    if (element) {
      return element.innerText ?? element.textContent ?? "";
    }
    if (performance.now() >= deadline) {
      throw new Error("Timed out waiting for locator " + selector);
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
})()`, selectorJSON, int(timeout/time.Millisecond))
}
