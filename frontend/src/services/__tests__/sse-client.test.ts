import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";

const apiGet = vi.fn();
vi.mock("../api", () => ({ default: { get: (...a: unknown[]) => apiGet(...a) } }));
vi.mock("../../config", () => ({ getBackendUrl: () => "https://app.test" }));

class FakeEventSource {
  static CLOSED = 2;
  static instances: FakeEventSource[] = [];
  readyState = 1;
  onerror: (() => void) | null = null;
  onmessage: (() => void) | null = null;
  constructor(public url: string) {
    FakeEventSource.instances.push(this);
  }
  addEventListener() {}
  close() {
    this.readyState = FakeEventSource.CLOSED;
  }
}

function jwt(expSecondsFromNow: number): string {
  const body = btoa(JSON.stringify({ exp: Math.floor(Date.now() / 1000) + expSecondsFromNow }))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
  return `h.${body}.s`;
}

function storeToken(t: string | null) {
  if (t === null) localStorage.removeItem("token");
  else localStorage.setItem("token", JSON.stringify(t));
}

async function load() {
  vi.resetModules();
  return import("../sse-client");
}

describe("sse-client com access token expirado", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    FakeEventSource.instances = [];
    apiGet.mockReset();
    apiGet.mockResolvedValue({ data: {} });
    vi.stubGlobal("EventSource", FakeEventSource);
    localStorage.clear();
  });
  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  it("token válido: abre o stream direto, sem renovar", async () => {
    storeToken(jwt(3600));
    const { subscribeToSocket } = await load();
    subscribeToSocket({ ticket: () => {} }, (s) => s.emit("joinChat", 1));
    await vi.advanceTimersByTimeAsync(200);

    expect(FakeEventSource.instances).toHaveLength(1);
    expect(apiGet).not.toHaveBeenCalled();
  });

  it("token expirado: renova via axios ANTES de abrir o stream (nunca envia o vencido)", async () => {
    const expired = jwt(-60);
    storeToken(expired);
    apiGet.mockImplementation(async () => {
      storeToken(jwt(3600));
      return { data: {} };
    });
    const { subscribeToSocket } = await load();
    subscribeToSocket({ ticket: () => {} }, (s) => s.emit("joinChat", 1));
    await vi.advanceTimersByTimeAsync(200);

    expect(apiGet).toHaveBeenCalledTimes(1);
    expect(FakeEventSource.instances).toHaveLength(1);
    expect(FakeEventSource.instances[0].url).not.toContain(expired);
  });

  it("refresh falha e a sessão é limpa: para de reconectar de vez", async () => {
    storeToken(jwt(-60));
    apiGet.mockImplementation(async () => {
      storeToken(null);
      throw new Error("401");
    });
    const { subscribeToSocket } = await load();
    subscribeToSocket({ ticket: () => {} }, (s) => s.emit("joinChat", 1));
    await vi.advanceTimersByTimeAsync(60_000);

    expect(FakeEventSource.instances).toHaveLength(0);
    expect(apiGet).toHaveBeenCalledTimes(1);
  });

  it("onerror com token que expirou: renova e reabre com o token novo", async () => {
    storeToken(jwt(3600));
    const { subscribeToSocket } = await load();
    subscribeToSocket({ ticket: () => {} }, (s) => s.emit("joinChat", 1));
    await vi.advanceTimersByTimeAsync(200);
    expect(FakeEventSource.instances).toHaveLength(1);

    const fresh = jwt(7200);
    storeToken(jwt(-5));
    apiGet.mockImplementation(async () => {
      storeToken(fresh);
      return { data: {} };
    });
    FakeEventSource.instances[0].onerror?.();
    await vi.advanceTimersByTimeAsync(2_000);

    expect(apiGet).toHaveBeenCalledTimes(1);
    expect(FakeEventSource.instances).toHaveLength(2);
    expect(FakeEventSource.instances[1].url).toContain(encodeURIComponent(fresh));
  });

  it("refresh responde 200 mas o token segue vencido: não entra em loop apertado", async () => {
    storeToken(jwt(-60));
    const { subscribeToSocket } = await load();
    subscribeToSocket({ ticket: () => {} }, (s) => s.emit("joinChat", 1));
    await vi.advanceTimersByTimeAsync(20_000);

    expect(FakeEventSource.instances).toHaveLength(0);
    expect(apiGet.mock.calls.length).toBeLessThanOrEqual(6);
  });
});
