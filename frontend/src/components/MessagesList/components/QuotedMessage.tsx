import React from "react";
import clsx from "clsx";
import { Mic, Music } from "lucide-react";
import { parseData, getQuotePreview } from "../utils/messageHelpers";
import { formatDuration } from "../../../lib/audio/format";
import { Message } from "../types";

interface Props {
  message: Message;
  isGroup?: boolean;
}

/** Conteúdo da citação: ícone e duração no áudio (como o celular), miniatura na foto/vídeo, o texto nos demais. */
export const QuoteContent: React.FC<{ quoted: Parameters<typeof getQuotePreview>[0] }> = ({ quoted }) => {
  const p = getQuotePreview(quoted);
  return (
    <div className="flex items-center gap-2" data-testid="quote-content">
      {p.audio &&
        (p.audio.voice ? <Mic size={14} aria-hidden="true" className="shrink-0" /> : <Music size={14} aria-hidden="true" className="shrink-0" />)}
      <span className="min-w-0 flex-1 break-words">{p.label}</span>
      {p.audio?.seconds ? <span className="shrink-0 tabular-nums text-xs">{formatDuration(p.audio.seconds)}</span> : null}
      {p.thumbnail && (
        <img src={`data:image/jpeg;base64,${p.thumbnail}`} alt="" className="h-10 w-10 shrink-0 rounded object-cover" data-testid="quote-thumb" />
      )}
    </div>
  );
};

const QuotedMessage: React.FC<Props> = ({ message, isGroup }) => {
  const quoted = message.quotedMsg;
  if (!quoted) return null;

  const getQuotedSenderName = (): string | undefined => {
    const contact = quoted.contact;
    if (isGroup && contact?.isGroup && quoted.participant) {
      const d = parseData(quoted.dataJson);
      if (d?.pushName) return d.pushName as string;
      return (quoted.participant as string).replace(/@[^@]+$/, "");
    }
    return contact?.name;
  };

  let pushName: string | null = null;
  let participantNumber: string | null = null;
  if (quoted.dataJson) {
    try {
      const data = parseData(quoted.dataJson);
      pushName = data.pushName as string | null;
    } catch {
      /* invalid dataJson */
    }
  }
  if (quoted.participant) {
    participantNumber = (quoted.participant as string).replace(/\D/g, "");
  }
  const quotedSender = pushName
    ? `~${pushName}`
    : participantNumber
    ? `~${participantNumber}`
    : getQuotedSenderName();

  return (
    <div
      className={clsx(
        "-mt-0.5 -mr-20 mb-1.5 -ml-1.5 overflow-hidden bg-[hsl(var(--message-quote-bg))] rounded-[7.5px] flex relative",
        { "-mr-20": !message.fromMe, "overflow-y-hidden": message.fromMe }
      )}
    >
      <span
        className={clsx("flex-none w-1", {
          "bg-[hsl(var(--message-quote-side-left))]": !quoted.fromMe,
          "bg-[hsl(var(--message-quote-side-right))]": quoted.fromMe,
        })}
      />
      <div className="p-2.5 max-w-[300px] h-auto whitespace-pre-wrap">
        {!quoted.fromMe && (
          <span className="flex text-[hsl(var(--message-quote-side-left))] font-medium">
            {quotedSender}
          </span>
        )}
        <QuoteContent quoted={quoted} />
      </div>
    </div>
  );
};

export default QuotedMessage;
