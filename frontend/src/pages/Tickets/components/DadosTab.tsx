import React, { useState } from "react";
import { Users, User, Phone, Mail, AtSign, RefreshCw, UserPlus, Briefcase } from "lucide-react";
import { toast } from "react-toastify";
import { Avatar } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { getBackendUrl } from "../../../helpers/urlUtils";
import api from "../../../services/api";
import { TicketInfo } from "../ticketsTypes";
import PipelinesSection from "./PipelinesSection";
import FlowsSection from "./FlowsSection";
import HelpdeskSection from "./HelpdeskSection";
import TicketTagsSection from "./TicketTagsSection";
import ClientModal from "../../Clients/ClientModal";
import { TicketConnectionBadge } from "../../../components/TicketConnectionBadge";
import { i18n } from "../../../translate/i18n";

// i18n.t() deste projeto pode devolver null; a tela só quer string.
const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

interface DadosTabProps {
  ticket: TicketInfo | null;
  loading: boolean;
}

const DadosTab: React.FC<DadosTabProps> = ({ ticket, loading }) => {
  const [syncing, setSyncing] = useState(false);
  const [clientModalOpen, setClientModalOpen] = useState(false);
  const contact = ticket?.contact;
  const isGroup = contact?.isGroup || ticket?.isGroup;

  const handleSyncContact = async () => {
    if (!contact?.id) return;
    setSyncing(true);
    try {
      await api.post(`/contacts/${contact.id}/sync`);
      toast.success(tr("tickets.details.syncRequested"));
    } catch {
      toast.error(tr("tickets.details.syncError"));
    } finally {
      setSyncing(false);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col p-4 gap-3 animate-pulse">
        <div className="flex flex-col items-center gap-2 py-4">
          <div className="w-20 h-20 rounded-full bg-muted" />
          <div className="h-4 w-32 rounded bg-muted" />
          <div className="h-3 w-24 rounded bg-muted" />
        </div>
        <div className="h-3 w-full rounded bg-muted" />
        <div className="h-3 w-3/4 rounded bg-muted" />
      </div>
    );
  }

  if (!contact) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground text-sm p-4 text-center">
        {tr("tickets.details.selectTicketDetails")}
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      {/* Avatar + name */}
      <div className="flex flex-col items-center gap-2 px-4 py-6 border-b border-border">
        <Avatar
          src={contact.profilePicUrl ? getBackendUrl(contact.profilePicUrl) : null}
          name={contact.name || "?"}
          className="w-20 h-20"
          isGroup={!!isGroup}
        />
        <p className="font-semibold text-sm text-center leading-tight">{contact.name}</p>
        {(ticket?.connection || ticket?.channel) && (
          <div className="flex flex-col items-center gap-1" data-testid="details-connection">
            <span className="text-[0.6875rem] uppercase tracking-wide text-muted-foreground">
              {String(i18n.t("tickets.details.connection") ?? "")}
            </span>
            <TicketConnectionBadge channel={ticket?.channel} connection={ticket?.connection} size="md" />
          </div>
        )}
        {isGroup ? (
          <span className="inline-flex items-center gap-1 text-xs text-muted-foreground bg-muted rounded-full px-2 py-0.5">
            <Users className="w-3 h-3" />
            {tr("tickets.details.groupOfWhatsapp")}
          </span>
        ) : (
          <span className="inline-flex items-center gap-1 text-xs text-muted-foreground bg-muted rounded-full px-2 py-0.5">
            <User className="w-3 h-3" />
            {tr("uiText.common.contact")}
          </span>
        )}
        <Button
          variant="outline"
          size="sm"
          className="mt-1 h-7 gap-1.5 text-xs"
          disabled={syncing || !contact.id}
          onClick={handleSyncContact}
        >
          <RefreshCw className={`w-3 h-3 ${syncing ? "animate-spin" : ""}`} />
          {tr("tickets.details.syncContact")}
        </Button>
      </div>

      {/* Details */}
      <div className="flex flex-col gap-3 px-4 py-4">
        {ticket?.id && (
          <div className="flex flex-col gap-1.5">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{tr("tickets.details.sections.ticket")}</p>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">{tr("tickets.details.fields.number")}</span>
              <span className="font-medium">#{ticket.id}</span>
            </div>
            {ticket.user && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">{tr("tickets.details.fields.agent")}</span>
                <span className="font-medium">{ticket.user.name}</span>
              </div>
            )}
          </div>
        )}

        {ticket?.id && (
          <TicketTagsSection ticketId={ticket.id} />
        )}

        <div className="flex flex-col gap-1.5">
          <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
            {tr(isGroup ? "tickets.details.sections.group" : "tickets.details.sections.contact")}
          </p>
          {contact.number && (
            <div className="flex items-center gap-2 text-sm">
              <Phone className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
              <a href={`tel:${contact.number}`} className="text-primary hover:underline truncate">
                {contact.number}
              </a>
            </div>
          )}
          {contact.email && (
            <div className="flex items-center gap-2 text-sm">
              <Mail className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
              <a href={`mailto:${contact.email}`} className="text-primary hover:underline truncate">
                {contact.email}
              </a>
            </div>
          )}
          {contact.lid && !contact.number && (
            <div className="flex items-center gap-2 text-sm">
              <AtSign className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
              <span className="text-muted-foreground truncate">{contact.lid}</span>
            </div>
          )}
          {isGroup && !contact.number && !contact.email && (
            <p className="text-xs text-muted-foreground">
              {tr("tickets.details.groupHint")}
            </p>
          )}
        </div>

        {/* Cliente — somente para tickets individuais */}
        {!isGroup && contact?.id && (
          <div className="flex flex-col gap-1.5">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">{tr("tickets.details.sections.client")}</p>
            {contact.client ? (
              <div className="flex items-center gap-2 text-sm">
                <Briefcase className="w-3.5 h-3.5 text-muted-foreground shrink-0" />
                <span className="font-medium truncate">
                  {contact.client.socialName || tr("tickets.details.linkedClient")}
                </span>
              </div>
            ) : (
              <Button
                variant="outline"
                size="sm"
                className="h-8 gap-1.5 text-xs w-fit"
                onClick={() => setClientModalOpen(true)}
              >
                <UserPlus className="w-3.5 h-3.5" />
                {tr("tickets.details.registerClient")}
              </Button>
            )}
          </div>
        )}

        {/* Pipelines — somente para tickets individuais */}
        {!isGroup && ticket?.id && contact?.id && (
          <PipelinesSection
            ticketId={ticket.id}
            contactId={contact.id}
            contactName={contact.name}
          />
        )}

        {/* Fluxos — somente para tickets individuais */}
        {!isGroup && contact?.id && (
          <FlowsSection contactId={contact.id} />
        )}

        {/* Helpdesk — somente para tickets individuais, plugin "helpdesk" ativo */}
        {!isGroup && contact?.id && (
          <HelpdeskSection contactId={contact.id} contactName={contact.name} ticketId={ticket?.id} />
        )}
      </div>

      {contact && clientModalOpen && (
        <ClientModal
          open={clientModalOpen}
          onClose={() => setClientModalOpen(false)}
          initialContact={{
            id: String(contact.id),
            name: contact.name,
            number: contact.number,
            email: contact.email,
          }}
        />
      )}
    </div>
  );
};

export default DadosTab;
