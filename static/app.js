import { mountWebSocketClient } from "./ws-ui.js";

document.querySelectorAll("[data-ws-client]").forEach((client) => {
  mountWebSocketClient(client);
});
