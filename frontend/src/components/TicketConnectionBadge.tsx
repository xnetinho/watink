import React from "react";
import { cn } from "../lib/utils";
import { CHANNEL_FALLBACK, channelDefinition } from "../config/channels";
import { channelIcon } from "./ChannelBadge";
import type { TicketConnection } from "../types/Ticket";

interface Props {
  /** Canal do ticket (chave do catálogo). Usado quando a conexão não veio (ex.: foi apagada). */
  channel?: string;
  connection?: TicketConnection | null;
  size?: "sm" | "md";
  className?: string;
}

/**
 * Selo de identificação do ticket: ícone do canal + nome da conexão ("☎ Comercial", "✈ Jarvis"). Um só componente para
 * todo canal: o ícone e a cor vêm do catálogo (config/channels.ts) e o nome, da conexão. Duas conexões do mesmo canal se
 * distinguem pelo nome; a cor é só por canal. Sem conexão (apagada), mostra o nome do canal.
 */
export const TicketConnectionBadge: React.FC<Props> = ({ channel, connection, size = "sm", className }) => {
  const kind = connection?.kind ?? channel ?? CHANNEL_FALLBACK.kind;
  const def = channelDefinition(kind);
  const label = connection?.name?.trim() || def.name;
  return (
    <span
      data-testid="ticket-connection-badge"
      data-channel={def.kind}
      title={connection?.name ? `${def.name} · ${connection.name}` : def.name}
      className={cn(
        "inline-flex max-w-full items-center gap-1 rounded-full border px-1.5 py-px font-medium",
        size === "sm" ? "text-[0.625rem]" : "text-xs",
        def.bgClass,
        def.colorClass,
        def.borderClass,
        className,
      )}
    >
      <span className="shrink-0">{channelIcon(def.kind, size === "sm" ? 11 : 14)}</span>
      <span className="truncate">{label}</span>
    </span>
  );
};

export default TicketConnectionBadge;
