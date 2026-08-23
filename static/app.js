import { mountWebSocketClient } from "./ws-ui.js";
import { createTranslator, readPageTranslations } from "./i18n.js";

const page = document.body;
const translate = createTranslator(readPageTranslations(page));
const locale = page.dataset.locale || "en";

document.querySelectorAll("[data-ws-client]").forEach((client) => {
  mountWebSocketClient(client, { locale, translate });
});
