import { describe, expect, it } from "vitest";
import { presenceLabel } from "../format";
import { i18n } from "../../../translate/i18n";

const now = new Date(2026, 9, 27, 15, 30);
const at = (y: number, m: number, d: number, h: number, mi: number) => Math.floor(new Date(y, m, d, h, mi).getTime() / 1000);
const tIn = (lng: string) => (k: string, o?: Record<string, string | number>) => String(i18n.t(k, { lng, ...(o ?? {}) }) ?? "");

describe("presenceLabel", () => {
  const t = tIn("pt");
  it("sem informação não mostra nada", () => {
    expect(presenceLabel(undefined, t, "pt-BR", now)).toBe("");
  });
  it("online", () => {
    expect(presenceLabel({ online: true }, t, "pt-BR", now)).toBe("online");
  });
  it("offline sem última vez (contato escondeu) não mostra nada", () => {
    expect(presenceLabel({ online: false }, t, "pt-BR", now)).toBe("");
  });
  it("hoje, ontem e data completa", () => {
    expect(presenceLabel({ online: false, lastSeen: at(2026, 9, 27, 14, 5) }, t, "pt-BR", now)).toBe("visto por último hoje às 14:05");
    expect(presenceLabel({ online: false, lastSeen: at(2026, 9, 26, 23, 59) }, t, "pt-BR", now)).toBe("visto por último ontem às 23:59");
    expect(presenceLabel({ online: false, lastSeen: at(2026, 9, 20, 9, 0) }, t, "pt-BR", now)).toMatch(/^visto por último em .*20.*às 09:00$/);
  });
  it("virada de dia conta pelo calendário, não por 24h", () => {
    const late = new Date(2026, 9, 27, 0, 10);
    expect(presenceLabel({ online: false, lastSeen: at(2026, 9, 26, 23, 50) }, t, "pt-BR", late)).toBe("visto por último ontem às 23:50");
  });
  it("o texto vem do arquivo de idioma: en e es traduzem", () => {
    const seen = { online: false, lastSeen: at(2026, 9, 27, 14, 5) };
    expect(presenceLabel(seen, tIn("en"), "en-US", now)).toMatch(/^last seen today at /);
    expect(presenceLabel(seen, tIn("es"), "es-ES", now)).toMatch(/^últ\. vez hoy a las /);
    expect(presenceLabel({ online: true }, tIn("es"), "es-ES", now)).toBe("en línea");
  });
});
