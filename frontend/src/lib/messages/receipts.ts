import api from "../../services/api";

/** Dados de entrega de UM participante. Conversa individual = uma linha só (senderJid vazio); grupo = uma por membro. */
export interface MessageReceiptRow {
  senderJid: string;
  /** Nome do contato quando o participante casa com um contato da empresa; vazio senão (a tela mostra o número). */
  name: string;
  /** Instantes ISO-8601 segundo o WhatsApp; null = ainda não. */
  delivered: string | null;
  read: string | null;
  played: string | null;
}

/** Busca os dados de entrega e leitura de uma mensagem nossa. */
export function fetchMessageReceipts(messageId: string | number): Promise<MessageReceiptRow[]> {
  return api.get<{ receipts: MessageReceiptRow[] }>(`/message/${messageId}/receipts`).then(({ data }) => data.receipts ?? []);
}

/**
 * O item "Dados da mensagem" só faz sentido em mensagem NOSSA que o canal já confirmou (ack 1+) e que tem recibo por
 * mensagem: hoje só o WhatsApp (o Telegram não informa quando o contato recebeu ou leu).
 */
export function canShowReceipts(m: { fromMe?: boolean; isDeleted?: boolean; ack?: number }, channel?: string): boolean {
  if (!m.fromMe || m.isDeleted) return false;
  if (channel && channel !== "whatsapp") return false;
  return typeof m.ack === "number" && m.ack >= 1 && m.ack <= 4;
}
