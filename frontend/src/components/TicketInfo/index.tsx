import React from "react";

import { Avatar } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";
import { i18n } from "../../translate/i18n";
import { getBackendUrl } from "../../helpers/urlUtils";
import { getContactDisplayName } from "../../utils/clientDisplayName";
import { useContactPresence } from "../../hooks/useContactPresence";
import { presenceLabel } from "../../lib/presence/format";
import { TicketConnectionBadge } from "../TicketConnectionBadge";
import { Emojified } from "../Emojified";
import type { TicketConnection } from "../../types/Ticket";

interface Contact {
  name: string;
  number?: string;
  lid?: string | null;
  profilePicUrl?: string | null;
  isGroup?: boolean;
  client?: { id: number; socialName?: string | null } | null;
}

interface TicketUser {
  name: string;
}

interface Ticket {
  id: number;
  connection?: TicketConnection;
  /** "whatsapp" | "telegram" | ...: só o WhatsApp informa presença */
  channel?: string;
  user?: TicketUser | null;
}

interface TicketInfoProps {
  contact: Contact;
  ticket: Ticket;
  onClick?: React.MouseEventHandler<HTMLDivElement>;
}

const TicketInfo: React.FC<TicketInfoProps> = ({ contact, ticket, onClick }) => {
  const presence = useContactPresence(ticket.id, [contact.number, contact.lid], ticket.channel === "whatsapp" && !contact.isGroup);
  const presenceText = presenceLabel(presence, (k, o) => String(i18n.t(k, o ?? {}) ?? ""), i18n.language || "pt-BR");
  const assignedTo =
    ticket.user
      ? `${i18n.t("messagesList.header.assignedTo")} ${ticket.user.name}`
      : null;

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if ((e.key === "Enter" || e.key === " ") && onClick) {
          onClick(e as unknown as React.MouseEvent<HTMLDivElement>);
        }
      }}
      className={cn(
        "flex min-w-0 flex-1 cursor-pointer items-center gap-3 px-3 py-2"
      )}
    >
      <Avatar
        src={contact.profilePicUrl ? getBackendUrl(contact.profilePicUrl) : null}
        name={getContactDisplayName(contact)}
        size="md"
        isGroup={contact.isGroup}
        aria-label={getContactDisplayName(contact)}
      />

      <div className="flex min-w-0 flex-col">
        <span className="truncate text-sm font-semibold leading-tight text-foreground">
          <Emojified>{getContactDisplayName(contact)}</Emojified> #{ticket.id}
        </span>
        {(ticket.connection || ticket.channel) && (
          <TicketConnectionBadge channel={ticket.channel} connection={ticket.connection} size="sm" className="self-start" />
        )}
        {presenceText && (
          <span className="truncate text-xs text-muted-foreground" data-testid="contact-presence">
            {presenceText}
          </span>
        )}
        {assignedTo && (
          <span className="truncate text-xs text-muted-foreground">
            {assignedTo}
          </span>
        )}
      </div>
    </div>
  );
};

export default TicketInfo;
