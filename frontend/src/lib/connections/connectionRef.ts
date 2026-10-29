// Uma conexão é identificada pelo par (origem, id): o WhatsApp vive em Whatsapps e os demais canais em
// ChannelConnections, e os ids podem repetir entre as duas tabelas (WhatsApp 1 e Telegram 1).
export type ConnectionSource = "whatsapp" | "channel";

export interface ConnectionRef {
  source: ConnectionSource;
  id: number;
}

export interface ConnectionOption extends ConnectionRef {
  kind: string; // chave do catálogo de canais ("whatsapp", "telegram"...)
  name: string;
  status: string;
}

/** Chave estável para select/lista: "whatsapp:1", "channel:1". */
export const connectionKey = (c: ConnectionRef): string => `${c.source}:${c.id}`;

export const parseConnectionKey = (key: string): ConnectionRef | null => {
  const [source, id] = key.split(":");
  const n = Number(id);
  return (source === "whatsapp" || source === "channel") && Number.isInteger(n) && n > 0 ? { source, id: n } : null;
};

interface RawWhatsapp {
  id: number;
  name: string;
  status: string;
}
interface RawChannel extends RawWhatsapp {
  kind: string;
}

/** Junta as duas listas da API (/whatsapp e /channels/connections) numa só, sem perder a origem de cada uma. */
export const toConnectionOptions = (whatsapps: RawWhatsapp[], channels: RawChannel[]): ConnectionOption[] => [
  ...whatsapps.map((w) => ({ source: "whatsapp" as const, id: w.id, kind: "whatsapp", name: w.name, status: w.status })),
  ...channels.map((c) => ({ source: "channel" as const, id: c.id, kind: c.kind, name: c.name, status: c.status })),
];
