import React from "react";
import {
  Plus,
  RefreshCw,
  Edit,
  MessageSquare,
  CheckCircle,
  Paperclip,
} from "lucide-react";
import { format } from "date-fns";
import { dateLocale } from "../../../lib/dateLocale";
import PaperCard from "../../../components/PaperCard";
import type { HistoryItem } from "../protocolTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const ACTION_ICONS: Record<string, React.ReactNode> = {
  created: <Plus className="h-4 w-4" />,
  status_changed: <RefreshCw className="h-4 w-4" />,
  priority_changed: <Edit className="h-4 w-4" />,
  commented: <MessageSquare className="h-4 w-4" />,
  resolved: <CheckCircle className="h-4 w-4" />,
  attachment: <Paperclip className="h-4 w-4" />,
};

const ACTION_LABELS: Record<string, string> = {
  created: "uiText.helpdesk.action.created",
  status_changed: "uiText.helpdesk.action.status_changed",
  priority_changed: "uiText.helpdesk.action.priority_changed",
  commented: "uiText.helpdesk.action.commented",
  resolved: "uiText.helpdesk.action.resolved",
  closed: "uiText.helpdesk.action.closed",
  attachment: "uiText.helpdesk.action.attachment",
};

const getActionIcon = (action: string): React.ReactNode =>
  ACTION_ICONS[action] ?? <RefreshCw className="h-4 w-4" />;

const getActionLabel = (action: string): string =>
  ACTION_LABELS[action] ? tr(ACTION_LABELS[action]) : action;

interface ProtocolHistoryCardProps {
  history: HistoryItem[];
}

const ProtocolHistoryCard: React.FC<ProtocolHistoryCardProps> = ({ history }) => (
  <PaperCard>
    <h2 className="mb-4 text-base font-semibold">{tr("uiText.helpdesk.history")}</h2>
    {history.length === 0 ? (
      <p className="text-center text-sm text-muted-foreground">
        {tr("uiText.helpdesk.noHistory")}
      </p>
    ) : (
      <ul className="flex flex-col gap-3">
        {history.map((item, index) => (
          <li
            key={index}
            className="ml-4 border-l-2 border-[var(--primary-main)] pl-3"
          >
            <div className="flex items-center gap-1.5">
              <span className="text-muted-foreground">{getActionIcon(item.action)}</span>
              <span className="text-sm font-semibold">{getActionLabel(item.action)}</span>
            </div>
            {item.previousValue && item.newValue && (
              <p className="mt-0.5 text-xs text-muted-foreground">
                {item.previousValue} → {item.newValue}
              </p>
            )}
            {item.comment && (
              <p className="mt-1 text-xs italic">"{item.comment}"</p>
            )}
            <div className="mt-1">
              <p className="text-xs text-muted-foreground">
                {item.user?.name || "Sistema"}
              </p>
              <p className="text-xs text-muted-foreground">
                {format(new Date(item.createdAt), "dd/MM/yyyy HH:mm", { locale: dateLocale() })}
              </p>
            </div>
          </li>
        ))}
      </ul>
    )}
  </PaperCard>
);

export default ProtocolHistoryCard;
