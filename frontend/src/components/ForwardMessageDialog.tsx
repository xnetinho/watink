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
import { Emojified } from "./Emojified";
import type { Ticket } from "../types/Ticket";

// i18n.t() deste projeto pode devolver null; a tela só quer string.
const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

// Mesmo limite do backend (maxForwardTargets).
const MAX_TARGETS = 20;

type Mode = "forward" | "copy";
type Result = { ticketId: number; ok: boolean; code?: string };

interface Props {
  open: boolean;
  onClose: () => void;
  messageId: string | number;
  /** Canal da conversa de origem: só conversas do mesmo canal são oferecidas como destino. */
  channel?: string;
  currentTicketId?: number | string;
}

const ForwardMessageDialog: React.FC<Props> = ({ open, onClose, messageId, channel, currentTicketId }) => {
  const kind = channelDefinition(channel).kind;
  const [search, setSearch] = useState("");
  const [tickets, setTickets] = useState<Ticket[]>([]);
  const [selected, setSelected] = useState<number[]>([]);
  const [mode, setMode] = useState<Mode>("forward");
  const [sending, setSending] = useState(false);

  useEffect(() => {
    if (!open) return;
    setSelected([]);
    setSearch("");
    setMode("forward");
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const timer = setTimeout(async () => {
      try {
        const { data } = await api.get("/tickets", { params: { isGroup: "all", channel: kind, searchParam: search || undefined, limit: 50 } });
        setTickets((data.tickets as Ticket[]).filter((t) => String(t.id) !== String(currentTicketId)));
      } catch (err) {
        toastError(err);
      }
    }, 250);
    return () => clearTimeout(timer);
  }, [open, kind, search, currentTicketId]);

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
      const { data } = await api.post<{ results: Result[] }>(`/message/${messageId}/forward`, { ticketIds: selected, mode });
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
      setSelected(failed.map((r) => r.ticketId));
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
            {tickets.length === 0 && <li className="p-3 text-center text-sm text-[hsl(var(--text-muted))]">{tr("messageOptionsMenu.forwardDialog.none")}</li>}
            {tickets.map((t) => (
              <li key={t.id}>
                <label className="flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 hover:bg-[var(--border-subtle)]">
                  <Checkbox checked={selected.includes(t.id)} onCheckedChange={() => toggle(t.id)} aria-label={t.contact?.name} />
                  <span className="min-w-0 flex-1 truncate text-sm">
                    <Emojified>{t.contact?.name || t.contact?.number}</Emojified>
                  </span>
                  <TicketConnectionBadge channel={t.channel} connection={t.connection} size="sm" />
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
