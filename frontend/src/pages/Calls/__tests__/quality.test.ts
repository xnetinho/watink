import { describe, expect, it } from "vitest";
import { formatMos, summaryLevel } from "../quality";

const s = (over = {}) => ({ rttMax: null, lossMax: null, jitterMax: null, mosEstimated: null, ...over });

describe("summaryLevel", () => {
  it("sem nenhuma medição não classifica (nunca 'boa' por falta de dado)", () => {
    expect(summaryLevel(s())).toBeNull();
  });
  it("tudo dentro dos limites é bom", () => {
    expect(summaryLevel(s({ rttMax: 120, lossMax: 1, jitterMax: 10 }))).toBe(3);
  });
  it.each([
    [{ rttMax: 200 }, 3],
    [{ rttMax: 201 }, 2],
    [{ rttMax: 400 }, 2],
    [{ rttMax: 401 }, 1],
    [{ lossMax: 2 }, 3],
    [{ lossMax: 2.1 }, 2],
    [{ lossMax: 5 }, 2],
    [{ lossMax: 5.1 }, 1],
    [{ jitterMax: 30 }, 3],
    [{ jitterMax: 30.5 }, 2],
    [{ jitterMax: 60 }, 2],
    [{ jitterMax: 61 }, 1],
  ])("limites de borda %j -> %s", (over, want) => {
    expect(summaryLevel(s(over))).toBe(want);
  });
  it("o pior fator manda", () => {
    expect(summaryLevel(s({ rttMax: 250, lossMax: 8, jitterMax: 5 }))).toBe(1);
  });
  it("RTT não medido não impede classificar por perda e jitter", () => {
    expect(summaryLevel(s({ lossMax: 6 }))).toBe(1);
  });
});

describe("formatMos", () => {
  it("vírgula decimal e traço quando ausente", () => {
    expect(formatMos(4.3)).toBe("4,3");
    expect(formatMos(null)).toBe("—");
  });
});
