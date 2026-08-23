export function createTranslator(catalog) {
  return (key, values = {}) => {
    const template = catalog[key] || key;
    return template.replace(/\{(\w+)\}/g, (_, name) => String(values[name] ?? `{${name}}`));
  };
}

export function readPageTranslations(element = document.body) {
  try {
    return JSON.parse(element.dataset.translations || "{}");
  } catch {
    return {};
  }
}
