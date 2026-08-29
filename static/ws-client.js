import { createCorrelationID } from "./correlation-id.js";

const OPEN = 1;

export function createWebSocketClient({ url, onEvent, onStateChange, createCorrelationIDImpl = createCorrelationID }) {
  let socket = null;

  function emit(type, detail = {}) {
    onEvent({ type, detail, timestamp: new Date() });
  }

  function setState(state, detail = {}) {
    onStateChange({ state, detail });
  }

  function connect() {
    if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === OPEN)) {
      return;
    }

    const endpoint = new URL(url, window.location.href);
    endpoint.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const correlationID = createCorrelationIDImpl();
    endpoint.searchParams.set("correlation_id", correlationID);
    setState("connecting");
    emit("connection.connecting", { url: endpoint.toString(), correlation_id: correlationID });

    const connection = new WebSocket(endpoint.toString());
    socket = connection;

    connection.addEventListener("open", () => {
      if (socket !== connection) return;
      setState("connected");
      emit("connection.open", { url: endpoint.toString(), correlation_id: correlationID });
    });

    connection.addEventListener("message", (event) => {
      if (socket !== connection) return;
      try {
        const payload = JSON.parse(event.data);
        emit("broadcast.received", payload);
      } catch (error) {
        setState("error", { message: "The server sent invalid JSON." });
        emit("connection.error", { message: "Invalid JSON received", error: String(error) });
      }
    });

    connection.addEventListener("error", () => {
      if (socket !== connection) return;
      setState("error", { message: "The WebSocket reported an error." });
      emit("connection.error", { message: "The WebSocket reported an error." });
    });

    connection.addEventListener("close", (event) => {
      if (socket === connection) {
        socket = null;
        setState("disconnected", { code: event.code, reason: event.reason });
      }
      emit("connection.closed", {
        code: event.code,
        reason: event.reason || "No reason provided",
        clean: event.wasClean,
        correlation_id: correlationID,
      });
    });
  }

  function disconnect() {
    if (!socket) {
      setState("disconnected");
      return;
    }
    emit("connection.disconnecting", { reason: "Client requested disconnect" });
    socket.close(1000, "Client requested disconnect");
  }

  function send(message) {
    if (!socket || socket.readyState !== OPEN) {
      setState("error", { message: "Connect this client before sending." });
      emit("message.error", { message: "Connect this client before sending." });
      return false;
    }

    const payload = { message };
    socket.send(JSON.stringify(payload));
    emit("message.sent", payload);
    return true;
  }

  return { connect, disconnect, send };
}
