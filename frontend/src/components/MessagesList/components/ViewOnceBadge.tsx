import React from "react";
import { Eye } from "lucide-react";
import { i18n } from "../../../translate/i18n";
import { Message } from "../types";
import { parseData } from "../utils/messageHelpers";

/** O selo se aplica a esta mensagem? Recuperada (do contato) ou enviada por nós como visualização única. */
export const hasViewOnceBadge = (message: Message): boolean => {
  if (message.isDeleted) return false;
  const data = parseData(message.dataJson);
  return data.recoveredViewOnce === true || (message.fromMe === true && data.viewOnce === true);
};

/**
 * Selo do rodapé à esquerda (mesmo lugar do "Editada"). Duas origens:
 *  - recebida: o contato mandou como visualização única e o Watink a recuperou quando o celular principal respondeu;
 *  - enviada: o operador mandou como visualização única pelo painel de envio (a mídia fica guardada aqui).
 */
const ViewOnceBadge: React.FC<{ message: Message }> = ({ message }) => {
  if (!hasViewOnceBadge(message)) return null;
  const sent = message.fromMe === true && parseData(message.dataJson).viewOnce === true;
  const text = String(i18n.t(sent ? "chat.viewOnce.badgeSent" : "chat.viewOnce.badge") ?? "");
  return (
    <span
      className="absolute bottom-0 left-1.5 inline-flex items-center gap-1 text-[11px] italic text-[hsl(var(--text-muted))]"
      title={String(i18n.t(sent ? "chat.viewOnce.badgeSentHint" : "chat.viewOnce.badgeHint") ?? "")}
      data-testid="view-once-badge"
    >
      <Eye className="h-3 w-3" aria-hidden="true" />
      {text}
    </span>
  );
};

export default ViewOnceBadge;
