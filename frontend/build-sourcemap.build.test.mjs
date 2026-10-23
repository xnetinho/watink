import { describe, expect, it } from "vitest";
// O elo que a função sozinha não cobre: o vite.config.mjs usa mesmo a função. Roda o build de verdade (poucos
// segundos) numa pasta temporária e confere o que ele PUBLICARIA: nenhum .map e nenhuma referência a mapa nos .js.
import { execFileSync } from "node:child_process";
import { mkdtempSync, readdirSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

function listFiles(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap((e) =>
    e.isDirectory() ? listFiles(join(dir, e.name)) : [join(dir, e.name)]
  );
}

function buildTo(env) {
  const out = mkdtempSync(join(tmpdir(), "watink-build-"));
  try {
    execFileSync("npx", ["vite", "build", "--outDir", out, "--emptyOutDir"], {
      cwd: process.cwd(),
      env: { ...process.env, ...env },
      stdio: "pipe",
      timeout: 180_000,
    });
    return listFiles(out).map((f) => f.slice(out.length));
  } finally {
    rmSync(out, { recursive: true, force: true });
  }
}

describe("build de produção", () => {
  it("por padrão não publica nenhum mapa de fonte", () => {
    const files = buildTo({ VITE_SOURCEMAP: "" });
    expect(files.filter((f) => f.endsWith(".map"))).toEqual([]);
  }, 200_000);

  it("os .js não apontam para um mapa que não existe", () => {
    const out = mkdtempSync(join(tmpdir(), "watink-build-"));
    try {
      execFileSync("npx", ["vite", "build", "--outDir", out, "--emptyOutDir"], {
        cwd: process.cwd(),
        env: { ...process.env, VITE_SOURCEMAP: "" },
        stdio: "pipe",
        timeout: 180_000,
      });
      const refs = listFiles(out)
        .filter((f) => f.endsWith(".js"))
        .filter((f) => readFileSync(f, "utf8").includes("sourceMappingURL"));
      expect(refs).toEqual([]);
    } finally {
      rmSync(out, { recursive: true, force: true });
    }
  }, 200_000);

  it("VITE_SOURCEMAP=true gera os mapas (depuração pontual continua possível)", () => {
    const files = buildTo({ VITE_SOURCEMAP: "true" });
    expect(files.filter((f) => f.endsWith(".map")).length).toBeGreaterThan(0);
  }, 200_000);
});
