import React, { useState } from "react";
import { Pencil } from "lucide-react";
import { i18n } from "../../../translate/i18n";
import { Message } from "../types";
import MessageEditsDialog from "./MessageEditsDialog";

const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

/**
 * Selo "Editada" do balão, no rodapé à esquerda (o horário fica à direita). Mostra que a mensagem mudou e abre o histórico completo (todas as versões, do original ao
 * texto atual, com data e hora). Some quando a mensagem nunca foi editada.
 */
// Na mensagem NOSSA o selo diz que fomos nós que editamos; na do contato, que ela foi editada.
const label = (fromMe: boolean, count: number): string => {
  if (fromMe) return tr(count > 1 ? "chat.edits.youEditedMany" : "chat.edits.youEdited", { count });
  return tr(count > 1 ? "chat.edits.editedMany" : "chat.edited", { count });
};

const EditedBadge: React.FC<{ message: Message }> = ({ message }) => {
  const [open, setOpen] = useState(false);
  const count = Number(message.editCount ?? 0);
  if (count <= 0 || message.isDeleted) return null;
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="absolute bottom-0 left-1.5 inline-flex items-center gap-1 text-[11px] italic text-[hsl(var(--text-muted))] hover:underline"
        title={tr("chat.edits.viewHistory")}
        aria-label={tr("chat.edits.viewHistory")}
        data-testid="edited-badge"
      >
        <Pencil className="h-3 w-3" aria-hidden="true" />
        {label(message.fromMe, count)}
      </button>
      {open && <MessageEditsDialog messageId={message.id} onClose={() => setOpen(false)} />}
    </>
  );
};

export default EditedBadge;
