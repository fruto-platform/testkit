export function createCorrelationID({
  now = Date.now,
  getRandomValues = (target) => globalThis.crypto.getRandomValues(target),
} = {}) {
  const bytes = new Uint8Array(16);
  getRandomValues(bytes);
  const milliseconds = BigInt(now());
  for (let index = 5; index >= 0; index -= 1) {
    bytes[index] = Number((milliseconds >> BigInt((5 - index) * 8)) & 0xffn);
  }
  bytes[6] = (bytes[6] & 0x0f) | 0x70;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;

  const encoded = [...bytes].map((value) => value.toString(16).padStart(2, "0")).join("");
  return `${encoded.slice(0, 8)}-${encoded.slice(8, 12)}-${encoded.slice(12, 16)}-${encoded.slice(16, 20)}-${encoded.slice(20)}`;
}
