import React from "react";
import { Forward } from "lucide-react";
import { parseData } from "../utils/messageHelpers";
import { i18n } from "../../../translate/i18n";
import { Message } from "../types";

// Acima de 4 encaminhamentos o WhatsApp rotula "encaminhada com frequência".
export const FREQUENT_FORWARD_SCORE = 4;

const ForwardedLabel: React.FC<{ message: Message }> = ({ message }) => {
  const d = parseData(message.dataJson);
  if (d.forwarded !== true) return null;
  const frequent = Number(d.forwardScore) > FREQUENT_FORWARD_SCORE;
  return (
    <div className="mb-0.5 flex items-center gap-1 text-[11px] italic text-[var(--text-muted)]" data-testid="forwarded-label">
      <Forward className="h-3 w-3" />
      {String(i18n.t(frequent ? "chat.forwardedMany" : "chat.forwarded") ?? "")}
    </div>
  );
};

export default ForwardedLabel;
