import assert from "node:assert/strict";
import { test } from "node:test";

class FakeElement {
  constructor() {
    this.children = [];
    this.dataset = {};
    this.listeners = new Map();
    this.className = "";
    this.textContent = "";
    this.hidden = false;
    this.disabled = false;
    this.value = "";
    this.scrollTop = 0;
    this.scrollHeight = 0;
    this.attributes = new Map();
  }

  addEventListener(type, listener) {
    this.listeners.set(type, listener);
  }

  dispatch(type, event = {}) {
    this.listeners.get(type)?.({ currentTarget: this, preventDefault() {}, ...event });
  }

  append(...children) {
    this.children.push(...children);
  }

  removeChild(child) {
    const index = this.children.indexOf(child);
    if (index >= 0) this.children.splice(index, 1);
  }

  querySelector(selector) {
    return this.controls.get(selector);
  }

  querySelectorAll(selector) {
    return this.collections.get(selector) || [];
  }
}

globalThis.document = {
  createElement: () => new FakeElement(),
};

const { mountRestLab, restPresets } = await import("../static/rest-ui.js");
const { mountGraphQLLab, graphQLPresets } = await import("../static/graphql-ui.js");
const { mountSSELab } = await import("../static/sse-ui.js");

function rootFor(selectors, collections = {}) {
  const root = new FakeElement();
  root.controls = new Map(Object.entries(selectors));
  root.collections = new Map(Object.entries(collections));
  return root;
}

function labTranslator(key) {
  return key;
}

function createHTTPRoot(prefix, names = ["status", "items", "echo", "invalid"]) {
  const buttons = names.map((name) => {
    const button = new FakeElement();
    button.dataset[`${prefix}Preset`] = name;
    return button;
  });
  const selectors = {
    [`[data-${prefix}-request-line]`]: new FakeElement(),
    [`[data-${prefix}-status]`]: new FakeElement(),
    [`[data-${prefix}-request]`]: new FakeElement(),
    [`[data-${prefix}-response]`]: new FakeElement(),
    [`[data-${prefix}-duration]`]: new FakeElement(),
    [`[data-${prefix}-content-type]`]: new FakeElement(),
    [`[data-${prefix}-hint]`]: new FakeElement(),
  };
  return rootFor(selectors, { [`[data-${prefix}-preset]`]: buttons });
}

test("REST presets send the existing contracts and render responses", async () => {
  const root = createHTTPRoot("rest");
  const calls = [];
  const fetchImpl = async (path, options) => {
    calls.push({ path, options });
    return {
      ok: true,
      status: 200,
      statusText: "OK",
      headers: { get: () => "application/json" },
      text: async () => '{"status":"ok","version":"dev"}',
    };
  };

  mountRestLab(root, { fetchImpl, translate: labTranslator });
  root.collections.get("[data-rest-preset]")[2].dispatch("click");
  await new Promise((resolve) => setImmediate(resolve));

  assert.equal(restPresets.echo.method, "POST");
  assert.equal(calls[0].path, "/api/echo");
  assert.deepEqual(JSON.parse(calls[0].options.body), { hello: "world" });
  assert.equal(root.controls.get("[data-rest-status]").textContent, "200 OK");
  assert.match(root.controls.get("[data-rest-response]").textContent, /"status": "ok"/);
});

test("REST invalid JSON preset surfaces the existing error response", async () => {
  const root = createHTTPRoot("rest");
  const fetchImpl = async (_path, options) => {
    assert.equal(options.body, '{"hello":');
    return {
      ok: false,
      status: 400,
      statusText: "Bad Request",
      headers: { get: () => "text/plain; charset=utf-8" },
      text: async () => "invalid JSON request",
    };
  };

  mountRestLab(root, { fetchImpl, translate: labTranslator });
  root.collections.get("[data-rest-preset]")[3].dispatch("click");
  await new Promise((resolve) => setImmediate(resolve));

  assert.equal(root.controls.get("[data-rest-status]").textContent, "400 Bad Request");
  assert.equal(root.controls.get("[data-rest-response]").textContent, "invalid JSON request");
});

test("GraphQL presets preserve query variables and expose GraphQL errors", async () => {
  const root = createHTTPRoot("graphql", ["status", "echo", "invalid"]);
  const calls = [];
  const fetchImpl = async (_path, options) => {
    calls.push(options);
    return {
      ok: true,
      status: 200,
      statusText: "OK",
      headers: { get: () => "application/json" },
      text: async () => '{"errors":[{"message":"Cannot query field \\"missing\\""}]}',
    };
  };

  mountGraphQLLab(root, { fetchImpl, translate: labTranslator });
  root.collections.get("[data-graphql-preset]")[1].dispatch("click");
  await new Promise((resolve) => setImmediate(resolve));

  const payload = JSON.parse(calls[0].body);
  assert.equal(graphQLPresets.echo.variables.message, "hello from GraphQL");
  assert.match(payload.query, /Echo/);
  assert.deepEqual(payload.variables, { message: "hello from GraphQL" });

  root.collections.get("[data-graphql-preset]")[2].dispatch("click");
  await new Promise((resolve) => setImmediate(resolve));
  assert.match(root.controls.get("[data-graphql-response]").textContent, /errors/);
});

class FakeEventSource {
  static instances = [];

  constructor(url) {
    this.url = url;
    this.listeners = new Map();
    this.closed = false;
    FakeEventSource.instances.push(this);
  }

  addEventListener(type, listener) {
    this.listeners.set(type, listener);
  }

  emit(type, event = {}) {
    if (type === "status") this.listeners.get("status")?.(event);
    else this[`on${type}`]?.(event);
  }

  close() {
    this.closed = true;
  }
}

test("SSE connects, renders named events and only reconnects explicitly", () => {
  FakeEventSource.instances.length = 0;
  const connect = new FakeElement();
  const disconnect = new FakeElement();
  const root = rootFor({
    "[data-sse-connect]": connect,
    "[data-sse-disconnect]": disconnect,
    "[data-sse-status]": new FakeElement(),
    "[data-sse-status-text]": new FakeElement(),
    "[data-sse-hint]": new FakeElement(),
    "[data-sse-events]": new FakeElement(),
    "[data-sse-empty]": new FakeElement(),
    "[data-sse-count]": new FakeElement(),
  });
  root.dataset.sseUrl = "/events";
  root.controls.get("[data-sse-events]").append(root.controls.get("[data-sse-empty]"));

  mountSSELab(root, { EventSourceImpl: FakeEventSource, translate: labTranslator });
  connect.dispatch("click");
  const source = FakeEventSource.instances[0];
  assert.equal(source.url, "/events");
  source.emit("open");
  source.emit("status", { lastEventId: "7", type: "status", data: '{"status":"ok"}' });

  assert.equal(root.controls.get("[data-sse-status-text]").textContent, "sse.connected");
  assert.equal(root.controls.get("[data-sse-count]").textContent, "1");
  assert.match(root.controls.get("[data-sse-events]").children[1].children[1].textContent, /"status": "ok"/);

  source.emit("error");
  assert.equal(source.closed, true);
  assert.equal(FakeEventSource.instances.length, 1);
  connect.dispatch("click");
  assert.equal(FakeEventSource.instances.length, 2);
});
