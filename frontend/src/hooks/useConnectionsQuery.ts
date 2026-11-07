import api from "../services/api";
import { useQuery } from "@tanstack/react-query";

export interface ChannelConnection {
  id: number;
  name: string;
  kind?: string;
  status: string;
  updatedAt?: string;
  qrcode?: string;
  retries?: number;
  lastError?: string;
  config?: string;
  [key: string]: unknown;
}

const fetchConnections = async () => {
  const { data } = await api.get<ChannelConnection[]>("/channels/connections");
  return data;
};

export const useConnectionsQuery = () => {
  return useQuery({ queryKey: ["connections"], queryFn: fetchConnections });
};

/** Só as conexões WhatsApp, da mesma consulta (um único GET). */
export const useWhatsAppsQuery = () => {
  return useQuery({
    queryKey: ["connections"],
    queryFn: fetchConnections,
    select: (list) => list.filter((c) => c.kind === "whatsapp"),
  });
};

export default useConnectionsQuery;
