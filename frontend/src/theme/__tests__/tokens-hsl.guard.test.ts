import { describe, expect, it } from "vitest";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join, relative, sep } from "node:path";

// Os tokens de cor deste projeto são HSL CRU ("217 89% 76%"), por convenção (theme/README.md). Usar `var(--token)` direto
// como cor gera `color: 217 89% 76%`, que é INVÁLIDO: o navegador ignora e cai na cor herdada. Foi assim que o tique de
// "lido" ficou cinza. O uso correto é `hsl(var(--token))`.

const SRC = join(__dirname, "..", "..");

const rawTokens = (): Set<string> => {
  const css = readFileSync(join(SRC, "theme", "tokens", "colors.css"), "utf8");
  const raw = new Set<string>();
  for (const m of css.matchAll(/^\s*--([a-z0-9-]+)\s*:\s*(-?[\d.]+\s+[\d.]+%\s+[\d.]+%(?:\s*\/\s*[\d.%]+)?)\s*;/gim)) raw.add(m[1]);
  // bridge.css cria aliases (--primary: var(--action-primary)): herdam o formato cru.
  const bridge = readFileSync(join(SRC, "theme", "bridge.css"), "utf8");
  for (const m of bridge.matchAll(/^\s*--([a-z0-9-]+)\s*:\s*var\(--([a-z0-9-]+)\)\s*;/gim)) if (raw.has(m[2])) raw.add(m[1]);
  return raw;
};

const files = (dir: string, out: string[] = []): string[] => {
  for (const f of readdirSync(dir)) {
    const p = join(dir, f);
    if (statSync(p).isDirectory()) {
      if (f === "__tests__" || f === "node_modules") continue;
      files(p, out);
    } else if (/\.(tsx?|jsx?)$/.test(f) && !/\.test\./.test(f)) out.push(p);
  }
  return out;
};

// Usos de cor em classe Tailwind arbitrária: text-[var(--x)], bg-[var(--x)], border-l-[var(--x)], from-[...]...
const COLOR_CLASS = /\b(?:text|bg|border(?:-[lrtbxy])?|from|to|via|fill|stroke|ring|accent|divide|outline|caret|decoration|placeholder|shadow)-\[var\(--([a-z0-9-]+)\)\]/g;

describe("tokens de cor HSL cru", () => {
  const raw = rawTokens();

  it("colors.css expõe tokens HSL crus (a premissa desta guarda)", () => {
    expect(raw.has("action-primary")).toBe(true);
    expect(raw.size).toBeGreaterThan(20);
  });

  it("nenhuma classe Tailwind usa var(--token-hsl-cru) sem hsl()", () => {
    const offenders: string[] = [];
    for (const f of files(SRC)) {
      const lines = readFileSync(f, "utf8").split("\n");
      lines.forEach((line: string, i: number) => {
        for (const m of line.matchAll(COLOR_CLASS)) {
          if (raw.has(m[1])) offenders.push(`${relative(SRC, f).split(sep).join("/")}:${i + 1}  ${m[0]}  → use ${m[0].replace(/var\((--[a-z0-9-]+)\)/, "hsl(var($1))")}`);
        }
      });
    }
    expect(offenders, `cor inválida (token HSL cru sem hsl()):\n${offenders.join("\n")}`).toEqual([]);
  });
});
