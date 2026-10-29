import React, { useEffect, useState } from "react";
import { Search } from "lucide-react";
import api from "../services/api";
import toastError from "../errors/toastError";
import { toast } from "react-toastify";
import { i18n } from "../translate/i18n";
import { channelDefinition } from "../config/channels";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "./ui/dialog";
import { Button } from "./ui/button";
import { Checkbox } from "./ui/checkbox";
import { Input } from "./ui/input";
import { TicketConnectionBadge } from "./TicketConnectionBadge";
import { useConnectionOptions } from "../hooks/useConnectionOptions";
import { connectionKey, type ConnectionOption } from "../lib/connections/connectionRef";
import { Emojified } from "./Emojified";
import { AudioKindPicker, type AudioKind } from "./AudioKindPicker";
import type { Contact } from "../types/Ticket";

// i18n.t() deste projeto pode devolver null; a tela só quer string.
const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

// Mesmo limite do backend (maxForwardTargets).
const MAX_TARGETS = 20;

type Mode = "forward" | "copy";
type Result = { contactId: number; ok: boolean; code?: string };

interface Props {
  open: boolean;
  onClose: () => void;
  messageId: string | number;
  /** Canal da conversa de origem: só conversas do mesmo canal são oferecidas como destino. */
  channel?: string;
  /** Contato da conversa aberta: não é oferecido como destino. */
  currentContactId?: number | string;
  /** Tipo da mensagem a encaminhar: o seletor "como enviar o áudio" só aparece para áudio. */
  mediaType?: string;
  /** Se o áudio de origem era mensagem de voz (é o padrão da escolha). */
  sourceIsVoice?: boolean;
}

// O contato carrega só (origem, id) da conexão; o nome vem da lista de conexões da empresa.
const connectionOf = (c: Contact, options: ConnectionOption[]) => {
  const o = options.find((x) => connectionKey(x) === `${c.connectionSource}:${c.connectionId}`);
  return o ? { kind: o.kind, id: o.id, name: o.name, source: o.source } : undefined;
};

const ForwardMessageDialog: React.FC<Props> = ({ open, onClose, messageId, channel, currentContactId, mediaType, sourceIsVoice }) => {
  const kind = channelDefinition(channel).kind;
  const [search, setSearch] = useState("");
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [selected, setSelected] = useState<number[]>([]);
  const [mode, setMode] = useState<Mode>("forward");
  const [sending, setSending] = useState(false);
  const isAudio = mediaType === "audio";
  const [audioKind, setAudioKind] = useState<AudioKind>(sourceIsVoice ? "voice" : "file");
  const { options: connections } = useConnectionOptions(open);

  useEffect(() => {
    if (!open) return;
    setSelected([]);
    setSearch("");
    setMode("forward");
    setAudioKind(sourceIsVoice ? "voice" : "file");
  }, [open, sourceIsVoice]);

  useEffect(() => {
    if (!open) return;
    const timer = setTimeout(async () => {
      try {
        // O destino é o CONTATO: lista os contatos do mesmo canal; cada um envia pela conexão a que pertence.
        const { data } = await api.get("/contacts", { params: { channel: kind, searchParam: search || undefined, limit: 50 } });
        setContacts((data.contacts as Contact[]).filter((c) => String(c.id) !== String(currentContactId)));
      } catch (err) {
        toastError(err);
      }
    }, 250);
    return () => clearTimeout(timer);
  }, [open, kind, search, currentContactId]);

  const toggle = (id: number) =>
    setSelected((cur) => {
      if (cur.includes(id)) return cur.filter((x) => x !== id);
      if (cur.length >= MAX_TARGETS) {
        toast.warning(tr("messageOptionsMenu.forwardDialog.maxTargets", { max: MAX_TARGETS }));
        return cur;
      }
      return [...cur, id];
    });

  const send = async () => {
    setSending(true);
    try {
      const { data } = await api.post<{ results: Result[] }>(`/message/${messageId}/forward`, { contactIds: selected, mode, ...(isAudio ? { audioKind } : {}) });
      const failed = data.results.filter((r) => !r.ok);
      if (failed.length === 0) {
        toast.success(tr("messageOptionsMenu.forwardDialog.done", { count: data.results.length }));
        onClose();
        return;
      }
      toast.warning(tr("messageOptionsMenu.forwardDialog.partial", { ok: data.results.length - failed.length, failed: failed.length }));
      const code = failed[0].code ?? "send_failed";
      const key = `messageOptionsMenu.forwardDialog.errors.${code}`;
      const text = tr(key);
      toast.error(text === key ? tr("messageOptionsMenu.forwardDialog.errors.send_failed") : text);
      // Mantém no diálogo só o que falhou, para tentar de novo (ex.: trocar para "reenviar como nova").
      setSelected(failed.map((r) => r.contactId));
    } catch (err) {
      const code = (err as { response?: { data?: { error?: string } } })?.response?.data?.error;
      const key = `messageOptionsMenu.forwardDialog.errors.${code}`;
      if (code && tr(key) !== key) toast.error(tr(key));
      else toastError(err);
    } finally {
      setSending(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md" data-testid="forward-dialog">
        <DialogHeader>
          <DialogTitle>{tr("messageOptionsMenu.forwardDialog.title")}</DialogTitle>
        </DialogHeader>

        <div className="flex flex-col gap-3">
          <fieldset className="flex flex-col gap-1.5" aria-label={tr("messageOptionsMenu.forwardDialog.modeLabel")}>
            {(["forward", "copy"] as Mode[]).map((m) => (
              <label key={m} className="flex cursor-pointer items-start gap-2 rounded-md border border-[hsl(var(--border-default))] p-2 text-sm has-[:checked]:border-[hsl(var(--action-primary))]">
                <input type="radio" name="forward-mode" className="mt-1" checked={mode === m} onChange={() => setMode(m)} />
                <span className="flex flex-col">
                  <span className="font-medium">{tr(m === "forward" ? "messageOptionsMenu.forwardDialog.modeForward" : "messageOptionsMenu.forwardDialog.modeCopy")}</span>
                  <span className="text-xs text-[hsl(var(--text-muted))]">
                    {tr(m === "forward" ? "messageOptionsMenu.forwardDialog.modeForwardHint" : "messageOptionsMenu.forwardDialog.modeCopyHint")}
                  </span>
                </span>
              </label>
            ))}
          </fieldset>

          {isAudio && <AudioKindPicker value={audioKind} onChange={setAudioKind} name="forward-audio-kind" />}

          <div className="relative">
            <Search className="absolute left-2 top-2.5 h-4 w-4 text-[hsl(var(--text-muted))]" />
            <Input
              className="pl-8"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={tr("messageOptionsMenu.forwardDialog.search")}
              aria-label={tr("messageOptionsMenu.forwardDialog.search")}
            />
          </div>
          <p className="text-xs text-[hsl(var(--text-muted))]">
            {tr("messageOptionsMenu.forwardDialog.sameChannelOnly", { channel: channelDefinition(channel).name })}
          </p>

          <ul className="max-h-64 overflow-y-auto" data-testid="forward-targets">
            {contacts.length === 0 && <li className="p-3 text-center text-sm text-[hsl(var(--text-muted))]">{tr("messageOptionsMenu.forwardDialog.none")}</li>}
            {contacts.map((c) => (
              <li key={c.id}>
                <label className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-[var(--border-subtle)]">
                  <Checkbox checked={selected.includes(c.id)} onCheckedChange={() => toggle(c.id)} aria-label={c.name} />
                  <span className="min-w-0 flex-1 truncate text-sm">
                    <Emojified>{c.name || c.number}</Emojified>
                    {c.number && c.name !== c.number && <span className="ml-2 text-xs text-[hsl(var(--text-muted))]">{c.number}</span>}
                  </span>
                  <TicketConnectionBadge connection={connectionOf(c, connections)} channel={kind} size="sm" />
                </label>
              </li>
            ))}
          </ul>
        </div>

        <DialogFooter className="items-center gap-2">
          <span className="mr-auto text-xs text-[hsl(var(--text-muted))]">{tr("messageOptionsMenu.forwardDialog.selected", { count: selected.length })}</span>
          <Button variant="outline" onClick={onClose}>
            {tr("messageOptionsMenu.forwardDialog.cancel")}
          </Button>
          <Button onClick={send} disabled={sending || selected.length === 0}>
            {sending ? tr("messageOptionsMenu.forwardDialog.sending") : tr("messageOptionsMenu.forwardDialog.send")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};

export default ForwardMessageDialog;
