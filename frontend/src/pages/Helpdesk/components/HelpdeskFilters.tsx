import React from "react";
import { Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { HelpdeskFiltersState } from "../helpdeskTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface HelpdeskFiltersProps {
  filters: HelpdeskFiltersState;
  onSearchChange: (value: string) => void;
  onStatusChange: (value: string) => void;
  onPriorityChange: (value: string) => void;
}

const HelpdeskFilters: React.FC<HelpdeskFiltersProps> = ({
  filters,
  onSearchChange,
  onStatusChange,
  onPriorityChange,
}) => {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
      <div className="relative">
        <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
        <Input
          placeholder={tr("uiText.helpdesk.searchPlaceholder")}
          className="pl-9"
          value={filters.searchParam}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>

      <Select value={filters.statusFilter} onValueChange={onStatusChange}>
        <SelectTrigger>
          <SelectValue placeholder={tr("uiText.common.status")} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">{tr("uiText.helpdesk.allStatuses")}</SelectItem>
          <SelectItem value="open">{tr("uiText.helpdesk.status.open")}</SelectItem>
          <SelectItem value="in_progress">{tr("uiText.helpdesk.status.in_progress")}</SelectItem>
          <SelectItem value="pending">{tr("uiText.helpdesk.status.pending")}</SelectItem>
          <SelectItem value="resolved">{tr("uiText.helpdesk.status.resolved")}</SelectItem>
          <SelectItem value="closed">{tr("uiText.helpdesk.status.closed")}</SelectItem>
        </SelectContent>
      </Select>

      <Select value={filters.priorityFilter} onValueChange={onPriorityChange}>
        <SelectTrigger>
          <SelectValue placeholder={tr("uiText.common.priority")} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Todas as prioridades</SelectItem>
          <SelectItem value="low">{tr("uiText.helpdesk.priority.low")}</SelectItem>
          <SelectItem value="medium">{tr("uiText.helpdesk.priority.medium")}</SelectItem>
          <SelectItem value="high">{tr("uiText.helpdesk.priority.high")}</SelectItem>
          <SelectItem value="urgent">{tr("uiText.helpdesk.priority.urgent")}</SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
};

export default HelpdeskFilters;
