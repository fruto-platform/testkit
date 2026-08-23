import assert from "node:assert/strict";
import { test } from "node:test";

const { createTranslator } = await import("../static/i18n.js");
const { formatRelativeTime } = await import("../static/ws-ui.js");

const catalogs = {
  en: {
    "time.day": "1 day ago",
    "time.days": "{count} days ago",
    "time.hour": "1 hour ago",
    "time.hours": "{count} hours ago",
    "time.less_than_minute": "less than a minute ago",
    "time.minute": "1 minute ago",
    "time.minutes": "{count} minutes ago",
    "websocket.sent": "Sent",
  },
  "pt-BR": {
    "time.day": "há 1 dia",
    "time.days": "há {count} dias",
    "time.hour": "há 1 hora",
    "time.hours": "há {count} horas",
    "time.less_than_minute": "há menos de um minuto",
    "time.minute": "há 1 minuto",
    "time.minutes": "há {count} minutos",
    "websocket.sent": "Enviada",
  },
  "es-AR": {
    "time.day": "hace 1 día",
    "time.days": "hace {count} días",
    "time.hour": "hace 1 hora",
    "time.hours": "hace {count} horas",
    "time.less_than_minute": "hace menos de un minuto",
    "time.minute": "hace 1 minuto",
    "time.minutes": "hace {count} minutos",
    "websocket.sent": "Enviado",
  },
};

test("interpolates translated UI labels and singular and plural relative time for every locale", () => {
  const now = new Date("2026-08-23T12:00:00Z");

  assert.deepEqual(
    Object.fromEntries(Object.entries(catalogs).map(([locale, catalog]) => {
      const translate = createTranslator(catalog);
      return [locale, {
        sent: translate("websocket.sent"),
        minute: formatRelativeTime(new Date("2026-08-23T11:59:00Z"), now, translate),
        hour: formatRelativeTime(new Date("2026-08-23T11:00:00Z"), now, translate),
        day: formatRelativeTime(new Date("2026-08-22T12:00:00Z"), now, translate),
        minutes: formatRelativeTime(new Date("2026-08-23T11:58:00Z"), now, translate),
      }];
    })),
    {
      en: { sent: "Sent", minute: "1 minute ago", hour: "1 hour ago", day: "1 day ago", minutes: "2 minutes ago" },
      "pt-BR": { sent: "Enviada", minute: "há 1 minuto", hour: "há 1 hora", day: "há 1 dia", minutes: "há 2 minutos" },
      "es-AR": { sent: "Enviado", minute: "hace 1 minuto", hour: "hace 1 hora", day: "hace 1 día", minutes: "hace 2 minutos" },
    },
  );
});
