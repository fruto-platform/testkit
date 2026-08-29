import assert from "node:assert/strict";
import { test } from "node:test";

import { createCorrelationID } from "../static/correlation-id.js";

test("creates a lowercase UUIDv7 carrying the supplied Unix millisecond", () => {
  const now = 1_717_171_717_171;
  const random = Uint8Array.from({ length: 16 }, (_, index) => index);
  const correlationID = createCorrelationID({
    now: () => now,
    getRandomValues: (target) => target.set(random),
  });

  assert.match(correlationID, /^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  assert.equal(Number.parseInt(correlationID.slice(0, 8) + correlationID.slice(9, 13), 16), now);
});
