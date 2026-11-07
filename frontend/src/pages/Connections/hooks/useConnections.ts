import { useState, useEffect } from "react";
import { useNavigate } from "react-router";
import { toast } from "react-toastify";

import api from "../../../services/api";
import toastError from "../../../errors/toastError";
import { i18n } from "../../../translate/i18n";
import { useConnectionsQuery } from "../../../hooks/useConnectionsQuery";
import { subscribeToSocket } from "../../../services/sse-client";

import type { ConnectionSession } from "../connectionsTypes";

export interface ChannelConnection {
  id: number;
  kind: string;
  name: string;
  status: string;
  lastError?: string;
  config?: string;
}

export interface UseConnectionsReturn {
  whatsApps: ConnectionSession[];
  reloadWhatsApps: () => void;
  channelConnections: ChannelConnection[];
  handleDeleteChannelConnection: (id: number) => Promise<void>;
  whatsAppModalOpen: boolean;
  webchatModalOpen: boolean;
  confirmModalOpen: boolean;
  selectedWhatsApp: ConnectionSession | null;
  /** IDs com um POST /channels/connections/:id/session em voo — usado pra desabilitar o botão e evitar
   * disparar o mesmo start (lock Redis de 10s) duas vezes antes do SSE atualizar o status. */
  pendingSessionIds: Set<number>;
  handleStartSession: (id: number) => Promise<void>;
  handleRequestNewQrCode: (id: number) => Promise<void>;
  handleDisconnectSession: (id: number) => Promise<void>;
  handleDeleteWhatsApp: () => Promise<void>;
  handleEditWhatsApp: (whatsApp: ConnectionSession) => void;
  handleOpenNewWhatsApp: () => void;
  handleOpenNewWebchat: () => void;
  handleCloseWhatsAppModal: () => void;
  handleCloseWebchatModal: () => void;
  handleOpenConfirm: (whatsApp: ConnectionSession) => void;
  handleCloseConfirm: () => void;
  navigate: ReturnType<typeof useNavigate>;
}

export const useConnections = (): UseConnectionsReturn => {
  const navigate = useNavigate();
  const { data: all = [], refetch: reloadWhatsApps } = useConnectionsQuery();
  const whatsApps = all.filter((c) => c.kind === "whatsapp");
  const channelConnections = all.filter((c) => c.kind !== "whatsapp") as ChannelConnection[];

  const [whatsAppModalOpen, setWhatsAppModalOpen] = useState(false);
  const [webchatModalOpen, setWebchatModalOpen] = useState(false);
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);
  const [selectedWhatsApp, setSelectedWhatsApp] = useState<ConnectionSession | null>(null);
  const [pendingSessionIds, setPendingSessionIds] = useState<Set<number>>(new Set());

  const handleDeleteChannelConnection = async (id: number) => {
    try {
      await api.delete(`/channels/connections/${id}`);
      await reloadWhatsApps();
      toast.warn(i18n.t("connections.toasts.deleted"));
    } catch (err) {
      toastError(err);
    }
  };

  const addPending = (id: number) => setPendingSessionIds((prev) => new Set(prev).add(id));
  const removePending = (id: number) =>
    setPendingSessionIds((prev) => {
      if (!prev.has(id)) return prev;
      const next = new Set(prev);
      next.delete(id);
      return next;
    });

  useEffect(() => {
    const refetch = () => { reloadWhatsApps(); };
    // whatsappSessionRisk é o mesmo refetch — o valor persistido (lastRiskCode/
    // lastRiskAt) já vem no GET /channels/connections, então só precisamos disparar o reload.
    // channelConnection também libera o pending assim que o status muda (OPENING/QRCODE/...):
    // sem isso o botão ficaria desabilitado até o próximo fetch manual.
    const onConnection = (data: { connection?: { id?: number } }) => {
      if (typeof data?.connection?.id === "number") removePending(data.connection.id);
      reloadWhatsApps();
    };
    return subscribeToSocket({ channelConnection: onConnection, whatsappSessionRisk: refetch });
  }, [reloadWhatsApps]);

  // Clique duplo (ou "Conectar" + "Reiniciar Sessão" do mesmo card) batia no lock Redis de 10s do
  // backend (ERR_SESSION_STARTING_ALREADY) sem feedback visual nenhum — o pending local evita o
  // segundo POST/PUT antes do primeiro responder ou do SSE atualizar o status.
  const handleStartSession = async (id: number) => {
    if (pendingSessionIds.has(id)) return;
    addPending(id);
    try {
      await api.post(`/channels/connections/${id}/session`);
    } catch (err) {
      removePending(id);
      toastError(err);
    }
  };

  const handleRequestNewQrCode = async (id: number) => {
    if (pendingSessionIds.has(id)) return;
    addPending(id);
    try {
      await api.post(`/channels/connections/${id}/session`);
    } catch (err) {
      removePending(id);
      toastError(err);
    }
  };

  const handleDisconnectSession = async (id: number) => {
    try {
      await api.delete(`/channels/connections/${id}/session`);
      await reloadWhatsApps();
      toast.success(i18n.t("connections.toasts.disconnected"));
    } catch (err) {
      toastError(err);
    }
  };

  const handleDeleteWhatsApp = async () => {
    if (!selectedWhatsApp) return;
    try {
      if (
        selectedWhatsApp.status !== "DISCONNECTED" &&
        selectedWhatsApp.status !== "TIMEOUT"
      ) {
        try {
          await api.delete(`/channels/connections/${selectedWhatsApp.id}/session`);
        } catch (err) {
          console.warn("Could not stop session before deleting:", err);
        }
      }
      await api.delete(`/channels/connections/${selectedWhatsApp.id}`);
      await reloadWhatsApps();
      toast.warn(i18n.t("connections.toasts.deleted"));
    } catch (err) {
      toastError(err);
    }
    setConfirmModalOpen(false);
    setSelectedWhatsApp(null);
  };

  const handleEditWhatsApp = (whatsApp: ConnectionSession) => {
    setSelectedWhatsApp(whatsApp);
    if (whatsApp.type === "webchat") {
      setWebchatModalOpen(true);
    } else {
      setWhatsAppModalOpen(true);
    }
  };

  const handleOpenNewWhatsApp = () => {
    setSelectedWhatsApp(null);
    setWhatsAppModalOpen(true);
  };

  const handleOpenNewWebchat = () => {
    setSelectedWhatsApp(null);
    setWebchatModalOpen(true);
  };

  const handleCloseWhatsAppModal = () => {
    setWhatsAppModalOpen(false);
    setSelectedWhatsApp(null);
  };

  const handleCloseWebchatModal = () => {
    setWebchatModalOpen(false);
    setSelectedWhatsApp(null);
  };

  const handleOpenConfirm = (whatsApp: ConnectionSession) => {
    setSelectedWhatsApp(whatsApp);
    setConfirmModalOpen(true);
  };

  const handleCloseConfirm = () => {
    setConfirmModalOpen(false);
    setSelectedWhatsApp(null);
  };

  return {
    whatsApps: whatsApps as unknown as ConnectionSession[],
    reloadWhatsApps,
    channelConnections,
    handleDeleteChannelConnection,
    whatsAppModalOpen,
    webchatModalOpen,
    confirmModalOpen,
    selectedWhatsApp,
    pendingSessionIds,
    handleStartSession,
    handleRequestNewQrCode,
    handleDisconnectSession,
    handleDeleteWhatsApp,
    handleEditWhatsApp,
    handleOpenNewWhatsApp,
    handleOpenNewWebchat,
    handleCloseWhatsAppModal,
    handleCloseWebchatModal,
    handleOpenConfirm,
    handleCloseConfirm,
    navigate,
  };
};
