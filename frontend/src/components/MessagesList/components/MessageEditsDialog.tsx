import React, { useEffect, useState } from "react";
import { format, parseISO } from "date-fns";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "../../ui/dialog";
import { Button } from "../../ui/button";
import { Skeleton } from "../../ui/skeleton";
import { i18n } from "../../../translate/i18n";
import { fetchMessageEdits, type MessageEditVersion } from "../../../lib/messages/edits";

const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

interface Props {
  messageId: string | number;
  onClose: () => void;
}

/** Linha do tempo das versões de uma mensagem editada: a original no topo, a atual no fim. */
const MessageEditsDialog: React.FC<Props> = ({ messageId, onClose }) => {
  const [versions, setVersions] = useState<MessageEditVersion[] | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let alive = true;
    fetchMessageEdits(messageId)
      .then((v) => alive && setVersions(v))
      .catch(() => alive && setFailed(true));
    return () => {
      alive = false;
    };
  }, [messageId]);

  const last = versions ? versions.length : 0;

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{tr("chat.edits.title")}</DialogTitle>
        </DialogHeader>

        {failed && <p className="text-sm text-[hsl(var(--status-error))]">{tr("chat.edits.loadFailed")}</p>}
        {!failed && versions === null && <Skeleton className="h-24 w-full" />}
        {versions && versions.length === 0 && <p className="text-sm text-[hsl(var(--text-muted))]">{tr("chat.edits.empty")}</p>}

        {versions && versions.length > 0 && (
          <ol className="flex max-h-[60vh] flex-col gap-3 overflow-y-auto" data-testid="edits-list">
            {versions.map((v, i) => {
              const isOriginal = v.version === 1;
              const isCurrent = i === last - 1;
              return (
                <li key={v.id} className="rounded-lg border border-[hsl(var(--border))] p-3" data-testid={`edit-version-${v.version}`}>
                  <div className="mb-1 flex flex-wrap items-center gap-x-2 text-[11px] text-[hsl(var(--text-muted))]">
                    <span className="font-medium">
                      {isOriginal ? tr("chat.edits.original") : tr("chat.edits.version", { number: v.version - 1 })}
                    </span>
                    {isCurrent && <span className="font-medium text-[hsl(var(--action-primary))]">{tr("chat.edits.current")}</span>}
                    <span>{format(parseISO(v.editedAt), "dd/MM/yyyy HH:mm:ss")}</span>
                    {!isOriginal && <span>{tr(v.editedBy === "me" ? "chat.edits.byMe" : "chat.edits.byContact")}</span>}
                  </div>
                  <p className="whitespace-pre-wrap break-words text-sm">{v.body}</p>
                </li>
              );
            })}
          </ol>
        )}

        <div className="flex justify-end">
          <Button variant="outline" onClick={onClose}>
            {tr("chat.edits.close")}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
};

export default MessageEditsDialog;
