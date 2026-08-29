import assert from "node:assert/strict";
import { test } from "node:test";

class FakeWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static instances = [];

  constructor(url) {
    this.url = url;
    this.readyState = FakeWebSocket.CONNECTING;
    this.listeners = new Map();
    this.sent = [];
    this.closeArguments = null;
    FakeWebSocket.instances.push(this);
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

  close(code, reason) {
    this.closeArguments = { code, reason };
    this.readyState = 2;
  }
}

globalThis.WebSocket = FakeWebSocket;
globalThis.window = {
  location: {
    href: "http://testkit.example/dashboard",
    protocol: "http:",
  },
};

const { createWebSocketClient } = await import("../static/ws-client.js");

const browserCorrelationID = "018f47de-1234-7abc-8def-0123456789ab";

function createClient() {
  const events = [];
  const states = [];
  const client = createWebSocketClient({
    url: "/ws",
    onEvent: (event) => events.push(event),
    onStateChange: (state) => states.push(state),
    createCorrelationIDImpl: () => browserCorrelationID,
  });
  return { client, events, states };
}

test("connects, sends the contract, receives broadcast and exposes close details", () => {
  FakeWebSocket.instances.length = 0;
  const { client, events, states } = createClient();

  client.connect();
  const socket = FakeWebSocket.instances[0];
  assert.equal(socket.url, `ws://testkit.example/ws?correlation_id=${browserCorrelationID}`);
  assert.equal(states.at(-1).state, "connecting");
  assert.equal(events.at(-1).detail.correlation_id, browserCorrelationID);

  socket.emit("open");
  assert.equal(states.at(-1).state, "connected");
  assert.equal(events.at(-1).detail.correlation_id, browserCorrelationID);

  assert.equal(client.send("hello"), true);
  assert.deepEqual(JSON.parse(socket.sent[0]), { message: "hello" });
  assert.equal(events.at(-1).type, "message.sent");

  socket.emit("message", {
    data: JSON.stringify({ message: "hello", version: "dev" }),
  });
  assert.equal(events.at(-1).type, "broadcast.received");
  assert.deepEqual(events.at(-1).detail, { message: "hello", version: "dev" });

  socket.emit("close", { code: 1006, reason: "network failure", wasClean: false });
  assert.deepEqual(states.at(-1), {
    state: "disconnected",
    detail: { code: 1006, reason: "network failure" },
  });
  assert.deepEqual(events.at(-1).detail, {
    code: 1006,
    reason: "network failure",
    clean: false,
    correlation_id: browserCorrelationID,
  });
});

test("rejects sending before connection and surfaces invalid JSON", () => {
  FakeWebSocket.instances.length = 0;
  const first = createClient();

  assert.equal(first.client.send("before connect"), false);
  assert.equal(first.states.at(-1).state, "error");
  assert.equal(first.events.at(-1).type, "message.error");

  first.client.connect();
  const socket = FakeWebSocket.instances[0];
  socket.emit("open");
  socket.emit("message", { data: "not-json" });

  assert.equal(first.states.at(-1).state, "error");
  assert.equal(first.events.at(-1).type, "connection.error");
});

test("ignores events from a stale socket after a new connection starts", () => {
  FakeWebSocket.instances.length = 0;
  const { client, states } = createClient();

  client.connect();
  const staleSocket = FakeWebSocket.instances[0];
  staleSocket.emit("close", { code: 1000, reason: "closed", wasClean: true });

  client.connect();
  const currentSocket = FakeWebSocket.instances[1];
  staleSocket.emit("open");
  assert.equal(states.at(-1).state, "connecting");

  currentSocket.emit("open");
  assert.equal(states.at(-1).state, "connected");
});

test("disconnect requests a normal close without automatic reconnection", () => {
  FakeWebSocket.instances.length = 0;
  const { client, states } = createClient();

  client.connect();
  const socket = FakeWebSocket.instances[0];
  socket.emit("open");
  client.disconnect();

  assert.deepEqual(socket.closeArguments, {
    code: 1000,
    reason: "Client requested disconnect",
  });
  assert.equal(FakeWebSocket.instances.length, 1);
  socket.emit("close", { code: 1000, reason: "", wasClean: true });
  assert.equal(states.at(-1).state, "disconnected");
});
