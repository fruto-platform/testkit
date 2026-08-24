const presets = {
  status: { query: "{ status version }" },
  echo: {
    query: "query Echo($message: String!) { echo(message: $message) }",
    variables: { message: "hello from GraphQL" },
  },
  invalid: { query: "query Invalid { missing }" },
};

function formatJSON(value) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

export function mountGraphQLLab(root, { translate = (key) => key, fetchImpl = globalThis.fetch } = {}) {
  const buttons = [...root.querySelectorAll("[data-graphql-preset]")];
  const requestLine = root.querySelector("[data-graphql-request-line]");
  const status = root.querySelector("[data-graphql-status]");
  const requestOutput = root.querySelector("[data-graphql-request]");
  const responseOutput = root.querySelector("[data-graphql-response]");
  const duration = root.querySelector("[data-graphql-duration]");
  const contentType = root.querySelector("[data-graphql-content-type]");
  const hint = root.querySelector("[data-graphql-hint]");

  async function run(name) {
    const preset = presets[name];
    if (!preset) return;

    for (const button of buttons) button.className = `preset-button${button.dataset.graphqlPreset === name ? " preset-button--active" : ""}`;
    const payload = { query: preset.query };
    if (preset.variables) payload.variables = preset.variables;
    requestLine.textContent = "POST /graphql";
    requestOutput.textContent = JSON.stringify(payload, null, 2);
    responseOutput.textContent = translate("lab.running");
    status.className = "lab-result-status lab-result-status--running";
    status.textContent = translate("lab.running");
    hint.textContent = translate("graphql.running");
    duration.textContent = "—";
    contentType.textContent = "—";

    const started = Date.now();
    try {
      const response = await fetchImpl("/graphql", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const body = await response.text();
      duration.textContent = `${Date.now() - started} ms`;
      contentType.textContent = response.headers.get("content-type") || translate("lab.not_available");
      responseOutput.textContent = formatJSON(body);
      status.className = `lab-result-status ${response.ok ? "lab-result-status--ok" : "lab-result-status--error"}`;
      status.textContent = `${response.status} ${response.statusText}`;
      hint.textContent = response.ok ? translate("graphql.completed") : translate("graphql.failed");
    } catch (error) {
      duration.textContent = `${Date.now() - started} ms`;
      responseOutput.textContent = String(error);
      status.className = "lab-result-status lab-result-status--error";
      status.textContent = translate("lab.network_error");
      hint.textContent = translate("graphql.failed");
    }
  }

  for (const button of buttons) button.addEventListener("click", () => run(button.dataset.graphqlPreset));
}

export { presets as graphQLPresets };
