import React, { useEffect, useState } from "react";
import { format, parseISO } from "date-fns";
import { Check, CheckCheck, Headphones } from "lucide-react";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "../../ui/dialog";
import { Button } from "../../ui/button";
import { Skeleton } from "../../ui/skeleton";
import { i18n } from "../../../translate/i18n";
import { fetchMessageReceipts, type MessageReceiptRow } from "../../../lib/messages/receipts";

const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

interface Props {
  messageId: string | number;
  /** Mostra a linha "Reproduzida" (só faz sentido em áudio). */
  isAudio?: boolean;
  onClose: () => void;
}

const when = (iso: string | null): string => (iso ? format(parseISO(iso), "dd/MM/yyyy HH:mm:ss") : tr("chat.receipts.none"));

/** "Dados da mensagem": quando cada participante recebeu, leu e (em áudio) ouviu a mensagem. */
const MessageReceiptsDialog: React.FC<Props> = ({ messageId, isAudio = false, onClose }) => {
  const [rows, setRows] = useState<MessageReceiptRow[] | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let alive = true;
    fetchMessageReceipts(messageId)
      .then((r) => alive && setRows(r))
      .catch(() => alive && setFailed(true));
    return () => {
      alive = false;
    };
  }, [messageId]);

  const several = (rows?.length ?? 0) > 1 || (rows?.[0]?.senderJid ?? "") !== "";

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{tr("chat.receipts.title")}</DialogTitle>
        </DialogHeader>

        {failed && <p className="text-sm text-[hsl(var(--status-error))]">{tr("chat.receipts.loadFailed")}</p>}
        {!failed && rows === null && <Skeleton className="h-24 w-full" />}
        {rows && rows.length === 0 && <p className="text-sm text-[hsl(var(--text-muted))]">{tr("chat.receipts.empty")}</p>}

        {rows && rows.length > 0 && (
          <ul className="flex max-h-[60vh] flex-col gap-3 overflow-y-auto" data-testid="receipts-list">
            {rows.map((r, i) => (
              <li key={r.senderJid || i} className="rounded-lg border border-[hsl(var(--border))] p-3" data-testid="receipt-row">
                {several && (
                  <p className="mb-2 text-sm font-medium">{r.name || r.senderJid.split("@")[0] || tr("chat.receipts.unknown")}</p>
                )}
                <dl className="grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-1.5 text-sm">
                  <dt className="flex items-center gap-1.5 text-[hsl(var(--text-muted))]">
                    <CheckCheck className="h-4 w-4" aria-hidden="true" />
                    {tr("chat.receipts.delivered")}
                  </dt>
                  <dd data-testid="receipt-delivered">{when(r.delivered)}</dd>
                  <dt className="flex items-center gap-1.5 text-[hsl(var(--text-muted))]">
                    <CheckCheck className="h-4 w-4 text-[hsl(var(--action-primary))]" aria-hidden="true" />
                    {tr("chat.receipts.read")}
                  </dt>
                  <dd data-testid="receipt-read">{when(r.read)}</dd>
                  {isAudio && (
                    <>
                      <dt className="flex items-center gap-1.5 text-[hsl(var(--text-muted))]">
                        <Headphones className="h-4 w-4" aria-hidden="true" />
                        {tr("chat.receipts.played")}
                      </dt>
                      <dd data-testid="receipt-played">{when(r.played)}</dd>
                    </>
                  )}
                </dl>
              </li>
            ))}
          </ul>
        )}

        <p className="flex items-start gap-1.5 text-[11px] text-[hsl(var(--text-muted))]">
          <Check className="mt-0.5 h-3 w-3 shrink-0" aria-hidden="true" />
          {tr("chat.receipts.hint")}
        </p>

        <div className="flex justify-end">
          <Button variant="outline" onClick={onClose}>
            {tr("chat.receipts.close")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default MessageReceiptsDialog;
