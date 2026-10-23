import { describe, expect, it } from "vitest";
import { buildSaveBody, needsAck, normalizeMode, saveState } from "../callRecording";

describe("normalizeMode", () => {
  it("só aceita optional e auto; o resto é off", () => {
    expect(normalizeMode("optional")).toBe("optional");
    expect(normalizeMode("auto")).toBe("auto");
    expect(normalizeMode("off")).toBe("off");
    for (const v of [undefined, null, "", "on", "true", "AUTO", 1]) expect(normalizeMode(v)).toBe("off");
  });
});

describe("needsAck", () => {
  it.each([
    ["off", "optional", true],
    ["off", "auto", true],
    ["off", "off", false],
    ["optional", "auto", false],
    ["auto", "optional", false],
    ["optional", "off", false],
    ["auto", "off", false],
  ] as const)("%s -> %s exige aceite? %s", (a, b, want) => expect(needsAck(a, b)).toBe(want));
});

describe("saveState", () => {
  it("sem mudança não há o que salvar", () => {
    expect(saveState("off", "off", false, true).canSave).toBe(false);
    expect(saveState("auto", "auto", true, true).canSave).toBe(false);
  });
  it("sair de off mostra o termo e só habilita Salvar depois do aceite", () => {
    expect(saveState("off", "auto", false, true)).toEqual({ canSave: false, showAck: true });
    expect(saveState("off", "auto", true, true)).toEqual({ canSave: true, showAck: true });
  });
  it("entre modos ligados e voltando para off não pede aceite", () => {
    expect(saveState("optional", "auto", false, true)).toEqual({ canSave: true, showAck: false });
    expect(saveState("auto", "off", false, true)).toEqual({ canSave: true, showAck: false });
  });
  it("sem armazenamento só 'off' pode ser salvo", () => {
    expect(saveState("off", "auto", true, false).canSave).toBe(false);
    expect(saveState("auto", "off", false, false).canSave).toBe(true);
  });
});

describe("buildSaveBody", () => {
  it("o aceite só vai quando é exigido e foi dado", () => {
    expect(buildSaveBody("off", "auto", true)).toEqual({ mode: "auto", ack: true });
    expect(buildSaveBody("off", "auto", false)).toEqual({ mode: "auto" });
    expect(buildSaveBody("optional", "auto", true)).toEqual({ mode: "auto" });
    expect(buildSaveBody("auto", "off", true)).toEqual({ mode: "off" });
  });
});
