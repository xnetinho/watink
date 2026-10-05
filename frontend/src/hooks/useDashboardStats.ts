import { useQuery, UseQueryOptions } from "@tanstack/react-query";
import api from "../services/api";

export interface DashboardStats {
  ticketsCount: number;
  openTickets: number;
  pendingTickets: number;
  closedTickets: number;
  ticketsByHour: Record<string, number>;
  metrics: {
    avgResponseTime: number;
    avgWaitTime: number;
  };
}

// Formato de GET /dashboard (business/internal/controllers/dashboard.go).
interface DashboardResponse {
  tickets?: { open?: number; pending?: number; closed?: number };
  ticketsByHour?: Record<string, number> | null;
  metrics?: { avgResponseTime?: number; avgWaitTime?: number };
}

export const toDashboardStats = (d: DashboardResponse): DashboardStats => {
  const open = d.tickets?.open ?? 0;
  const pending = d.tickets?.pending ?? 0;
  const closed = d.tickets?.closed ?? 0;
  return {
    ticketsCount: open + pending + closed,
    openTickets: open,
    pendingTickets: pending,
    closedTickets: closed,
    ticketsByHour: d.ticketsByHour ?? {},
    metrics: {
      avgResponseTime: d.metrics?.avgResponseTime ?? 0,
      avgWaitTime: d.metrics?.avgWaitTime ?? 0,
    },
  };
};

// Passa pelo axios (api): leva o token e o baseURL. O fetch puro anterior batia
// em /api/dashboard/stats, rota que nunca existiu, e sem Authorization.
export const fetchDashboardStats = async (): Promise<DashboardStats> => {
  const { data } = await api.get<DashboardResponse>("/dashboard");
  return toDashboardStats(data);
};

export const useDashboardStats = (options?: UseQueryOptions<DashboardStats, Error>) => {
  return useQuery<DashboardStats, Error>({
    queryKey: ["dashboard-stats"],
    queryFn: fetchDashboardStats,
    ...options,
  });
};
