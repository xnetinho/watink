export type SenderDevice = "ios" | "android" | "web" | "desktop";

/**
 * Estima o aparelho que enviou uma mensagem do WhatsApp pelo FORMATO do id dela. Não é um campo do protocolo: é uma previsão (o
 * WhatsApp não informa a plataforma), por isso a tela marca o resultado com "~".
 *
 * Regras e ordem copiadas de `getDevice` do Baileys (WhiskeySockets/Baileys, src/Utils/messages.ts, MIT); validadas em 7/nov/2026
 * com mensagens reais de um Android (32 hex), um iPhone (`3A` + 18) e um WhatsApp Web (`3E` + 20).
 * Devolve null quando o id não segue nenhum padrão (nunca chuta).
 */
export function senderDeviceFromId(id: unknown): SenderDevice | null {
  if (typeof id !== "string") return null;
  if (/^3A.{18}$/.test(id)) return "ios";
  if (/^3E.{20}$/.test(id)) return "web";
  if (/^(.{21}|.{32})$/.test(id)) return "android";
  if (/^(3F|.{18}$)/.test(id)) return "desktop";
  return null;
}
