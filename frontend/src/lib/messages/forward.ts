// Espelha forwardableMedia do backend (message_forward.go): só o que o envio sabe reproduzir. Enquete, localização,
// contato, PIX, chamada e visualização única são cartões do canal de origem e não têm envio equivalente.
const FORWARDABLE = new Set(["", "chat", "image", "video", "audio", "document", "sticker"]);

interface Forwardable {
  id?: number | string;
  mediaType?: unknown;
  mediaUrl?: unknown;
  isDeleted?: unknown;
}

/** A mensagem pode ser encaminhada? Mídia ainda não baixada não tem arquivo para reenviar. */
export function canForward(m: Forwardable): boolean {
  if (m.id === undefined || m.isDeleted === true) return false;
  const type = typeof m.mediaType === "string" ? m.mediaType : "";
  if (!FORWARDABLE.has(type)) return false;
  return type === "" || type === "chat" || (typeof m.mediaUrl === "string" && m.mediaUrl !== "");
}
