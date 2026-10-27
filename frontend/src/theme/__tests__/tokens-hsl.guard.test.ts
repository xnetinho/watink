import { describe, expect, it } from "vitest";

// Os tokens de cor deste projeto são HSL CRU ("217 89% 76%"), por convenção (theme/README.md). Usar `var(--token)` direto
// como cor gera `color: 217 89% 76%`, que é INVÁLIDO: o navegador ignora e cai na cor herdada. Foi assim que o tique de
// "lido" ficou cinza. O uso correto é `hsl(var(--token))`.

// Todo o código-fonte (menos testes), lido como texto pelo Vite: não precisa de fs/path, então tipa sem @types/node.
const sources = import.meta.glob(["../../**/*.{ts,tsx}", "!../../**/__tests__/**", "!../../**/*.test.{ts,tsx}"], {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

// O vitest passa .css pelo Tailwind e devolve vazio para `?raw`/`?inline`, então o CSS é lido do disco. Declaração local
// mínima (em vez de @types/node, que o projeto não instala) para o typecheck do app continuar estrito.
declare const process: { cwd(): string };
const readText = async (rel: string): Promise<string> => {
  const mod = "node:fs"; // por variável: o TS não resolve o módulo (sem @types/node) e o Vite o carrega em runtime
  const fs = (await import(/* @vite-ignore */ mod)) as { readFileSync(p: string, enc: string): string };
  return fs.readFileSync(`${process.cwd()}/${rel}`, "utf8");
};

const rawTokens = async (): Promise<Set<string>> => {
  const colorsCss = await readText("src/theme/tokens/colors.css");
  const bridgeCss = await readText("src/theme/bridge.css");
  const raw = new Set<string>();
  for (const m of colorsCss.matchAll(/^\s*--([a-z0-9-]+)\s*:\s*(-?[\d.]+\s+[\d.]+%\s+[\d.]+%(?:\s*\/\s*[\d.%]+)?)\s*;/gim)) raw.add(m[1]);
  // bridge.css cria aliases (--primary: var(--action-primary)): herdam o formato cru.
  for (const m of bridgeCss.matchAll(/^\s*--([a-z0-9-]+)\s*:\s*var\(--([a-z0-9-]+)\)\s*;/gim)) if (raw.has(m[2])) raw.add(m[1]);
  return raw;
};

// Cor em classe Tailwind arbitrária: text-[var(--x)], bg-[var(--x)], border-l-[var(--x)], from-[...]...
const COLOR_CLASS = /\b(?:text|bg|border(?:-[lrtbxy])?|from|to|via|fill|stroke|ring|accent|divide|outline|caret|decoration|placeholder|shadow)-\[var\(--([a-z0-9-]+)\)\]/g;

describe("tokens de cor HSL cru", () => {
  it("colors.css expõe tokens HSL crus (a premissa desta guarda)", async () => {
    const raw = await rawTokens();
    expect(raw.has("action-primary")).toBe(true);
    expect(raw.size).toBeGreaterThan(20);
  });

  it("a guarda enxerga o código-fonte (senão passaria vazia)", () => {
    expect(Object.keys(sources).length).toBeGreaterThan(100);
  });

  it("nenhuma classe Tailwind usa var(--token-hsl-cru) sem hsl()", async () => {
    const raw = await rawTokens();
    const offenders: string[] = [];
    for (const [file, text] of Object.entries(sources)) {
      text.split("\n").forEach((line: string, i: number) => {
        for (const m of line.matchAll(COLOR_CLASS)) {
          if (raw.has(m[1])) offenders.push(`${file.replace(/^(\.\.\/)+/, "")}:${i + 1}  ${m[0]}  → use ${m[0].replace(/var\((--[a-z0-9-]+)\)/, "hsl(var($1))")}`);
        }
      });
    }
    expect(offenders, `cor inválida (token HSL cru sem hsl()):\n${offenders.join("\n")}`).toEqual([]);
  });
});
