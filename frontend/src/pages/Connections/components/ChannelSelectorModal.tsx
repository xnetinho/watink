/* @jsxImportSource react */
import React, { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "../../../components/ui/dialog";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { CHANNELS_CATALOG, ChannelDefinition } from "../../../config/channels";
import { MessageCircle, Send, ShoppingBag, Camera, Tag, Video, ArrowLeft, AlertCircle } from "lucide-react";
import api from "../../../services/api";
import { notify } from "../../../lib/notify";

interface ChannelSelectorModalProps {
  open: boolean;
  onClose: () => void;
  onSelectWhatsApp: () => void;
  onChannelCreated?: () => void;
}

export const ChannelSelectorModal: React.FC<ChannelSelectorModalProps> = ({
  open,
  onClose,
  onSelectWhatsApp,
  onChannelCreated,
}) => {
  const [selectedChannel, setSelectedChannel] = useState<ChannelDefinition | null>(null);
  const [tokenInput, setTokenInput] = useState("");
  const [nameInput, setNameInput] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const resetState = () => {
    setSelectedChannel(null);
    setTokenInput("");
    setNameInput("");
    setIsSubmitting(false);
  };

  const handleClose = () => {
    resetState();
    onClose();
  };

  const handleSelect = (channel: ChannelDefinition) => {
    if (channel.kind === "whatsapp") {
      handleClose();
      onSelectWhatsApp();
      return;
    }
    setSelectedChannel(channel);
  };

  const handleConnectToken = async () => {
    if (!tokenInput.trim() || !nameInput.trim()) {
      notify.warning("Preencha o nome da conexão e o token de acesso.");
      return;
    }

    setIsSubmitting(true);
    try {
      await api.post("/channels/connections", {
        kind: selectedChannel?.kind,
        name: nameInput.trim(),
        token: tokenInput.trim(),
      });
      notify.success(`Canal ${selectedChannel?.name} conectado com sucesso!`);
      onChannelCreated?.();
      handleClose();
    } catch (err) {
      notify.error(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  const getChannelIcon = (kind: string) => {
    switch (kind) {
      case "whatsapp":
        return <MessageCircle className="h-6 w-6 text-emerald-500" />;
      case "telegram":
        return <Send className="h-6 w-6 text-sky-500" />;
      case "mercadolivre":
        return <ShoppingBag className="h-6 w-6 text-amber-500" />;
      case "instagram":
        return <Camera className="h-6 w-6 text-pink-500" />;
      case "olx":
        return <Tag className="h-6 w-6 text-purple-500" />;
      case "tiktok":
        return <Video className="h-6 w-6 text-foreground" />;
      default:
        return <MessageCircle className="h-6 w-6" />;
    }
  };

  const getAvailabilityBadge = (av: string) => {
    switch (av) {
      case "available":
        return <span className="text-[10px] bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 font-semibold px-2 py-0.5 rounded-full border border-emerald-500/20">Disponível</span>;
      case "beta":
        return <span className="text-[10px] bg-amber-500/15 text-amber-600 dark:text-amber-400 font-semibold px-2 py-0.5 rounded-full border border-amber-500/20">Beta</span>;
      case "requires_approval":
        return <span className="text-[10px] bg-blue-500/15 text-blue-600 dark:text-blue-400 font-semibold px-2 py-0.5 rounded-full border border-blue-500/20">Requer Aprovação</span>;
      case "coming_soon":
        return <span className="text-[10px] bg-muted text-muted-foreground font-semibold px-2 py-0.5 rounded-full border">Em breve</span>;
      default:
        return null;
    }
  };

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <div className="flex items-center gap-2">
            {selectedChannel && (
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => setSelectedChannel(null)}>
                <ArrowLeft className="h-4 w-4" />
              </Button>
            )}
            <DialogTitle>
              {selectedChannel ? `Conectar ${selectedChannel.name}` : "Selecione o Canal de Atendimento"}
            </DialogTitle>
          </div>
          <DialogDescription>
            {selectedChannel
              ? selectedChannel.description
              : "Escolha o canal que deseja integrar para centralizar seus atendimentos no Watink."}
          </DialogDescription>
        </DialogHeader>

        {!selectedChannel ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 py-2 max-h-[60vh] overflow-y-auto">
            {CHANNELS_CATALOG.map((c) => (
              <div
                key={c.kind}
                onClick={() => handleSelect(c)}
                className="group relative flex flex-col justify-between p-4 rounded-xl border border-border/60 hover:border-primary/60 hover:bg-accent/40 cursor-pointer transition-all gap-2"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="p-2 rounded-lg bg-muted/60 group-hover:bg-background transition-colors">
                    {getChannelIcon(c.kind)}
                  </div>
                  {getAvailabilityBadge(c.availability)}
                </div>
                <div>
                  <h4 className="font-semibold text-sm text-foreground">{c.name}</h4>
                  <p className="text-xs text-muted-foreground line-clamp-2 mt-1">{c.description}</p>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="space-y-4 py-2">
            {selectedChannel.authKind === "token" && (
              <div className="space-y-3">
                <div className="space-y-1">
                  <label className="text-xs font-medium text-foreground">Nome da Conexão</label>
                  <Input
                    placeholder="Ex: Bot de Suporte Principal"
                    value={nameInput}
                    onChange={(e) => setNameInput(e.target.value)}
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs font-medium text-foreground">Token do Bot (API Token)</label>
                  <Input
                    placeholder="Cole o token recebido do @BotFather"
                    type="password"
                    value={tokenInput}
                    onChange={(e) => setTokenInput(e.target.value)}
                  />
                  <p className="text-[11px] text-muted-foreground mt-1">
                    Crie um bot no Telegram falando com o @BotFather e copie o token gerado.
                  </p>
                </div>
                <div className="flex justify-end gap-2 pt-2">
                  <Button variant="ghost" onClick={() => setSelectedChannel(null)}>Voltar</Button>
                  <Button onClick={handleConnectToken} disabled={isSubmitting}>
                    {isSubmitting ? "Validando..." : "Salvar Conexão"}
                  </Button>
                </div>
              </div>
            )}

            {(selectedChannel.authKind === "oauth" || selectedChannel.authKind === "oauth_review") && (
              <div className="rounded-xl border border-border/60 p-4 bg-muted/30 space-y-3">
                <div className="flex items-start gap-3">
                  <AlertCircle className="h-5 w-5 text-amber-500 shrink-0 mt-0.5" />
                  <div className="space-y-1">
                    <h5 className="text-xs font-semibold text-foreground">Requisitos de Conexão</h5>
                    <p className="text-xs text-muted-foreground">
                      Este canal requer autenticação via OAuth oficial da plataforma.
                      {selectedChannel.replyWindowHours && (
                        <span> As respostas a mensagens devem respeitar a janela de atendimento de até {selectedChannel.replyWindowHours} horas.</span>
                      )}
                    </p>
                  </div>
                </div>
                <div className="pt-2 flex justify-end gap-2">
                  <Button variant="ghost" onClick={() => setSelectedChannel(null)}>Voltar</Button>
                  <Button disabled variant="outline">
                    Em homologação (Fase 5)
                  </Button>
                </div>
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
};
