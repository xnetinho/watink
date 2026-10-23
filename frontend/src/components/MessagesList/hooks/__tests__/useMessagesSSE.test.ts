import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const put = vi.fn().mockResolvedValue({});
vi.mock("../../../../services/api", () => ({ default: { put: (...a: unknown[]) => put(...a), get: vi.fn() } }));

let handler: ((d: { action: string; message: Record<string, unknown> }) => void) | null = null;
vi.mock("../../../../services/sse-client", () => ({
  subscribeToSocket: (handlers: { appMessage: typeof handler }) => {
    handler = handlers.appMessage;
    return () => undefined;
  },
}));

import { useMessagesSocket } from "../useMessagesSSE";

const setVisibility = (v: "visible" | "hidden") =>
  Object.defineProperty(document, "visibilityState", { value: v, configurable: true });

describe("useMessagesSocket — zerar não lidas da conversa aberta", () => {
  beforeEach(() => {
    put.mockClear();
    handler = null;
    setVisibility("visible");
  });

  const mount = () => renderHook(() => useMessagesSocket(7, vi.fn(), { current: null }));

  it("mensagem RECEBIDA na conversa aberta e visível zera o contador", () => {
    mount();
    handler?.({ action: "create", message: { id: "a", ticketId: 7, fromMe: false } });
    expect(put).toHaveBeenCalledWith("/tickets/7", { unreadMessages: 0 });
  });

  it("mensagem que eu enviei não zera nada (não está 'não lida')", () => {
    mount();
    handler?.({ action: "create", message: { id: "b", ticketId: 7, fromMe: true } });
    expect(put).not.toHaveBeenCalled();
  });

  it("aba escondida: a mensagem continua não lida", () => {
    setVisibility("hidden");
    mount();
    handler?.({ action: "create", message: { id: "c", ticketId: 7, fromMe: false } });
    expect(put).not.toHaveBeenCalled();
  });

  it("mensagem de OUTRO ticket não zera este", () => {
    mount();
    handler?.({ action: "create", message: { id: "d", ticketId: 99, fromMe: false } });
    expect(put).not.toHaveBeenCalled();
  });

  it("atualização de ack não zera", () => {
    mount();
    handler?.({ action: "update", message: { id: "e", ticketId: 7, fromMe: false } });
    expect(put).not.toHaveBeenCalled();
  });
});
