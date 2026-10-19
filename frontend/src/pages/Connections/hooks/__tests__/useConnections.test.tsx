import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

// api e sse-client são mockados: o foco deste teste é o guard local contra o lock Redis de 10s
// (ERR_SESSION_STARTING_ALREADY) — não a infra de rede/SSE real, já exercitada em outros níveis.
vi.mock("../../../../services/api", () => ({
  default: { post: vi.fn(), put: vi.fn(), get: vi.fn(() => Promise.resolve({ data: [] })) },
}));
vi.mock("../../../../services/sse-client", () => ({ subscribeToSocket: vi.fn(() => () => undefined) }));

import api from "../../../../services/api";
import { useConnections } from "../useConnections";

const wrapper = ({ children }: { children: React.ReactNode }) => {
  const qc = new QueryClient();
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
};

describe("useConnections — guard contra o lock de 10s do backend", () => {
  beforeEach(() => {
    vi.mocked(api.post).mockReset();
    vi.mocked(api.put).mockReset();
  });

  it("handleStartSession: um segundo clique antes da resposta NÃO dispara outro POST", async () => {
    let resolvePost!: () => void;
    vi.mocked(api.post).mockReturnValue(new Promise((res) => { resolvePost = () => res({ data: {} }); }));
    const { result } = renderHook(() => useConnections(), { wrapper });

    act(() => { void result.current.handleStartSession(1); });
    await waitFor(() => expect(result.current.pendingSessionIds.has(1)).toBe(true));

    act(() => { void result.current.handleStartSession(1); });
    expect(api.post).toHaveBeenCalledTimes(1);

    resolvePost();
    await waitFor(() => expect(api.post).toHaveBeenCalledTimes(1));
  });

  it("handleRequestNewQrCode: o mesmo guard vale pro PUT (Reiniciar Sessão)", async () => {
    let resolvePut!: () => void;
    vi.mocked(api.put).mockReturnValue(new Promise((res) => { resolvePut = () => res({ data: {} }); }));
    const { result } = renderHook(() => useConnections(), { wrapper });

    act(() => { void result.current.handleRequestNewQrCode(7); });
    await waitFor(() => expect(result.current.pendingSessionIds.has(7)).toBe(true));

    act(() => { void result.current.handleRequestNewQrCode(7); });
    expect(api.put).toHaveBeenCalledTimes(1);
    resolvePut();
  });

  it("conexões diferentes não se bloqueiam entre si", async () => {
    vi.mocked(api.post).mockReturnValue(new Promise(() => undefined));
    const { result } = renderHook(() => useConnections(), { wrapper });

    act(() => { void result.current.handleStartSession(1); });
    act(() => { void result.current.handleStartSession(2); });
    await waitFor(() => {
      expect(result.current.pendingSessionIds.has(1)).toBe(true);
      expect(result.current.pendingSessionIds.has(2)).toBe(true);
    });
    expect(api.post).toHaveBeenCalledTimes(2);
  });

  it("erro na chamada libera o pending (não fica preso pra sempre)", async () => {
    vi.mocked(api.post).mockRejectedValue(new Error("409"));
    const { result } = renderHook(() => useConnections(), { wrapper });

    act(() => { void result.current.handleStartSession(5); });
    await waitFor(() => expect(result.current.pendingSessionIds.has(5)).toBe(false));
  });
});
