/* @jsxImportSource react */
import React, { useState, useEffect } from "react";
import * as Yup from "yup";
import { Formik, Form, Field } from "formik";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "../../../components/ui/dialog";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import { Switch } from "../../../components/ui/switch";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../components/ui/select";
import { CHANNELS_CATALOG, ChannelDefinition } from "../../../config/channels";
import {
  MessageCircle,
  Send,
  ShoppingBag,
  Camera,
  Tag,
  Video,
  ArrowLeft,
  AlertCircle,
  Loader2,
} from "lucide-react";
import api from "../../../services/api";
import { notify } from "../../../lib/notify";
import { i18n } from "../../../translate/i18n";
import QueueSelect from "../../../components/QueueSelect";
import type {
  WhatsAppFormValues,
  Proxy,
  ProxyGroup,
  ConnectionGroup,
} from "../../../types/domain";

const SessionSchema = Yup.object().shape({
  name: Yup.string().min(2, "Muito curto!").max(50, "Muito longo!").required("Obrigatório"),
});

interface ChannelSelectorModalProps {
  open: boolean;
  onClose: () => void;
  onSelectWhatsApp?: () => void;
  onChannelCreated?: () => void;
  /** Quando informado, o modal edita a conexão existente em vez de criar outra. */
  editConnection?: EditableChannelConnection | null;
}

export interface EditableChannelConnection {
  id: number;
  kind: string;
  name: string;
  config?: string;
}

export const ChannelSelectorModal: React.FC<ChannelSelectorModalProps> = ({
  open,
  onClose,
  onChannelCreated,
  editConnection,
}) => {
  const [selectedChannel, setSelectedChannel] = useState<ChannelDefinition | null>(null);
  const [tokenInput, setTokenInput] = useState("");
  const [nameInput, setNameInput] = useState("");
  const [modeInput, setModeInput] = useState<"webhook" | "polling">("webhook");
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Estados para configuração do WhatsApp
  const [whatsAppForm, setWhatsAppForm] = useState<WhatsAppFormValues>({
    name: "",
    isDefault: false,
    keepAlive: false,
    syncHistory: false,
    engineType: "whatsmeow",
    proxyMode: "none",
    proxyId: null,
    proxyGroupId: null,
    connectionGroupId: null,
  });
  const [selectedQueueIds, setSelectedQueueIds] = useState<number[]>([]);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [proxyGroups, setProxyGroups] = useState<ProxyGroup[]>([]);
  const [connectionGroups, setConnectionGroups] = useState<ConnectionGroup[]>([]);

  useEffect(() => {
    if (!open) return;
    api.get<Proxy[]>("/proxies", { params: { status: "active" } })
      .then(({ data }) => setProxies(Array.isArray(data) ? data : []))
      .catch(() => setProxies([]));
    api.get<ProxyGroup[]>("/proxy-groups")
      .then(({ data }) => setProxyGroups(Array.isArray(data) ? data : []))
      .catch(() => setProxyGroups([]));
    api.get<ConnectionGroup[]>("/connection-groups")
      .then(({ data }) => setConnectionGroups(Array.isArray(data) ? data : []))
      .catch(() => setConnectionGroups([]));
  }, [open]);

  const resetState = () => {
    setSelectedChannel(null);
    setTokenInput("");
    setNameInput("");
    setModeInput("webhook");
    setIsSubmitting(false);
    setWhatsAppForm({
      name: "",
      isDefault: false,
      keepAlive: false,
      syncHistory: false,
      engineType: "whatsmeow",
      proxyMode: "none",
      proxyId: null,
      proxyGroupId: null,
      connectionGroupId: null,
    });
    setSelectedQueueIds([]);
  };

  // Edição: pula a escolha do tipo, pré-preenche nome/modo e deixa o token em branco
  // (em branco mantém o segredo atual no backend).
  useEffect(() => {
    if (!open || !editConnection) return;
    const def = CHANNELS_CATALOG.find((c) => c.kind === editConnection.kind) || null;
    setSelectedChannel(def);
    setNameInput(editConnection.name);
    setTokenInput("");
    try {
      const cfg = JSON.parse(editConnection.config || "{}");
      setModeInput(cfg.mode === "polling" ? "polling" : "webhook");
    } catch {
      setModeInput("webhook");
    }
  }, [open, editConnection]);

  const handleClose = () => {
    resetState();
    onClose();
  };

  const handleSelect = (channel: ChannelDefinition) => {
    setSelectedChannel(channel);
  };

  const handleConnectToken = async () => {
    if (!nameInput.trim() || (!editConnection && !tokenInput.trim())) {
      notify.warning("Preencha o nome da conexão e o token de acesso.");
      return;
    }

    setIsSubmitting(true);
    try {
      if (editConnection) {
        await api.put(`/channels/connections/${editConnection.id}`, {
          name: nameInput.trim(),
          token: tokenInput.trim(),
          mode: modeInput,
        });
        notify.success(`Canal ${selectedChannel?.name ?? ""} atualizado com sucesso!`);
      } else {
        await api.post("/channels/connections", {
          kind: selectedChannel?.kind,
          name: nameInput.trim(),
          token: tokenInput.trim(),
          mode: modeInput,
        });
        notify.success(`Canal ${selectedChannel?.name} conectado com sucesso!`);
      }
      onChannelCreated?.();
      handleClose();
    } catch (err) {
      notify.error(err);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSaveWhatsApp = async (values: WhatsAppFormValues) => {
    setIsSubmitting(true);
    const payload = { ...values, queueIds: selectedQueueIds };
    try {
      await api.post("/whatsapp", payload);
      notify.success(i18n.t("whatsappModal.success"));
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
            {selectedChannel && !editConnection && (
              <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={() => setSelectedChannel(null)}>
                <ArrowLeft className="h-4 w-4" />
              </Button>
            )}
            <DialogTitle>
              {selectedChannel
                ? editConnection
                  ? `Editar ${selectedChannel.name}`
                  : `Conectar ${selectedChannel.name}`
                : "Selecione o Canal de Atendimento"}
            </DialogTitle>
          </div>
          <DialogDescription>
            {selectedChannel
              ? String(i18n.t(selectedChannel.description))
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
                  <p className="text-xs text-muted-foreground line-clamp-2 mt-1">{String(i18n.t(c.description))}</p>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="space-y-4 py-2">
            {selectedChannel.kind === "whatsapp" && (
              <Formik
                initialValues={whatsAppForm}
                enableReinitialize
                validationSchema={SessionSchema}
                onSubmit={async (values) => {
                  await handleSaveWhatsApp(values);
                }}
              >
                {({ values, touched, errors, setFieldValue }) => (
                  <Form className="space-y-4">
                    <div className="space-y-1">
                      <Label>{i18n.t("whatsappModal.form.name")}</Label>
                      <Field as={Input} name="name" placeholder="Ex: WhatsApp Vendas" autoFocus aria-invalid={!!(errors.name && touched.name)} />
                      {errors.name && touched.name && <p className="text-xs text-destructive">{errors.name as string}</p>}
                    </div>

                    <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 py-1">
                      <div className="flex items-center gap-2">
                        <Switch
                          id="wa-modal-default"
                          checked={values.isDefault}
                          onCheckedChange={(checked) => setFieldValue("isDefault", checked)}
                        />
                        <Label htmlFor="wa-modal-default" className="text-xs">{i18n.t("whatsappModal.form.default")}</Label>
                      </div>

                      <div className="flex items-center gap-2">
                        <Switch
                          id="wa-modal-keep-alive"
                          checked={values.keepAlive}
                          onCheckedChange={(checked) => setFieldValue("keepAlive", checked)}
                        />
                        <Label htmlFor="wa-modal-keep-alive" className="text-xs">Keep Alive</Label>
                      </div>

                      <div className="flex items-center gap-2">
                        <Switch
                          id="wa-modal-sync"
                          checked={values.syncHistory}
                          onCheckedChange={(checked) => setFieldValue("syncHistory", checked)}
                        />
                        <Label htmlFor="wa-modal-sync" className="text-xs">Sincronizar Histórico</Label>
                      </div>
                    </div>

                    <div className="space-y-1 pt-1">
                      <Label className="text-xs">Proxy (anti-ban)</Label>
                      <Select
                        value={
                          values.proxyMode === "group" && values.proxyGroupId
                            ? `group:${values.proxyGroupId}`
                            : values.proxyMode === "single" && values.proxyId
                            ? `single:${values.proxyId}`
                            : "none"
                        }
                        onValueChange={(v) => {
                          if (v === "none") {
                            setFieldValue("proxyMode", "none");
                            setFieldValue("proxyId", null);
                            setFieldValue("proxyGroupId", null);
                          } else if (v.startsWith("group:")) {
                            setFieldValue("proxyMode", "group");
                            setFieldValue("proxyGroupId", Number(v.slice(6)));
                            setFieldValue("proxyId", null);
                          } else {
                            setFieldValue("proxyMode", "single");
                            setFieldValue("proxyId", Number(v.slice(7)));
                            setFieldValue("proxyGroupId", null);
                          }
                        }}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="Sem proxy" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">Sem proxy (IP do servidor)</SelectItem>
                          {proxyGroups.map((g) => (
                            <SelectItem key={`g${g.id}`} value={`group:${g.id}`}>
                              Grupo: {g.name} ({g.rotationStrategy === "rotate" ? "rotação" : "fixo"})
                            </SelectItem>
                          ))}
                          {proxies.map((p) => (
                            <SelectItem key={`p${p.id}`} value={`single:${p.id}`}>
                              {p.label ? `${p.label} — ` : ""}{p.scheme}://{p.host}:{p.port}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="space-y-1 pt-1">
                      <Label className="text-xs">Grupo de conexões</Label>
                      <Select
                        value={values.connectionGroupId ? String(values.connectionGroupId) : "none"}
                        onValueChange={(v) => setFieldValue("connectionGroupId", v === "none" ? null : Number(v))}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="Sem grupo" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="none">Sem grupo</SelectItem>
                          {connectionGroups.map((g) => (
                            <SelectItem key={g.id} value={String(g.id)}>{g.name}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>

                    <div className="pt-1">
                      <Label className="mb-1 block text-xs">Filas Permitidas</Label>
                      <QueueSelect
                        selectedQueueIds={selectedQueueIds}
                        onChange={setSelectedQueueIds}
                      />
                    </div>

                    <div className="flex justify-end gap-2 pt-3 border-t border-border/50">
                      <Button variant="ghost" type="button" onClick={() => setSelectedChannel(null)}>
                        Voltar
                      </Button>
                      <Button type="submit" disabled={isSubmitting}>
                        {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                        {i18n.t("whatsappModal.buttons.okAdd")}
                      </Button>
                    </div>
                  </Form>
                )}
              </Formik>
            )}

            {selectedChannel.authKind === "token" && selectedChannel.kind !== "whatsapp" && (
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
                  <label className="text-xs font-medium text-foreground">Modo de Operação</label>
                  <Select value={modeInput} onValueChange={(val: "webhook" | "polling") => setModeInput(val)}>
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder="Selecione o modo" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="webhook">Webhook (Recomendado - Instantâneo)</SelectItem>
                      <SelectItem value="polling">Polling (Busca Ativa - Sem URL pública)</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="text-[11px] text-muted-foreground mt-0.5">
                    {modeInput === "webhook"
                      ? "O Telegram enviará as notificações diretamente para a URL pública do Watink."
                      : "O Watink buscará novas mensagens em loop (ideal se o servidor não tiver IP/domínio público)."}
                  </p>
                </div>
                <div className="space-y-1">
                  <label className="text-xs font-medium text-foreground">Token do Bot (API Token)</label>
                  <Input
                    placeholder={
                      editConnection
                        ? "Deixe em branco para manter o token atual"
                        : "Cole o token recebido do @BotFather"
                    }
                    type="password"
                    value={tokenInput}
                    onChange={(e) => setTokenInput(e.target.value)}
                  />
                  <p className="text-[11px] text-muted-foreground mt-1">
                    {editConnection
                      ? "Preencha apenas se quiser trocar o token do bot."
                      : "Crie um bot no Telegram falando com o @BotFather e copie o token gerado."}
                  </p>
                </div>
                <div className="flex justify-end gap-2 pt-2">
                  <Button variant="ghost" onClick={handleClose}>Cancelar</Button>
                  <Button onClick={handleConnectToken} disabled={isSubmitting}>
                    {isSubmitting
                      ? "Validando..."
                      : editConnection
                      ? "Salvar alterações"
                      : "Salvar Conexão"}
                  </Button>
                </div>
              </div>
            )}

            {(selectedChannel.authKind === "oauth" || selectedChannel.authKind === "oauth_review") && selectedChannel.kind !== "whatsapp" && (
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
