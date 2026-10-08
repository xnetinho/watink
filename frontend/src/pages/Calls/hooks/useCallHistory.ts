import { useCallback, useEffect, useState } from "react";
import api from "@/services/api";
import type { CallFilters, CallHistoryResponse, CallLogRow } from "../types";

const DEFAULT: CallFilters = { status: "", direction: "", page: 1, pageSize: 20 };

/** Query string do histórico: só envia filtros preenchidos. */
export function buildHistoryParams(f: CallFilters): Record<string, string | number> {
  const p: Record<string, string | number> = { page: f.page, pageSize: f.pageSize };
  if (f.status) p.status = f.status;
  if (f.direction) p.direction = f.direction;
  return p;
}

export function useCallHistory() {
  const [filters, setFilters] = useState<CallFilters>(DEFAULT);
  const [rows, setRows] = useState<CallLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(false);
    try {
      const { data } = await api.get<CallHistoryResponse>("/calls", { params: buildHistoryParams(filters) });
      setRows(data.calls ?? []);
      setTotal(data.total ?? 0);
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    void load();
  }, [load]);

  const update = useCallback((patch: Partial<CallFilters>) => {
    // Mudar um filtro volta à primeira página; mudar só a página não.
    setFilters((f) => ({ ...f, ...patch, page: "page" in patch ? (patch.page as number) : 1 }));
  }, []);

  const removeRecordingLocally = useCallback((callId: string) => {
    setRows((rs) => rs.map((r) => (r.callId === callId ? { ...r, recordingStatus: "deleted" } : r)));
  }, []);

  return { filters, update, rows, total, loading, error, reload: load, removeRecordingLocally };
}
