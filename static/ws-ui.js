import { createWebSocketClient } from "./ws-client.js";

const stateLabels = {
  disconnected: "Disconnected",
  connecting: "Connecting",
  connected: "Connected",
  error: "Error",
};

const MAX_LOG_ENTRIES = 50;

export function mountWebSocketClient(root) {
  const connectButton = root.querySelector("[data-ws-connect]");
  const disconnectButton = root.querySelector("[data-ws-disconnect]");
  const form = root.querySelector("[data-ws-form]");
  const messageInput = root.querySelector("[data-ws-message]");
  const sendButton = root.querySelector("[data-ws-send]");
  const status = root.querySelector("[data-ws-status]");
  const statusText = root.querySelector("[data-ws-status-text]");
  const hint = root.querySelector("[data-ws-hint]");
  const log = root.querySelector("[data-ws-log]");
  const emptyLog = root.querySelector("[data-ws-empty]");
  const client = createWebSocketClient({
    url: root.dataset.wsUrl,
    onStateChange: ({ state, detail }) => updateState(state, detail),
    onEvent: (event) => appendEvent(event),
  });

  function updateState(state, detail = {}) {
    status.className = `connection-status connection-status--${state}`;
    statusText.textContent = stateLabels[state] || state;
    const connected = state === "connected";
    const connecting = state === "connecting";
    connectButton.disabled = connected || connecting;
    disconnectButton.disabled = !connected && !connecting;
    sendButton.disabled = !connected;
    messageInput.disabled = !connected;

    if (state === "connected") {
      hint.textContent = "Messages are sent as { message } and broadcast to every client.";
    } else if (state === "error") {
      hint.textContent = detail.message || "Check the event log and connect again.";
    } else if (state === "connecting") {
      hint.textContent = "Opening a same-origin WebSocket connection…";
    } else {
      hint.textContent = "Connect before sending a JSON message.";
    }
  }

  function appendEvent(event) {
    emptyLog.hidden = true;
    const entry = document.createElement("li");
    entry.className = `event-log__entry event-log__entry--${event.type.includes("error") ? "error" : "normal"}`;

    const meta = document.createElement("div");
    meta.className = "event-log__meta";
    const time = document.createElement("time");
    time.dateTime = event.timestamp.toISOString();
    time.textContent = event.timestamp.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" });
    const type = document.createElement("span");
    type.textContent = event.type;
    meta.append(time, type);

    const payload = document.createElement("code");
    payload.className = "event-log__payload";
    payload.textContent = JSON.stringify(event.detail);
    entry.append(meta, payload);
    log.append(entry);

    while (log.children.length > MAX_LOG_ENTRIES + 1) {
      log.removeChild(log.children[1]);
    }
    log.scrollTop = log.scrollHeight;
  }

  connectButton.addEventListener("click", () => client.connect());
  disconnectButton.addEventListener("click", () => client.disconnect());
  form.addEventListener("submit", (event) => {
    event.preventDefault();
    const message = messageInput.value.trim();
    if (!message) {
      hint.textContent = "Enter a message before sending.";
      messageInput.focus();
      return;
    }
    if (client.send(message)) {
      messageInput.select();
    }
  });

  updateState("disconnected");
}
