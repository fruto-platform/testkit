const presets = {
  status: { method: "GET", path: "/api/status" },
  items: { method: "GET", path: "/api/items" },
  echo: { method: "POST", path: "/api/echo", body: JSON.stringify({ hello: "world" }) },
  invalid: { method: "POST", path: "/api/echo", body: '{"hello":' },
};

function formatJSON(value) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

export function mountRestLab(root, { translate = (key) => key, fetchImpl = globalThis.fetch } = {}) {
  const buttons = [...root.querySelectorAll("[data-rest-preset]")];
  const requestLine = root.querySelector("[data-rest-request-line]");
  const status = root.querySelector("[data-rest-status]");
  const requestOutput = root.querySelector("[data-rest-request]");
  const responseOutput = root.querySelector("[data-rest-response]");
  const duration = root.querySelector("[data-rest-duration]");
  const contentType = root.querySelector("[data-rest-content-type]");
  const hint = root.querySelector("[data-rest-hint]");

  async function run(name) {
    const preset = presets[name];
    if (!preset) return;

    for (const button of buttons) button.className = `preset-button${button.dataset.restPreset === name ? " preset-button--active" : ""}`;
    const request = { method: preset.method, path: preset.path };
    if (preset.body) request.body = preset.body;
    requestLine.textContent = `${preset.method} ${preset.path}`;
    requestOutput.textContent = JSON.stringify(request, null, 2);
    responseOutput.textContent = translate("lab.running");
    status.className = "lab-result-status lab-result-status--running";
    status.textContent = translate("lab.running");
    hint.textContent = translate("rest.running");
    duration.textContent = "—";
    contentType.textContent = "—";

    const started = Date.now();
    try {
      const response = await fetchImpl(preset.path, {
        method: preset.method,
        headers: preset.body ? { "Content-Type": "application/json" } : undefined,
        body: preset.body,
      });
      const body = await response.text();
      duration.textContent = `${Date.now() - started} ms`;
      contentType.textContent = response.headers.get("content-type") || translate("lab.not_available");
      responseOutput.textContent = formatJSON(body);
      status.className = `lab-result-status ${response.ok ? "lab-result-status--ok" : "lab-result-status--error"}`;
      status.textContent = `${response.status} ${response.statusText}`;
      hint.textContent = response.ok ? translate("rest.completed") : translate("rest.failed");
    } catch (error) {
      duration.textContent = `${Date.now() - started} ms`;
      responseOutput.textContent = String(error);
      status.className = "lab-result-status lab-result-status--error";
      status.textContent = translate("lab.network_error");
      hint.textContent = translate("rest.failed");
    }
  }

  for (const button of buttons) button.addEventListener("click", () => run(button.dataset.restPreset));
}

export { presets as restPresets };
