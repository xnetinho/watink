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
  | { type: "UPDATE_CONNECTIONS"; payload: ChannelConnection }
  | { type: "UPDATE_SESSION"; payload: Partial<ChannelConnection> & { id: number } }
  | { type: "DELETE_CONNECTIONS"; payload: number }
  | { type: "RESET" };

function reducer(state: ChannelConnection[], action: ConnectionAction): ChannelConnection[] {
  switch (action.type) {
    case "LOAD_CONNECTIONS":
      return [...action.payload];

    case "UPDATE_CONNECTIONS": {
      const idx = state.findIndex((s) => s.id === action.payload.id);
      if (idx !== -1) {
        const next = [...state];
        next[idx] = action.payload;
        return next;
      }
      return [action.payload, ...state];
    }

    case "UPDATE_SESSION": {
      const idx = state.findIndex((s) => s.id === action.payload.id);
      if (idx !== -1) {
        const next = [...state];
        next[idx] = { ...next[idx], ...action.payload };
        return next;
      }
      return state;
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
      const [connRes, waRes] = await Promise.allSettled([
        api.get<ChannelConnection[]>("/connections"),
        api.get<ChannelConnection[]>("/whatsapp/"),
      ]);
      const conns = connRes.status === "fulfilled" && Array.isArray(connRes.value.data) ? connRes.value.data : [];
      const was = waRes.status === "fulfilled" && Array.isArray(waRes.value.data) ? waRes.value.data : [];
      dispatch({ type: "LOAD_CONNECTIONS", payload: [...conns, ...was] });
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
      connection?: ChannelConnection;
      whatsapp?: ChannelConnection;
      id?: number;
      whatsappId?: number;
    }) => {
      const item = data.connection || data.whatsapp;
      const id = data.id || data.whatsappId;
      if (data.action === "update" && item) {
        dispatch({ type: "UPDATE_CONNECTIONS", payload: item });
      }
      if (data.action === "delete" && id) {
        dispatch({ type: "DELETE_CONNECTIONS", payload: id });
      }
    };

    const handleSession = (data: {
      action: string;
      session?: Partial<ChannelConnection> & { id: number };
    }) => {
      if (data.action === "update" && data.session) {
        dispatch({ type: "UPDATE_SESSION", payload: data.session });
      }
    };

    return subscribeToSocket({
      connection: handleConnection,
      whatsapp: handleConnection,
      whatsappSession: handleSession,
    });
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
