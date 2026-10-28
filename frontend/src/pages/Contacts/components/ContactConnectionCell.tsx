import React from "react";
import { AlertTriangle, Loader2 } from "lucide-react";
import { TicketConnectionBadge } from "../../../components/TicketConnectionBadge";
import { connectionKey, type ConnectionOption } from "../../../lib/connections/connectionRef";
import { i18n } from "../../../translate/i18n";
import { Contact } from "../contactsTypes";

const tr = (key: string): string => String(i18n.t(key) ?? "");

interface Props {
  contact: Contact;
  connections: ConnectionOption[];
}

/** De qual conexão o contato deriva e, no WhatsApp, se ele chegou à agenda do celular. */
const ContactConnectionCell: React.FC<Props> = ({ contact, connections }) => {
  const conn = connections.find((c) => connectionKey(c) === `${contact.connectionSource}:${contact.connectionId}`);
  return (
    <span className="inline-flex flex-wrap items-center gap-1.5" data-testid="contact-connection">
      {conn && <TicketConnectionBadge connection={{ kind: conn.kind, id: conn.id, name: conn.name, source: conn.source }} size="sm" />}
      {contact.phoneSync === "pending" && (
        <span className="inline-flex items-center gap-1 text-[0.6875rem] text-[hsl(var(--text-muted))]" title={tr("contacts.phoneSync.pending")}>
          <Loader2 className="h-3 w-3 animate-spin" />
          {tr("contacts.phoneSync.pending")}
        </span>
      )}
      {contact.phoneSync === "failed" && (
        <span
          className="inline-flex items-center gap-1 text-[0.6875rem] text-[hsl(var(--status-warning))]"
          title={tr(`contacts.phoneSync.errors.${contact.phoneSyncError}`) === `contacts.phoneSync.errors.${contact.phoneSyncError}` ? contact.phoneSyncError : tr(`contacts.phoneSync.errors.${contact.phoneSyncError}`)}
          data-testid="phone-sync-failed"
        >
          <AlertTriangle className="h-3 w-3" />
          {tr("contacts.phoneSync.failed")}
        </span>
      )}
    </span>
  );
};

export default ContactConnectionCell;
