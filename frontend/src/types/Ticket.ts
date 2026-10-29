export interface ExtraInfo {
  id?: number;
  name: string;
  value: string;
}

export interface Contact {
  id: number;
  name: string;
  number: string;
  profilePicUrl?: string;
  email?: string;
  isGroup?: boolean;
  /** WhatsApp linked-device identifier — present when contact is verified */
  lid?: string;
  extraInfo?: ExtraInfo[];
  client?: { id: number; socialName?: string | null } | null;
  /** Conexão de origem: o contato pertence a UMA conexão e é por ela que recebe e envia. */
  connectionSource?: "whatsapp" | "channel";
  connectionId?: number;
  /** Resultado de gravar o contato na agenda do celular (só WhatsApp): pending | saved | failed. */
  phoneSync?: "" | "pending" | "saved" | "failed";
  phoneSyncError?: string;
}

export interface Tag {
  id: number;
  name: string;
  color: string;
}

export interface Queue {
  id: number;
  name: string;
  color: string;
}

export interface Whatsapp {
  id: number;
  name: string;
  status?: string;
}

export interface ChannelCapabilities {
  initiateConversation?: boolean;
  replyWindow?: number | null;
  maxTextBytes?: number;
  media?: string[];
  groups?: boolean;
  typing?: boolean;
  readReceipts?: boolean;
  reactions?: boolean;
  hasPhone?: boolean;
  voice?: boolean;
  video?: boolean;
}

export interface Ticket {
  id: number;
  status: "open" | "pending" | "closed";
  unreadMessages: number;
  lastMessage: string;
  /** Marcador de tipo quando a última mensagem não tem texto (image, video, poll...); a tela o traduz. */
  lastMessageType?: string;
  isGroup: boolean;
  createdAt: string;
  updatedAt: string;
  contactId: number;
  contact: Contact;
  queueId?: number;
  queue?: Queue;
  whatsappId?: number;
  whatsapp?: Whatsapp;
  userId?: number;
  user?: {
    id: number;
    name: string;
  };
  tags?: Tag[];
  channel?: string;
  /** Identificação do ticket: canal + conexão, no mesmo formato para qualquer canal (ver TicketConnection). */
  connection?: TicketConnection;
  capabilities?: ChannelCapabilities;
  replyPolicy?: {
    canReply: boolean;
    reason?: string;
    message?: string;
  };
}

/**
 * Por onde um ticket conversa: o canal (`kind`, uma chave do catálogo em config/channels.ts) e a conexão (`id` + `name`).
 * É o contrato de identificação do ticket e vale para TODO canal (WhatsApp, Telegram, Mercado Livre, OLX, Instagram,
 * TikTok...): o backend o preenche a partir da conexão do ticket e a tela só o desenha, sem `if` por canal. Canal novo
 * entra no catálogo (config/channels.ts) e ganha o selo sem tocar nos componentes de ticket.
 */
export interface TicketConnection {
  kind: string;
  id: number;
  name: string;
  /** Tabela de origem do id ("whatsapp" | "channel"); some quando o WhatsApp migrar para ChannelConnections. */
  source: "whatsapp" | "channel";
  /** Foto da conexão (hoje só a do WhatsApp): avatar das mensagens enviadas pelo Watink. */
  profilePicUrl?: string;
}
