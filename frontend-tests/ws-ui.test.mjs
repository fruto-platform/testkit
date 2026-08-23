import assert from "node:assert/strict";
import { test } from "node:test";

class FakeElement {
  constructor() {
    this.children = [];
    this.listeners = new Map();
    this.className = "";
    this.dataset = {};
    this.disabled = false;
    this.hidden = false;
    this.scrollHeight = 0;
    this.scrollTop = 0;
    this.textContent = "";
    this.value = "";
  }

  addEventListener(type, listener) {
    const listeners = this.listeners.get(type) || [];
    listeners.push(listener);
    this.listeners.set(type, listeners);
  }

  dispatch(type, event = {}) {
    for (const listener of this.listeners.get(type) || []) {
      listener({ preventDefault() {}, ...event });
    }
  }

  append(...children) {
    this.children.push(...children);
  }

  querySelector(selector) {
    return this.controls.get(selector);
  }

  select() {
    this.selected = true;
  }
}

class FakeWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static current = null;

  constructor() {
    this.readyState = FakeWebSocket.CONNECTING;
    this.listeners = new Map();
    this.sent = [];
    FakeWebSocket.current = this;
  }

  addEventListener(type, listener) {
    const listeners = this.listeners.get(type) || [];
    listeners.push(listener);
    this.listeners.set(type, listeners);
  }

  emit(type, event = {}) {
    if (type === "open") this.readyState = FakeWebSocket.OPEN;
    if (type === "close") this.readyState = 3;
    for (const listener of this.listeners.get(type) || []) listener(event);
  }

  send(payload) {
    this.sent.push(payload);
  }

  close() {
    this.readyState = 2;
  }
}

globalThis.WebSocket = FakeWebSocket;
globalThis.window = {
  location: {
    href: "http://testkit.example/",
    protocol: "http:",
  },
};
globalThis.document = {
  createElement: () => new FakeElement(),
};

const { mountWebSocketClient } = await import("../static/ws-ui.js");

function createUI() {
  const root = new FakeElement();
  root.dataset.wsUrl = "/ws";
  root.controls = new Map([
    ["[data-ws-connect]", new FakeElement()],
    ["[data-ws-disconnect]", new FakeElement()],
    ["[data-ws-form]", new FakeElement()],
    ["[data-ws-message]", new FakeElement()],
    ["[data-ws-send]", new FakeElement()],
    ["[data-ws-status]", new FakeElement()],
    ["[data-ws-status-text]", new FakeElement()],
    ["[data-ws-hint]", new FakeElement()],
    ["[data-ws-log]", new FakeElement()],
    ["[data-ws-empty]", new FakeElement()],
  ]);
  mountWebSocketClient(root);
  return root;
}

test("renders state changes, close information and JSON with textContent", () => {
  const root = createUI();
  const statusText = root.controls.get("[data-ws-status-text]");
  const connectButton = root.controls.get("[data-ws-connect]");
  const messageInput = root.controls.get("[data-ws-message]");
  const form = root.controls.get("[data-ws-form]");
  const log = root.controls.get("[data-ws-log]");

  assert.equal(statusText.textContent, "Disconnected");
  connectButton.dispatch("click");
  const socket = FakeWebSocket.current;
  socket.emit("open");
  assert.equal(statusText.textContent, "Connected");

  const unsafeMessage = '<img src=x onerror="alert(1)">';
  messageInput.value = unsafeMessage;
  form.dispatch("submit");
  assert.deepEqual(JSON.parse(socket.sent[0]), { message: unsafeMessage });

  const received = { message: unsafeMessage, version: "dev" };
  socket.emit("message", { data: JSON.stringify(received) });
  const payload = log.children.at(-1).children.at(-1);
  assert.equal(payload.textContent, JSON.stringify(received));
  assert.equal(payload.innerHTML, undefined);

  socket.emit("close", { code: 1006, reason: "network failure", wasClean: false });
  assert.equal(statusText.textContent, "Disconnected");
  assert.match(log.children.at(-1).children.at(-1).textContent, /1006/);
  assert.match(log.children.at(-1).children.at(-1).textContent, /network failure/);
});

test("shows Error and prevents sending when the client is disconnected", () => {
  const root = createUI();
  const statusText = root.controls.get("[data-ws-status-text]");
  const messageInput = root.controls.get("[data-ws-message]");
  const form = root.controls.get("[data-ws-form]");

  messageInput.value = "hello";
  form.dispatch("submit");

  assert.equal(statusText.textContent, "Error");
  assert.match(root.controls.get("[data-ws-hint]").textContent, /Connect this client/);
});
