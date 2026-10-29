/* @jsxImportSource react */
import React, { useState } from "react";
import { toast } from "react-toastify";
import api from "../../services/api";
import toastError from "../../errors/toastError";
import { Plus, MessageSquare } from "lucide-react";

import { i18n } from "../../translate/i18n";
import {
  PageContainer,
  PageHeader,
  PageContent,
} from "../../components/ui/page-layout";
import { Button } from "../../components/ui/button";
import WhatsAppModal from "../../components/WhatsAppModal";
import WebchatModal from "../../components/WebchatModal";
import ConfirmationModal from "../../components/ConfirmationModal";

import { useConnections } from "./hooks/useConnections";
import { ConnectionCard } from "./components/ConnectionCard";
import { ChannelSelectorModal } from "./components/ChannelSelectorModal";
import type { EditableChannelConnection } from "./components/ChannelSelectorModal";
import { ChannelBadge } from "../../components/ChannelBadge";
import { Card, CardContent, CardHeader, CardTitle, CardDescription, CardFooter } from "../../components/ui/card";
import { Trash2, Pencil } from "lucide-react";

const Connections = () => {
  const [channelSelectorOpen, setChannelSelectorOpen] = React.useState(false);
  const [editingChannel, setEditingChannel] = React.useState<EditableChannelConnection | null>(null);
  const {
    whatsApps,
    reloadWhatsApps,
    channelConnections,
    reloadChannelConnections,
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
  } = useConnections();
  const [importingId, setImportingId] = useState<number | null>(null);

  // A importação é sempre de UMA conexão: o contato nasce ligado a ela.
  const handleImportContacts = async () => {
    if (importingId === null) return;
    try {
      await api.post("/contacts/import", { connectionId: importingId });
      toast.success(String(i18n.t("connections.importContacts.started") ?? ""));
    } catch (err) {
      toastError(err);
    }
    setImportingId(null);
  };

  return (
    <PageContainer>
      <WhatsAppModal
        open={whatsAppModalOpen}
        onClose={handleCloseWhatsAppModal}
        whatsAppId={selectedWhatsApp?.id}
        onSaved={async () => { await reloadWhatsApps(); }}
      />
      <ChannelSelectorModal
        open={channelSelectorOpen}
        editConnection={editingChannel}
        onClose={() => { setChannelSelectorOpen(false); setEditingChannel(null); }}
        onSelectWhatsApp={handleOpenNewWhatsApp}
        onChannelCreated={async () => { await Promise.all([reloadWhatsApps(), reloadChannelConnections()]); }}
      />
      <WebchatModal
        open={webchatModalOpen}
        onClose={handleCloseWebchatModal}
        whatsAppId={selectedWhatsApp?.id}
        onSaved={async () => { await reloadWhatsApps(); }}
      />
      <ConfirmationModal
        title={String(i18n.t("connections.importContacts.title") ?? "")}
        open={importingId !== null}
        onClose={() => setImportingId(null)}
        onConfirm={handleImportContacts}
      >
        {String(i18n.t("connections.importContacts.message") ?? "")}
      </ConfirmationModal>
      <ConfirmationModal
        title={i18n.t("connections.confirmationModal.deleteTitle")}
        open={confirmModalOpen}
        onClose={handleCloseConfirm}
        onConfirm={handleDeleteWhatsApp}
      >
        {i18n.t("connections.confirmationModal.deleteMessage")}
      </ConfirmationModal>

      <PageHeader
        title={i18n.t("connections.title")}
        description="Gerencie seus canais de comunicação e conexões WhatsApp"
      >
        <div className="flex gap-2">
          <Button variant="ghost" onClick={handleOpenNewWebchat}>
            <MessageSquare className="mr-2 h-4 w-4" />
            Configurar Webchat
          </Button>
          <Button onClick={() => { setEditingChannel(null); setChannelSelectorOpen(true); }}>
            <Plus className="mr-2 h-4 w-4" />
            Nova Conexão
          </Button>
        </div>
      </PageHeader>

      <PageContent>
        {whatsApps.length === 0 && channelConnections.length === 0 && (
          <div className="flex flex-col items-center justify-center py-20 text-center gap-4">
            <div className="h-14 w-14 rounded-2xl bg-muted flex items-center justify-center">
              <MessageSquare className="h-7 w-7 text-muted-foreground" />
            </div>
            <div>
              <p className="font-semibold text-foreground">Nenhuma conexão configurada</p>
              <p className="text-sm text-muted-foreground mt-1">
                Adicione um WhatsApp para começar a atender seus clientes
              </p>
            </div>
            <Button onClick={() => { setEditingChannel(null); setChannelSelectorOpen(true); }}>
              <Plus className="mr-2 h-4 w-4" />
              Nova Conexão
            </Button>
          </div>
        )}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {channelConnections.map((conn) => (
            <Card key={`channel-${conn.id}`}>
              <CardHeader>
                <div className="flex items-center justify-between gap-2">
                  <ChannelBadge channel={conn.kind} />
                  <span
                    className={
                      conn.status === "CONNECTED"
                        ? "text-xs font-medium text-emerald-600 dark:text-emerald-400"
                        : "text-xs font-medium text-muted-foreground"
                    }
                  >
                    {conn.status === "CONNECTED" ? "Conectado" : conn.status}
                  </span>
                </div>
                <CardTitle className="text-base mt-2">{conn.name}</CardTitle>
                <CardDescription>
                  {conn.lastError ? (
                    <span className="text-destructive font-medium block mt-1 bg-destructive/10 p-2 rounded text-xs border border-destructive/20">
                      {conn.lastError}
                    </span>
                  ) : (
                    "Canal de atendimento conectado"
                  )}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <p className="text-xs text-muted-foreground">
                  Modo:{" "}
                  {(() => {
                    try {
                      const cfg = JSON.parse(conn.config || "{}");
                      return cfg.mode === "polling" ? "Polling" : "Webhook";
                    } catch {
                      return "Webhook";
                    }
                  })()}
                </p>
              </CardContent>
              <CardFooter className="justify-end gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setEditingChannel({ id: conn.id, kind: conn.kind, name: conn.name, config: conn.config });
                    setChannelSelectorOpen(true);
                  }}
                >
                  <Pencil className="mr-2 h-4 w-4" />
                  Editar
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  className="text-destructive hover:text-destructive"
                  onClick={() => handleDeleteChannelConnection(conn.id)}
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Excluir
                </Button>
              </CardFooter>
            </Card>
          ))}
          {whatsApps.map((whatsApp) => (
            <ConnectionCard
              key={whatsApp.id}
              whatsApp={whatsApp}
              pending={pendingSessionIds.has(whatsApp.id)}
              onNavigate={(id) => navigate(`/connections/${id}`)}
              onConnect={handleStartSession}
              onDisconnect={handleDisconnectSession}
              onRequestQrCode={handleRequestNewQrCode}
              onEdit={handleEditWhatsApp}
              onDelete={handleOpenConfirm}
              onImportContacts={setImportingId}
            />
          ))}
        </div>
      </PageContent>
    </PageContainer>
  );
};

export default Connections;
