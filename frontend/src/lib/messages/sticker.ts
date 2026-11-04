interface StickerSource {
  id?: number | string;
  mediaType?: unknown;
  mediaUrl?: unknown;
  isDeleted?: unknown;
}

/**
 * A mensagem pode ir para a biblioteca de figurinhas? Só figurinha com arquivo guardado, e só no WhatsApp (a biblioteca é do
 * WhatsApp; os outros canais têm outro mecanismo de figurinha). Sem canal informado vale o padrão histórico (WhatsApp).
 */
export function canSaveSticker(m: StickerSource, channel?: string): boolean {
  if (m.id === undefined || m.isDeleted === true) return false;
  if (channel && channel !== "whatsapp") return false;
  return m.mediaType === "sticker" && typeof m.mediaUrl === "string" && m.mediaUrl !== "";
}
