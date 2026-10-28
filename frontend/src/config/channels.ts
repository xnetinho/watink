export type ChannelKind = "whatsapp" | "telegram" | "mercadolivre" | "olx" | "instagram" | "tiktok";

export type ChannelAvailability = "available" | "beta" | "coming_soon" | "requires_approval";

export interface ChannelDefinition {
  kind: ChannelKind;
  name: string;
  description: string;
  availability: ChannelAvailability;
  authKind: "qr" | "token" | "oauth" | "oauth_review";
  colorClass: string;
  bgClass: string;
  borderClass: string;
  iconName: string;
  hasPhone: boolean;
  /** A empresa pode cadastrar contato à mão? Falso quando só o cliente inicia a conversa (bot do Telegram, marketplaces). */
  canCreateContacts: boolean;
  replyWindowHours?: number;
}

export const CHANNELS_CATALOG: ChannelDefinition[] = [
  {
    kind: "whatsapp",
    name: "WhatsApp",
    description: "Mensagens, mídias e grupos pelo número oficial ou Whatsmeow.",
    availability: "available",
    authKind: "qr",
    colorClass: "text-emerald-600 dark:text-emerald-400",
    bgClass: "bg-emerald-500/15",
    borderClass: "border-emerald-500/30",
    iconName: "MessageCircle",
    hasPhone: true,
    canCreateContacts: true,
  },
  {
    kind: "telegram",
    name: "Telegram",
    description: "Conecte bots via token do @BotFather com envio de texto e mídias.",
    availability: "available",
    authKind: "token",
    colorClass: "text-sky-600 dark:text-sky-400",
    bgClass: "bg-sky-500/15",
    borderClass: "border-sky-500/30",
    iconName: "Send",
    hasPhone: false,
    canCreateContacts: false,
  },
  {
    kind: "mercadolivre",
    name: "Mercado Livre",
    description: "Atendimento pós-venda organizado por pedido (pack_id).",
    availability: "beta",
    authKind: "oauth",
    colorClass: "text-amber-600 dark:text-amber-400",
    bgClass: "bg-amber-500/15",
    borderClass: "border-amber-500/30",
    iconName: "ShoppingBag",
    hasPhone: false,
    canCreateContacts: false,
    replyWindowHours: 48,
  },
  {
    kind: "instagram",
    name: "Instagram Direct",
    description: "Responda a DMs e mensagens da sua página comercial do Instagram.",
    availability: "requires_approval",
    authKind: "oauth_review",
    colorClass: "text-pink-600 dark:text-pink-400",
    bgClass: "bg-pink-500/15",
    borderClass: "border-pink-500/30",
    iconName: "Instagram",
    hasPhone: false,
    canCreateContacts: false,
    replyWindowHours: 24,
  },
  {
    kind: "olx",
    name: "OLX Chat",
    description: "Converse com compradores e vendedores direto pelos anúncios da OLX.",
    availability: "coming_soon",
    authKind: "oauth",
    colorClass: "text-purple-600 dark:text-purple-400",
    bgClass: "bg-purple-500/15",
    borderClass: "border-purple-500/30",
    iconName: "Tag",
    hasPhone: true,
    canCreateContacts: false,
  },
  {
    kind: "tiktok",
    name: "TikTok Direct Messages",
    description: "Interaja com mensagens diretas recebidas na conta TikTok Business.",
    availability: "requires_approval",
    authKind: "oauth_review",
    colorClass: "text-foreground",
    bgClass: "bg-muted",
    borderClass: "border-border",
    iconName: "Video",
    hasPhone: false,
    canCreateContacts: false,
    replyWindowHours: 48,
  },
];

/** Canal usado quando o ticket não traz canal ou o canal não está no catálogo. */
export const CHANNEL_FALLBACK: ChannelDefinition = CHANNELS_CATALOG[0];

/** Definição do canal pela chave; canal desconhecido (ou ainda fora do catálogo) cai no genérico sem quebrar a tela. */
export const channelDefinition = (kind?: string): ChannelDefinition =>
  CHANNELS_CATALOG.find((c) => c.kind === kind) ??
  (kind
    ? {
        ...CHANNEL_FALLBACK,
        kind: kind as ChannelKind,
        name: kind,
        colorClass: "text-muted-foreground",
        bgClass: "bg-muted",
        borderClass: "border-border",
        iconName: "Globe",
        canCreateContacts: false, // canal que o catálogo não conhece: não promete cadastro manual
      }
    : CHANNEL_FALLBACK);
