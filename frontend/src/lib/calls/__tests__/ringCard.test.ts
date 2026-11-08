import { describe, expect, it } from "vitest";
import { MAX_CARDS, RING_TOTAL_MS, ringRemaining, ringSecondsLeft, shouldRing, visibleCards } from "../ringCard";

describe("ringRemaining / ringSecondsLeft", () => {
  it("começa cheio, esvazia em 45 s e nunca sai de 0 a 1", () => {
    expect(ringRemaining(1000, 1000)).toBe(1);
    expect(ringRemaining(1000, 1000 + RING_TOTAL_MS / 2)).toBeCloseTo(0.5);
    expect(ringRemaining(1000, 1000 + RING_TOTAL_MS)).toBe(0);
    expect(ringRemaining(1000, 1000 + 99_999)).toBe(0);
    expect(ringRemaining(1000, 500)).toBe(1);
  });
  it("sem instante de início não inventa prazo", () => {
    expect(ringRemaining(undefined, 123456)).toBe(1);
  });
  it("segundos restantes arredondam para cima", () => {
    expect(ringSecondsLeft(0, 0)).toBe(45);
    expect(ringSecondsLeft(0, 1)).toBe(45);
    expect(ringSecondsLeft(0, 44_001)).toBe(1);
    expect(ringSecondsLeft(0, 45_000)).toBe(0);
  });
});

describe("visibleCards", () => {
  const c = (id: string, ringingSince?: number) => ({ id, ringingSince });
  it(`mostra no máximo ${MAX_CARDS}, a mais antiga primeiro, e conta o resto`, () => {
    const r = visibleCards([c("c", 30), c("a", 10), c("b", 20)]);
    expect(r.shown.map((x) => x.id)).toEqual(["a", "b"]);
    expect(r.hidden).toBe(1);
  });
  it("com poucas chamadas não esconde nenhuma", () => {
    expect(visibleCards([c("a", 1)])).toEqual({ shown: [c("a", 1)], hidden: 0 });
    expect(visibleCards([])).toEqual({ shown: [], hidden: 0 });
  });
});

describe("shouldRing", () => {
  it("toca enquanto houver oferta não silenciada", () => {
    expect(shouldRing(["a"], new Set(), false)).toBe(true);
    expect(shouldRing(["a", "b"], new Set(["a"]), false)).toBe(true);
  });
  it("silencia quando todas foram ignoradas, quando não há oferta ou quando o operador já está em chamada", () => {
    expect(shouldRing(["a"], new Set(["a"]), false)).toBe(false);
    expect(shouldRing([], new Set(), false)).toBe(false);
    expect(shouldRing(["a"], new Set(), true)).toBe(false);
  });
});
