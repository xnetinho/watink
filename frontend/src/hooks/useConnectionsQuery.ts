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

export const useWhatsAppsQuery = () => {
  return useQuery({
    queryKey: ["whatsapp"],
    queryFn: async () => {
      const { data } = await api.get<ChannelConnection[]>("/whatsapp/");
      return data;
    },
  });
};

export const useConnectionsQuery = () => {
  return useQuery({
    queryKey: ["connections"],
    queryFn: async () => {
      const { data } = await api.get<ChannelConnection[]>("/connections");
      return data;
    },
  });
};

export default useConnectionsQuery;
