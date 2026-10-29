import { parseISO } from "date-fns";
import { Message, QuotedMsg } from "../types";
import { i18n } from "../../../translate/i18n";

export const PARTICIPANT_COLORS = [
  "var(--status-warning)",
  "var(--status-error)",
  "var(--text-muted)",
  "var(--action-primary)",
  "var(--status-success)",
  "var(--status-info)",
];

export const parseData = (
  raw: string | Record<string, unknown> | undefined
): Record<string, unknown> => {
  if (!raw) return {};
  if (typeof raw === "string") {
    try {
      return JSON.parse(raw) as Record<string, unknown>;
    } catch {
      return {};
    }
  }
  return raw;
};

export const isDateValid = (dateStr?: string): boolean => {
  if (!dateStr) return false;
  const date = parseISO(dateStr);
  return !isNaN(date.getTime()) && date.getFullYear() > 2000;
};

export const getSenderKey = (
  m: Message | undefined,
  isGroup?: boolean
): string => {
  if (!m) return "unknown";
  if (m.fromMe) return "me";
  if (isGroup) {
    const d = parseData(m.dataJson);
    return (
      (m.participant as string) ||
      (d.participant as string) ||
      (d.senderLid as string) ||
      (d.pushName as string) ||
      "unknown"
    );
  }
  return "other";
};

// Tipos que o chat sabe desenhar. Qualquer outro mediaType vira o aviso de "mensagem não suportada".
const KNOWN_MEDIA_TYPES = [
  "chat", "image", "video", "audio", "document", "sticker", "location", "vcard", "carousel", "call", "view_once", "poll", "event", "pix",
];
export const isKnownMediaType = (t: string): boolean => KNOWN_MEDIA_TYPES.includes(t);

const tr = (key: string): string => String(i18n.t(key) ?? "");

export const getMessageBody = (message: Message | QuotedMsg): string => {
  if (message.mediaType === "unsupported") return tr("chat.previews.unsupported");
  if (message.mediaType === "location") return tr("chat.previews.location");
  if (message.mediaType === "vcard") return tr("chat.previews.contact");
  if (message.mediaType === "carousel") return tr("chat.previews.carousel");
  return message.body ?? "";
};

export interface QuotePreview {
  /** Texto da citação: a legenda/corpo, ou o rótulo do tipo quando a mídia não tem texto. */
  label: string;
  /** Voz ou arquivo de áudio: o balão mostra o microfone/nota e a duração, como o celular. */
  audio?: { voice: boolean; seconds?: number };
  /** Miniatura JPEG em base64 (imagem/vídeo recebidos), para a caixa mostrar a foto. */
  thumbnail?: string;
}

/** O que a caixa de citação (no balão e na barra de resposta) mostra para a mensagem citada. */
export const getQuotePreview = (quoted: Pick<QuotedMsg, "body" | "mediaType" | "dataJson">): QuotePreview => {
  const data = parseData(quoted.dataJson);
  const rich = (data.rich ?? {}) as { voice?: boolean; seconds?: number };
  const type = quoted.mediaType ?? "";
  const body = getMessageBody(quoted as QuotedMsg);
  if (type === "audio") {
    const voice = data.voice === true || rich.voice === true;
    return { label: body || tr(voice ? "chat.previews.voice" : "chat.previews.audio"), audio: { voice, seconds: rich.seconds } };
  }
  const thumbnail = typeof data.thumbnail === "string" && data.thumbnail ? data.thumbnail : undefined;
  if (body) return { label: body, thumbnail };
  const kind = tr(`tickets.lastMessageKind.${type}`);
  return { label: kind && !kind.startsWith("tickets.") ? kind : "", thumbnail };
};

export const getFileNameFromUrl = (url?: string): string => {
  if (!url) return "";
  try {
    const urlObj = new URL(url);
    const pathname = urlObj.pathname;
    return pathname.substring(pathname.lastIndexOf("/") + 1);
  } catch {
    return url;
  }
};

export const getParticipantColor = (
  message: Message,
  colorCache: Map<string, string>
): string => {
  const participantId =
    (message.participant as string) ||
    (parseData(message.dataJson).participant as string) ||
    "unknown";
  if (!colorCache.has(participantId)) {
    const color =
      PARTICIPANT_COLORS[
        Math.floor(Math.random() * PARTICIPANT_COLORS.length)
      ];
    colorCache.set(participantId, color);
  }
  return colorCache.get(participantId)!;
};
