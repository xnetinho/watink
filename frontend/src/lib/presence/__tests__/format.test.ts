import { describe, expect, it } from "vitest";
import { presenceLabel } from "../format";

const now = new Date(2026, 9, 27, 15, 30);
const at = (y: number, m: number, d: number, h: number, mi: number) => Math.floor(new Date(y, m, d, h, mi).getTime() / 1000);

describe("presenceLabel", () => {
  it("sem informação não mostra nada", () => {
    expect(presenceLabel(undefined, now)).toBe("");
  });
  it("online", () => {
    expect(presenceLabel({ online: true }, now)).toBe("online");
  });
  it("offline sem última vez (contato escondeu) não mostra nada", () => {
    expect(presenceLabel({ online: false }, now)).toBe("");
  });
  it("hoje, ontem e data completa", () => {
    expect(presenceLabel({ online: false, lastSeen: at(2026, 9, 27, 14, 5) }, now)).toBe("visto por último hoje às 14:05");
    expect(presenceLabel({ online: false, lastSeen: at(2026, 9, 26, 23, 59) }, now)).toBe("visto por último ontem às 23:59");
    expect(presenceLabel({ online: false, lastSeen: at(2026, 9, 20, 9, 0) }, now)).toMatch(/^visto por último em .*20.*às 09:00$/);
  });
  it("virada de dia conta pelo calendário, não por 24h", () => {
    const late = new Date(2026, 9, 27, 0, 10);
    expect(presenceLabel({ online: false, lastSeen: at(2026, 9, 26, 23, 50) }, late)).toBe("visto por último ontem às 23:50");
  });
});
