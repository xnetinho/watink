/** Janela em que o WhatsApp aceita editar uma mensagem enviada (igual à do servidor: 20 minutos). */
export const EDIT_WINDOW_MS = 20 * 60 * 1000;

// Mídias cuja LEGENDA pode ser editada; o arquivo em si nunca.
const EDITABLE_MEDIA = new Set(["", "chat", "image", "video", "document"]);

interface EditableMessage {
  fromMe?: boolean;
  isDeleted?: boolean;
  ack?: number;
  mediaType?: string;
  createdAt?: string;
}

/**
 * Se a ação "Editar" deve aparecer no menu da mensagem. É só conveniência de tela: o servidor repete as mesmas regras e é
 * quem decide. Só WhatsApp (os outros canais editam por outra API), só mensagem NOSSA, que o servidor já aceitou (ack 1 a 4),
 * de texto ou com legenda, e dentro dos 20 minutos.
 */
export function canEdit(message: EditableMessage, channel: string | undefined, now: number = Date.now()): boolean {
  if (channel && channel !== "whatsapp") return false;
  if (!message.fromMe || message.isDeleted) return false;
  const ack = message.ack ?? 0;
  if (ack < 1 || ack > 4) return false;
  if (!EDITABLE_MEDIA.has(message.mediaType ?? "")) return false;
  const sent = message.createdAt ? Date.parse(message.createdAt) : NaN;
  if (!Number.isFinite(sent)) return false;
  return now - sent <= EDIT_WINDOW_MS;
}
