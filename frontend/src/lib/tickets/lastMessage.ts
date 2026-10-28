import { i18n } from "../../translate/i18n";

/**
 * Prévia da última mensagem na lista de tickets. O backend manda o TEXTO real (`lastMessage`) quando existe, ou um
 * MARCADOR de tipo (`lastMessageType`: image, video, poll...) quando a mensagem não tem texto legível. O marcador é
 * traduzido aqui, pelo idioma de quem olha: o banco nunca guarda rótulo em um idioma.
 */
export function lastMessagePreview(ticket: { lastMessage?: string; lastMessageType?: string }): string {
  if (ticket.lastMessage) return ticket.lastMessage;
  const kind = ticket.lastMessageType;
  if (kind) {
    const key = `tickets.lastMessageKind.${kind}`;
    const t = String(i18n.t(key) ?? "");
    // tipo que ainda não tem tradução (canal novo): não mostra a chave crua, cai no genérico
    return t && t !== key ? t : String(i18n.t("tickets.lastMessageKind.other") ?? "");
  }
  return "";
}
