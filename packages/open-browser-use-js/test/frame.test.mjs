import assert from "node:assert/strict";
import { createServer } from "node:net";
import { endianness } from "node:os";
import test from "node:test";

import { OpenBrowserUseClient, connectOpenBrowserUse, encodeFrame } from "../dist/index.js";

test("encodes native-endian length-prefixed JSON frames", () => {
  const frame = encodeFrame({ id: 1, method: "getInfo" });
  const length = endianness() === "LE" ? frame.readUInt32LE(0) : frame.readUInt32BE(0);
  assert.equal(length, frame.length - 4);
  assert.deepEqual(JSON.parse(frame.subarray(4).toString("utf8")), {
    id: 1,
    method: "getInfo"
  });
});

test("dispatches JSON-RPC notifications from the native socket", async () => {
  const server = createServer((socket) => {
    socket.write(
      encodeFrame({
        jsonrpc: "2.0",
        method: "onDownloadChange",
        params: { id: "1", status: "started" }
      })
    );
  });
  const socketPath = await listenTestServer(server);

  const client = new OpenBrowserUseClient({ socketPath });
  try {
    const notification = await new Promise((resolve) => {
      client.onNotification(resolve);
      void client.connect();
    });
    assert.deepEqual(notification, {
      method: "onDownloadChange",
      params: { id: "1", status: "started" }
    });
  } finally {
    client.close();
    server.close();
    await new Promise((resolve) => server.once("close", resolve));
  }
});

test("high-level browser tabs can goto, wait for load state, and read a DOM snapshot", async () => {
  const calls = [];
  const server = createServer((socket) => {
    let pending = Buffer.alloc(0);
    socket.on("data", (chunk) => {
      pending = Buffer.concat([pending, chunk]);
      while (pending.length >= 4) {
        const length = endianness() === "LE" ? pending.readUInt32LE(0) : pending.readUInt32BE(0);
        if (pending.length < 4 + length) {
          return;
        }
        const request = JSON.parse(pending.subarray(4, 4 + length).toString("utf8"));
        pending = pending.subarray(4 + length);
        calls.push([request.method, request.params.method ?? null]);
        let result = {};
        if (request.method === "createTab") {
          result = { id: 123 };
        } else if (request.method === "executeCdp" && request.params.method === "Page.navigate") {
          socket.write(
            encodeFrame({
              jsonrpc: "2.0",
              method: "onCDPEvent",
              params: {
                source: { tabId: 123 },
                method: "Page.domContentEventFired",
                params: {}
              }
            })
          );
          result = { frameId: "frame-1" };
        } else if (request.method === "executeCdp" && request.params.method === "Runtime.evaluate") {
          const expression = request.params.commandParams.expression;
          result = expression.includes("readyState")
            ? { result: { value: { href: "https://example.test/issues", readyState: "interactive" } } }
            : { result: { value: "Open\nClosed\nIssues\nStarred" } };
        }
        socket.write(encodeFrame({ jsonrpc: "2.0", id: request.id, result }));
      }
    });
  });
  const socketPath = await listenTestServer(server);

  const browser = await connectOpenBrowserUse({ socketPath });
  try {
    const tab = await browser.newTab();
    await tab.goto("https://example.test/issues", { waitUntil: "domcontentloaded", timeoutMs: 1000 });
    await tab.playwright.waitForLoadState({ state: "domcontentloaded", timeoutMs: 1000 });
    assert.equal(await tab.playwright.domSnapshot(), "Open\nClosed\nIssues\nStarred");
    assert.deepEqual(calls, [
      ["createTab", null],
      ["attach", null],
      ["executeCdp", "Page.enable"],
      ["executeCdp", "Page.navigate"],
      ["executeCdp", "Page.enable"],
      ["executeCdp", "Runtime.evaluate"],
      ["executeCdp", "Runtime.evaluate"]
    ]);
  } finally {
    browser.close();
    server.close();
    await new Promise((resolve) => server.once("close", resolve));
  }
});

test("high-level browser tabs expose structured read, screenshot, and interaction helpers", async () => {
  const calls = [];
  const pngData = Buffer.from("png").toString("base64");
  const server = createServer((socket) => {
    let pending = Buffer.alloc(0);
    socket.on("data", (chunk) => {
      pending = Buffer.concat([pending, chunk]);
      while (pending.length >= 4) {
        const length = endianness() === "LE" ? pending.readUInt32LE(0) : pending.readUInt32BE(0);
        if (pending.length < 4 + length) {
          return;
        }
        const request = JSON.parse(pending.subarray(4, 4 + length).toString("utf8"));
        pending = pending.subarray(4 + length);
        const method = request.method;
        const cdpMethod = request.params.method ?? null;
        const expression = request.params.commandParams?.expression ?? "";
        calls.push([method, cdpMethod]);
        let result = {};
        if (method === "executeCdp" && cdpMethod === "Runtime.evaluate") {
          if (expression.includes("document.title")) {
            result = {
              result: {
                value: {
                  title: "Example",
                  url: "https://example.test",
                  readyState: "complete",
                  text: "Hello"
                }
              }
            };
          } else if (expression.includes("__obu_refs")) {
            result = {
              result: {
                value: [{ index: 1, tag: "button", text: "Save", type: "", href: "", selector: "#save" }]
              }
            };
          } else if (expression.includes('action: "click"')) {
            result = {
              result: { value: { ok: true, ref: "1", action: "click", tag: "button", text: "Save" } }
            };
          } else if (expression.includes('action: "fill"')) {
            result = {
              result: { value: { ok: true, ref: "2", action: "fill", tag: "input", text: "hello", valueLength: 5 } }
            };
          } else {
            result = { result: { value: "Hello" } };
          }
        } else if (method === "executeCdp" && cdpMethod === "Page.captureScreenshot") {
          result = { data: pngData };
        }
        socket.write(encodeFrame({ jsonrpc: "2.0", id: request.id, result }));
      }
    });
  });
  const socketPath = await listenTestServer(server);

  const browser = await connectOpenBrowserUse({ socketPath });
  try {
    const tab = browser.tab(123);
    assert.deepEqual(await tab.pageInfo({ maxChars: 5 }), {
      title: "Example",
      url: "https://example.test",
      readyState: "complete",
      text: "Hello"
    });
    assert.deepEqual(await tab.text({ selector: "main", maxChars: 5 }), { text: "Hello" });
    assert.deepEqual(await tab.snapshot({ limit: 1 }), {
      items: [{ index: 1, tag: "button", text: "Save", type: "", href: "", selector: "#save" }]
    });
    assert.deepEqual(await tab.screenshot(), {
      data: pngData,
      bytes: 3,
      format: "png",
      tabId: 123
    });
    assert.equal((await tab.click("@1")).action, "click");
    assert.equal((await tab.fill("@2", "hello")).valueLength, 5);
    assert.deepEqual(calls, [
      ["attach", null],
      ["executeCdp", "Runtime.evaluate"],
      ["executeCdp", "Runtime.evaluate"],
      ["executeCdp", "Runtime.evaluate"],
      ["executeCdp", "Page.captureScreenshot"],
      ["executeCdp", "Runtime.evaluate"],
      ["executeCdp", "Runtime.evaluate"]
    ]);
  } finally {
    browser.close();
    server.close();
    await new Promise((resolve) => server.once("close", resolve));
  }
});

test("sends file chooser wrapper requests", async () => {
  const server = createServer((socket) => {
    socket.once("data", (chunk) => {
      const length = endianness() === "LE" ? chunk.readUInt32LE(0) : chunk.readUInt32BE(0);
      const request = JSON.parse(chunk.subarray(4, 4 + length).toString("utf8"));
      assert.equal(request.method, "waitForFileChooser");
      assert.equal(request.params.tabId, 123);
      socket.write(
        encodeFrame({
          jsonrpc: "2.0",
          id: request.id,
          result: { fileChooserId: "chooser-1", isMultiple: false }
        })
      );
    });
  });
  const socketPath = await listenTestServer(server);

  const client = new OpenBrowserUseClient({ socketPath });
  try {
    assert.deepEqual(await client.waitForFileChooser(123), {
      fileChooserId: "chooser-1",
      isMultiple: false
    });
  } finally {
    client.close();
    server.close();
    await new Promise((resolve) => server.once("close", resolve));
  }
});

test("sends download and clipboard wrapper requests", async () => {
  const expected = [
    ["waitForDownload", { tabId: 123, timeoutMs: 5000 }],
    ["downloadPath", { downloadId: "download-1" }],
    ["readClipboardText", { tabId: 123 }],
    ["writeClipboardText", { tabId: 123, text: "hello" }]
  ];
  const server = createServer((socket) => {
    let index = 0;
    socket.on("data", (chunk) => {
      const length = endianness() === "LE" ? chunk.readUInt32LE(0) : chunk.readUInt32BE(0);
      const request = JSON.parse(chunk.subarray(4, 4 + length).toString("utf8"));
      const [method, params] = expected[index++];
      assert.equal(request.method, method);
      assert.deepEqual(
        Object.fromEntries(Object.entries(params).map(([key, value]) => [key, request.params[key]])),
        params
      );
      socket.write(encodeFrame({ jsonrpc: "2.0", id: request.id, result: {} }));
    });
  });
  const socketPath = await listenTestServer(server);

  const client = new OpenBrowserUseClient({ socketPath });
  try {
    await client.waitForDownload(123, 5000);
    await client.downloadPath("download-1");
    await client.readClipboardText(123);
    await client.writeClipboardText(123, "hello");
    assert.equal(expected.length, 4);
  } finally {
    client.close();
    server.close();
    await new Promise((resolve) => server.once("close", resolve));
  }
});

async function listenTestServer(server) {
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  const address = server.address();
  assert.equal(typeof address, "object");
  return `${address.address}:${address.port}`;
}
