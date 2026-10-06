import { describe, expect, it } from "vitest";
import { connectionHasProxy, placeBlock } from "../canPlace";

const ok = { status: "CONNECTED", proxyMode: "none" };

describe("connectionHasProxy", () => {
  it("sem conexão ou sem proxy não conta", () => {
    expect(connectionHasProxy(undefined)).toBe(false);
    expect(connectionHasProxy({ proxyMode: "none" })).toBe(false);
    expect(connectionHasProxy({})).toBe(false);
  });
  it("qualquer modo, id ou grupo de proxy conta (inclui grupo sem pick atual: fail-closed)", () => {
    expect(connectionHasProxy({ proxyMode: "single" })).toBe(true);
    expect(connectionHasProxy({ proxyMode: "group" })).toBe(true);
    expect(connectionHasProxy({ proxyMode: "none", proxyId: 3 })).toBe(true);
    expect(connectionHasProxy({ proxyMode: "none", proxyGroupId: 2 })).toBe(true);
  });
});

describe("placeBlock", () => {
  it("conexão conectada, sem proxy, ticket individual: pode ligar", () => {
    expect(placeBlock({}, ok, false)).toBeNull();
  });
  it.each([
    [{ isGroup: true }],
    [{ isCommunity: true }],
    [{ isSubGroup: true }],
  ])("grupo/comunidade/canal %j não é individual", (t) => {
    expect(placeBlock(t, ok, false)).toBe("not_individual");
  });
  it("conexão desconectada ou inexistente", () => {
    expect(placeBlock({}, { status: "DISCONNECTED" }, false)).toBe("disconnected");
    expect(placeBlock({}, undefined, false)).toBe("disconnected");
  });
  it("conexão com proxy", () => {
    expect(placeBlock({}, { status: "CONNECTED", proxyMode: "single" }, false)).toBe("proxy");
  });
  it("operador já em chamada", () => {
    expect(placeBlock({}, ok, true)).toBe("busy");
  });
  it("a ordem prioriza o motivo mais útil: tipo > ocupado > desconectada > proxy", () => {
    expect(placeBlock({ isGroup: true }, undefined, true)).toBe("not_individual");
    expect(placeBlock({}, undefined, true)).toBe("busy");
    expect(placeBlock({}, { status: "DISCONNECTED", proxyMode: "single" }, false)).toBe("disconnected");
  });
});
