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
  capabilities?: ChannelCapabilities;
  replyPolicy?: {
    canReply: boolean;
    reason?: string;
    message?: string;
  };
}
