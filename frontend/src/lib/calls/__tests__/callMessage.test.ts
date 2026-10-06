import { describe, expect, it } from "vitest";
import { callDurationLabel, callTitleKey, callTone, parseCallData, recordingView } from "../callMessage";

const base = { callId: "C1", direction: "incoming", status: "ended", durationSec: 75 };

describe("parseCallData", () => {
  it("lê string JSON e objeto", () => {
    expect(parseCallData(JSON.stringify(base))?.callId).toBe("C1");
    expect(parseCallData(base)?.durationSec).toBe(75);
  });
  it("campos opcionais ausentes não quebram", () => {
    const d = parseCallData({ callId: "C1" });
    expect(d).toMatchObject({ direction: "incoming", status: "ended" });
    expect(d?.handledByName).toBeUndefined();
  });
  it("rejeita o que não é uma chamada", () => {
    expect(parseCallData(null)).toBeNull();
    expect(parseCallData("não é json")).toBeNull();
    expect(parseCallData("{}")).toBeNull();
    expect(parseCallData({ callId: "" })).toBeNull();
    expect(parseCallData(42)).toBeNull();
  });
  it("ignora tipos errados em vez de propagá-los", () => {
    const d = parseCallData({ callId: "C", durationSec: "75", handledByName: 9, mosEstimated: "x" });
    expect(d?.durationSec).toBeUndefined();
    expect(d?.handledByName).toBeUndefined();
    expect(d?.mosEstimated).toBeNull();
  });
});

describe("callTitleKey", () => {
  it.each([
    [{ status: "ended", direction: "incoming" }, "calls.message.received"],
    [{ status: "ended", direction: "outgoing" }, "calls.message.made"],
    [{ status: "missed", direction: "incoming" }, "calls.message.missed"],
    [{ status: "rejected", direction: "incoming" }, "calls.message.rejected"],
    [{ status: "failed", direction: "outgoing" }, "calls.message.interrupted"],
    [{ status: "interrupted", direction: "incoming" }, "calls.message.interrupted"],
  ] as const)("%j -> %s", (d, key) => expect(callTitleKey(d)).toBe(key));
});

describe("callTone", () => {
  it("atendida é sucesso, perdida alerta, recusada/falha erro", () => {
    expect(callTone("ended")).toBe("success");
    expect(callTone("missed")).toBe("warning");
    expect(callTone("rejected")).toBe("error");
    expect(callTone("failed")).toBe("error");
    expect(callTone("interrupted")).toBe("error");
    expect(callTone("desconhecido")).toBe("default");
  });
});

describe("callDurationLabel", () => {
  it("mostra a duração de chamada atendida", () => {
    expect(callDurationLabel({ status: "ended", durationSec: 75 })).toBe("01:15");
  });
  it("perdida ou recusada não tem duração a mostrar", () => {
    expect(callDurationLabel({ status: "missed", durationSec: 0 })).toBeNull();
    expect(callDurationLabel({ status: "rejected", durationSec: 3 })).toBeNull();
  });
  it("sem duração informada não mostra nada", () => {
    expect(callDurationLabel({ status: "ended" })).toBeNull();
  });
});

describe("recordingView", () => {
  it.each([
    ["ready", "ready"],
    ["failed", "failed"],
    ["deleted", "deleted"],
    ["recording", "recording"],
    [undefined, "none"],
    ["", "none"],
    ["qualquer", "none"],
  ])("%s -> %s", (s, v) => expect(recordingView({ recordingStatus: s })).toBe(v));
});
