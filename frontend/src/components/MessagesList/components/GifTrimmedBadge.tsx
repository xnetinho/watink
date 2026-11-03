import React from "react";
import { Scissors } from "lucide-react";
import { i18n } from "../../../translate/i18n";
import { Message } from "../types";
import { parseData } from "../utils/messageHelpers";

/** O GIF que enviamos era mais longo que o limite e foi cortado antes de ir ao destinatário? */
export const hasGifTrimmedBadge = (message: Message): boolean =>
  !message.isDeleted && message.fromMe === true && parseData(message.dataJson).gifTrimmed === true;

/**
 * Selo do rodapé à esquerda (mesmo lugar do "Editada" e do "Visualização única"): o operador precisa saber que o cliente viu
 * só o começo do GIF. Fica na mensagem, não num aviso que some.
 */
const GifTrimmedBadge: React.FC<{ message: Message }> = ({ message }) => {
  if (!hasGifTrimmedBadge(message)) return null;
  return (
    <span
      className="absolute bottom-0 left-1.5 inline-flex items-center gap-1 text-[11px] italic text-[hsl(var(--text-muted))]"
      title={String(i18n.t("chat.gifTrimmed.hint") ?? "")}
      data-testid="gif-trimmed-badge"
    >
      <Scissors className="h-3 w-3" aria-hidden="true" />
      {String(i18n.t("chat.gifTrimmed.badge") ?? "")}
    </span>
  );
};

export default GifTrimmedBadge;
