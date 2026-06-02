#!/usr/bin/env node
// Verifies every locale defines exactly the same set of keys as English (the
// source of truth) for every namespace. Fails (exit 1) on any missing/extra key
// so translation drift is caught in CI. Run: `node scripts/check-locales.mjs`.
import { readFileSync, readdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const root = join(dirname(fileURLToPath(import.meta.url)), "..", "src", "locales");
const SOURCE = "en";

/** Flatten nested object keys into dotted paths: { a: { b: 1 } } -> ["a.b"]. */
function flatKeys(obj, prefix = "") {
  return Object.entries(obj).flatMap(([k, v]) => {
    const path = prefix ? `${prefix}.${k}` : k;
    return v && typeof v === "object" && !Array.isArray(v)
      ? flatKeys(v, path)
      : [path];
  });
}

function load(lang, ns) {
  return JSON.parse(readFileSync(join(root, lang, `${ns}.json`), "utf8"));
}

const namespaces = readdirSync(join(root, SOURCE)).map((f) => f.replace(/\.json$/, ""));
const langs = readdirSync(root).filter((l) => l !== SOURCE);

let problems = 0;
for (const ns of namespaces) {
  const want = new Set(flatKeys(load(SOURCE, ns)));
  for (const lang of langs) {
    let got;
    try {
      got = new Set(flatKeys(load(lang, ns)));
    } catch {
      console.error(`✗ ${lang}/${ns}.json — missing or invalid`);
      problems++;
      continue;
    }
    const missing = [...want].filter((k) => !got.has(k));
    const extra = [...got].filter((k) => !want.has(k));
    if (missing.length || extra.length) {
      problems++;
      if (missing.length) console.error(`✗ ${lang}/${ns}: missing ${missing.join(", ")}`);
      if (extra.length) console.error(`✗ ${lang}/${ns}: extra ${extra.join(", ")}`);
    }
  }
}

if (problems) {
  console.error(`\n${problems} locale issue(s) found.`);
  process.exit(1);
}
console.log(`✓ All locales match ${SOURCE} across ${namespaces.length} namespace(s).`);
