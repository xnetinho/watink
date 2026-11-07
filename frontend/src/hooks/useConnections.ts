import { useState, useEffect, useReducer, useCallback } from "react";
import { subscribeToSocket } from "../services/sse-client";
import toastError from "../errors/toastError";
import api from "../services/api";

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

export type WhatsApp = ChannelConnection;

type ConnectionAction =
  | { type: "LOAD_CONNECTIONS"; payload: ChannelConnection[] }
  | { type: "UPDATE_CONNECTIONS"; payload: Partial<ChannelConnection> & { id: number } }
  | { type: "DELETE_CONNECTIONS"; payload: number }
  | { type: "RESET" };

function reducer(state: ChannelConnection[], action: ConnectionAction): ChannelConnection[] {
  switch (action.type) {
    case "LOAD_CONNECTIONS":
      return [...action.payload];

    // O evento pode trazer a conexão inteira ou só os campos que mudaram: mescla sobre o que já existe.
    case "UPDATE_CONNECTIONS": {
      const idx = state.findIndex((s) => s.id === action.payload.id);
      if (idx !== -1) {
        const next = [...state];
        next[idx] = { ...next[idx], ...action.payload };
        return next;
      }
      return [action.payload as ChannelConnection, ...state];
    }

    case "DELETE_CONNECTIONS": {
      return state.filter((s) => s.id !== action.payload);
    }

    case "RESET":
      return [];

    default:
      return state;
  }
}

export interface UseConnectionsResult {
  connections: ChannelConnection[];
  whatsApps: ChannelConnection[];
  loading: boolean;
  reloadConnections: () => Promise<void>;
  reloadWhatsApps: () => Promise<void>;
}

export const useConnections = (): UseConnectionsResult => {
  const [connections, dispatch] = useReducer(reducer, []);
  const [loading, setLoading] = useState(true);

  const reloadConnections = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await api.get<ChannelConnection[]>("/channels/connections");
      dispatch({ type: "LOAD_CONNECTIONS", payload: Array.isArray(data) ? data : [] });
    } catch (err) {
      toastError(err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    reloadConnections();
  }, [reloadConnections]);

  useEffect(() => {
    const handleConnection = (data: {
      action: string;
      connection?: Partial<ChannelConnection> & { id: number };
      connectionId?: number;
    }) => {
      if (data.action === "update" && data.connection) {
        dispatch({ type: "UPDATE_CONNECTIONS", payload: data.connection });
      }
      const id = data.connection?.id ?? data.connectionId;
      if (data.action === "delete" && id) {
        dispatch({ type: "DELETE_CONNECTIONS", payload: id });
      }
    };

    return subscribeToSocket({ channelConnection: handleConnection });
  }, []);

  return {
    connections,
    whatsApps: connections,
    loading,
    reloadConnections,
    reloadWhatsApps: reloadConnections,
  };
};

export default useConnections;
