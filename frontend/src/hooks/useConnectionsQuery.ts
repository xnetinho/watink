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

export const useConnectionsQuery = () => {
  return useQuery({
    queryKey: ["connections"],
    queryFn: async () => {
      const { data } = await api.get<ChannelConnection[]>("/connections");
      return data;
    },
  });
};

export const useWhatsAppsQuery = useConnectionsQuery;
export default useConnectionsQuery;
