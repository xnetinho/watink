import { describe, expect, it } from "vitest";
import { buildSourcemap } from "../build-sourcemap.mjs";

// O build de produção publica os arquivos do `build/` dentro da imagem do business, e o business serve tudo de
// `/assets/` sem filtro. Mapa de fonte publicado = qualquer visitante lê o TypeScript original no F12. Por isso o
// padrão é DESLIGADO; só um build pontual de depuração liga, de forma explícita.
describe("buildSourcemap", () => {
  it("padrão (sem variável) é desligado", () => {
    expect(buildSourcemap({})).toBe(false);
  });

  it("VITE_SOURCEMAP=true liga (build pontual de depuração)", () => {
    expect(buildSourcemap({ VITE_SOURCEMAP: "true" })).toBe(true);
    expect(buildSourcemap({ VITE_SOURCEMAP: "TRUE" })).toBe(true);
    expect(buildSourcemap({ VITE_SOURCEMAP: " true " })).toBe(true);
  });

  it("qualquer outro valor continua desligado (typo não pode publicar o código)", () => {
    for (const v of ["", "false", "0", "1", "yes", "on", "ture", "sourcemap"]) {
      expect(buildSourcemap({ VITE_SOURCEMAP: v }), `valor ${JSON.stringify(v)}`).toBe(false);
    }
  });

  it("undefined é desligado", () => {
    expect(buildSourcemap({ VITE_SOURCEMAP: undefined })).toBe(false);
  });
});
