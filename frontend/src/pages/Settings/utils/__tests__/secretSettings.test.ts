import { describe, it, expect } from "vitest";
import { MASKED_SECRET, isMaskedSecret, shouldSaveSecret } from "../secretSettings";

describe("secretSettings", () => {
  it("reconhece o placeholder do backend", () => {
    expect(isMaskedSecret(MASKED_SECRET)).toBe(true);
    expect(isMaskedSecret("sk-real")).toBe(false);
    expect(isMaskedSecret("")).toBe(false);
  });

  it("nunca grava o placeholder por cima da chave real", () => {
    expect(shouldSaveSecret(MASKED_SECRET, MASKED_SECRET)).toBe(false);
    expect(shouldSaveSecret(MASKED_SECRET, "sk-real")).toBe(false);
  });

  it("não regrava quando o valor não mudou", () => {
    expect(shouldSaveSecret("sk-real", "sk-real")).toBe(false);
  });

  it("grava uma chave nova, ou a limpeza explícita do campo", () => {
    expect(shouldSaveSecret("sk-nova", "sk-real")).toBe(true);
    expect(shouldSaveSecret("sk-nova", "")).toBe(true);
    expect(shouldSaveSecret("", "sk-real")).toBe(true);
  });
});
