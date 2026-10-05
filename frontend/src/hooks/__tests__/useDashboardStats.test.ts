import { describe, it, expect, vi } from "vitest";

const get = vi.fn();
vi.mock("../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a) } }));

import { fetchDashboardStats, toDashboardStats } from "../useDashboardStats";

describe("useDashboardStats", () => {
  it("busca em /dashboard (rota real) pelo axios, não por fetch em /api/dashboard/stats", async () => {
    get.mockClear();
    get.mockResolvedValue({ data: { tickets: { open: 2, pending: 1, closed: 4 } } });
    await fetchDashboardStats();
    expect(get).toHaveBeenCalledWith("/dashboard");
  });

  it("adapta o formato do backend para o que os widgets leem", async () => {
    get.mockResolvedValue({
      data: {
        tickets: { open: 2, pending: 1, closed: 4 },
        ticketsByHour: { "09:00": 3, "14:00": 1 },
        metrics: { avgResponseTime: 1.5, avgWaitTime: 7 },
      },
    });
    const s = await fetchDashboardStats();
    expect(s).toEqual({
      ticketsCount: 7,
      openTickets: 2,
      pendingTickets: 1,
      closedTickets: 4,
      ticketsByHour: { "09:00": 3, "14:00": 1 },
      metrics: { avgResponseTime: 1.5, avgWaitTime: 7 },
    });
  });

  it("tolera resposta parcial ou ticketsByHour nulo sem quebrar os widgets", () => {
    expect(toDashboardStats({})).toEqual({
      ticketsCount: 0,
      openTickets: 0,
      pendingTickets: 0,
      closedTickets: 0,
      ticketsByHour: {},
      metrics: { avgResponseTime: 0, avgWaitTime: 0 },
    });
    expect(toDashboardStats({ ticketsByHour: null }).ticketsByHour).toEqual({});
  });

  it("propaga erro da API (a query mostra o estado de erro, não zeros enganosos)", async () => {
    const failure = Object.assign(new Error("falha de rede"), { status: 500 });
    get.mockReturnValue(Promise.reject(failure));
    await expect(fetchDashboardStats()).rejects.toBe(failure);
  });
});
