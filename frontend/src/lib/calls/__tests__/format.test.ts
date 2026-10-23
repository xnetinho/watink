import { describe, expect, it } from "vitest";
import { displayName, elapsedSeconds, formatBitrate, formatDuration, levelPercent } from "../format";

describe("formatDuration", () => {
  it.each([
    [0, "00:00"],
    [5, "00:05"],
    [75, "01:15"],
    [3599, "59:59"],
    [3600, "1:00:00"],
    [3723, "1:02:03"],
    [-4, "00:00"],
    [12.9, "00:12"],
  ])("%s s -> %s", (s, want) => expect(formatDuration(s)).toBe(want));
});

describe("elapsedSeconds", () => {
  it("antes da mídia conectar o cronômetro não corre", () => {
    expect(elapsedSeconds(null, 1_000_000)).toBe(0);
  });
  it("conta desde a conexão real", () => {
    expect(elapsedSeconds(1_000_000, 1_075_400)).toBe(75);
  });
  it("relógio atrasado não gera tempo negativo", () => {
    expect(elapsedSeconds(2_000_000, 1_000_000)).toBe(0);
  });
});

describe("formatBitrate", () => {
  it("converte bytes/s em kbps com vírgula", () => {
    expect(formatBitrate(1550)).toBe("12,4 kbps");
    expect(formatBitrate(0)).toBe("0,0 kbps");
  });
  it("sem medida mostra traço", () => {
    expect(formatBitrate(undefined)).toBe("—");
    expect(formatBitrate(Number.NaN)).toBe("—");
  });
});

describe("levelPercent", () => {
  it("silêncio é 0 e satura em 100", () => {
    expect(levelPercent(0)).toBe(0);
    expect(levelPercent(undefined)).toBe(0);
    expect(levelPercent(1)).toBe(100);
  });
  it("é crescente", () => {
    expect(levelPercent(0.05)).toBeLessThan(levelPercent(0.2));
    expect(levelPercent(0.2)).toBeLessThan(levelPercent(0.6));
  });
});

describe("displayName", () => {
  it("prefere o nome, depois o número, depois o texto padrão", () => {
    expect(displayName({ name: " Maria ", number: "5511" }, "?")).toBe("Maria");
    expect(displayName({ name: "  ", number: "5511" }, "?")).toBe("5511");
    expect(displayName({}, "Sem nome")).toBe("Sem nome");
  });
});
