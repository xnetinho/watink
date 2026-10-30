import React from "react";
import { Eye } from "lucide-react";
import { i18n } from "../../../translate/i18n";
import { Message } from "../types";
import { parseData } from "../utils/messageHelpers";

/**
 * Selo do rodapé à esquerda (mesmo lugar do "Editada"): a mídia foi enviada como visualização única e o Watink a recuperou
 * quando o celular principal respondeu. Só aparece nas mensagens recuperadas.
 */
const ViewOnceBadge: React.FC<{ message: Message }> = ({ message }) => {
  if (message.isDeleted || parseData(message.dataJson).recoveredViewOnce !== true) return null;
  const text = String(i18n.t("chat.viewOnce.badge") ?? "");
  return (
    <span
      className="absolute bottom-0 left-1.5 inline-flex items-center gap-1 text-[11px] italic text-[hsl(var(--text-muted))]"
      title={String(i18n.t("chat.viewOnce.badgeHint") ?? "")}
      data-testid="view-once-badge"
    >
      <Eye className="h-3 w-3" aria-hidden="true" />
      {text}
    </span>
  );
};

export default ViewOnceBadge;
