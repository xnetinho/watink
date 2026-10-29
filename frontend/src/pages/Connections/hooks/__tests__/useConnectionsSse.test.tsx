import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClientProvider } from "@tanstack/react-query";
import { queryClient as appQueryClient } from "../../../../lib/queryClient";

const handlers: Record<string, (d: unknown) => void> = {};
vi.mock("../../../../services/sse-client", () => ({
  subscribeToSocket: vi.fn((h: Record<string, (d: unknown) => void>) => {
    Object.assign(handlers, h);
    return () => undefined;
  }),
}));
const get = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { post: vi.fn(), put: vi.fn(), get: (...a: unknown[]) => get(...a) } }));

import { useConnections } from "../useConnections";

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <QueryClientProvider client={appQueryClient}>
    <MemoryRouter>{children}</MemoryRouter>
  </QueryClientProvider>
);

describe("lista de conexões atualiza por SSE", () => {
  beforeEach(() => {
    appQueryClient.clear();
    get.mockReset();
    for (const k of Object.keys(handlers)) delete handlers[k];
  });

  it("evento whatsappSession com novo status mostra o novo status sem recarregar a página", async () => {
    let status = "QRCODE";
    get.mockImplementation((url: string) => Promise.resolve({ data: url.startsWith("/whatsapp") ? [{ id: 1, name: "Teste", status }] : [] }));
    const { result } = renderHook(() => useConnections(), { wrapper });
    await waitFor(() => expect(result.current.whatsApps[0]?.status).toBe("QRCODE"));

    status = "CONNECTED"; // o servidor já mudou; o navegador só fica sabendo pelo evento
    act(() => handlers.whatsappSession({ action: "update", session: { id: 1, status: "CONNECTED" } }));

    await waitFor(() => expect(result.current.whatsApps[0]?.status).toBe("CONNECTED"));
  });
});

describe("corrida entre eventos de status (OPENING seguido de CONNECTED)", () => {
  beforeEach(() => {
    appQueryClient.clear();
    get.mockReset();
    for (const k of Object.keys(handlers)) delete handlers[k];
  });

  it("o segundo evento chega com a primeira busca ainda em voo e o status final precisa aparecer", async () => {
    let status = "DISCONNECTED";
    const pending: Array<() => void> = [];
    // cada GET /whatsapp responde com o status que o servidor tinha QUANDO a requisição saiu, mas só depois de um tempo
    get.mockImplementation((url: string) => {
      if (!url.startsWith("/whatsapp")) return Promise.resolve({ data: [] });
      const snapshot = status;
      return new Promise((res) => pending.push(() => res({ data: [{ id: 1, name: "Teste", status: snapshot }] })));
    });
    const { result } = renderHook(() => useConnections(), { wrapper });
    await waitFor(() => expect(pending.length).toBe(1));
    act(() => pending.shift()!());
    await waitFor(() => expect(result.current.whatsApps[0]?.status).toBe("DISCONNECTED"));

    status = "OPENING";
    act(() => handlers.whatsappSession({ action: "update", session: { id: 1, status: "OPENING" } }));
    await waitFor(() => expect(pending.length).toBe(1)); // a busca do OPENING está em voo, ainda sem resposta

    status = "CONNECTED"; // muda no servidor e o evento chega ENQUANTO a busca anterior está pendente
    act(() => handlers.whatsappSession({ action: "update", session: { id: 1, status: "CONNECTED" } }));

    while (pending.length) act(() => pending.shift()!());
    await waitFor(() => expect(result.current.whatsApps[0]?.status).toBe("CONNECTED"), { timeout: 1500 });
  });
});
