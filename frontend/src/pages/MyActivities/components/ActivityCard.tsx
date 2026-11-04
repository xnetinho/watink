import React, { useState } from "react";
import { formatDistanceToNow } from "date-fns";
import { dateLocale } from "../../../lib/dateLocale";
import { ArrowRight, Clock, ListChecks, MessageSquareWarning } from "lucide-react";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { StatusChip } from "@/components/ui/status-chip";
import { Button } from "@/components/ui/button";
import notify from "@/lib/notify";
import api from "../../../services/api";
import { Activity, ACTIVITY_STATUS_LABELS, ACTIVITY_PRIORITY_LABELS, NewOccurrence } from "../activityTypes";
import { timeToMinutes } from "../activityHelpers";
import OccurrenceModal from "./OccurrenceModal";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const STATUS_VARIANT: Record<string, "default" | "secondary" | "outline" | "destructive"> = {
  pending: "outline",
  in_progress: "secondary",
  done: "default",
  cancelled: "destructive",
};

const DEFAULT_OCCURRENCE: NewOccurrence = { description: "", type: "info", timeImpact: "" };

interface ActivityCardProps {
  activity: Activity;
  onExecute: (activity: Activity) => void;
  onOccurrenceRegistered: () => void;
}

// Card da listagem — badge de status, badge de SLA/parada, progresso do
// checklist, prazo, atalho tr("uiText.activities.recordOccurrence") sem entrar na execução
// (docs/agents/activities.md §Melhorias de UX).
const ActivityCard: React.FC<ActivityCardProps> = ({ activity, onExecute, onOccurrenceRegistered }) => {
  const [occurrenceModalOpen, setOccurrenceModalOpen] = useState(false);
  const [newOccurrence, setNewOccurrence] = useState<NewOccurrence>(DEFAULT_OCCURRENCE);

  const handleQuickOccurrence = async () => {
    if (!newOccurrence.description.trim()) return;
    try {
      await api.post(`/activities/${activity.id}/occurrences`, {
        description: newOccurrence.description.trim(),
        type: newOccurrence.type,
        timeImpact: timeToMinutes(newOccurrence.timeImpact),
      });
      notify.success(tr("uiText.activities.occurrenceRecorded"));
      setOccurrenceModalOpen(false);
      setNewOccurrence(DEFAULT_OCCURRENCE);
      onOccurrenceRegistered();
    } catch (err) {
      notify.error(err);
    }
  };

  const progress = activity.checklistProgress;
  const slaChip = activity.slaStatus === "overdue"
    ? { label: activity.slaDueAt ? tr("uiText.activities.overdueAgo", { time: formatDistanceToNow(new Date(activity.slaDueAt), { locale: dateLocale() }) }) : tr("uiText.activities.overdueLabel"), status: "error" as const }
    : activity.slaStatus === "atRisk"
      ? { label: tr("uiText.activities.dueIn", { time: activity.slaDueAt ? formatDistanceToNow(new Date(activity.slaDueAt), { locale: dateLocale() }) : "" }), status: "warning" as const }
      : null;

  return (
    <Card className="flex flex-col">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-2 flex-wrap">
          <Badge variant={STATUS_VARIANT[activity.status] ?? "outline"}>
            {ACTIVITY_STATUS_LABELS[activity.status]}
          </Badge>
          <Badge variant="outline">{ACTIVITY_PRIORITY_LABELS[activity.priority]}</Badge>
          <span className="text-xs text-muted-foreground ml-auto">#{activity.id}</span>
        </div>
        <CardTitle className="text-base leading-snug">{activity.title}</CardTitle>
        <CardDescription className="line-clamp-2">
          {activity.description || tr("uiText.activities.noDescription")}
        </CardDescription>
      </CardHeader>
      <CardContent className="mt-auto flex flex-col gap-3">
        <div className="flex flex-wrap gap-1.5">
          {activity.staleSince && (
            <StatusChip
              status="warning"
              size="sm"
              label={tr("uiText.activities.stalledAgo", { time: formatDistanceToNow(new Date(activity.staleSince), { locale: dateLocale() }) })}
            />
          )}
          {slaChip && <StatusChip status={slaChip.status} size="sm" label={slaChip.label} />}
        </div>

        {progress && progress.total > 0 && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <ListChecks className="h-3.5 w-3.5" />
            {progress.done}/{progress.total} itens
          </div>
        )}

        <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
          <Clock className="h-3.5 w-3.5" />
          {formatDistanceToNow(new Date(activity.createdAt), { locale: dateLocale(), addSuffix: true })}
        </div>

        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setOccurrenceModalOpen(true)}
            title={tr("uiText.activities.recordOccurrence")}
          >
            <MessageSquareWarning className="h-4 w-4" />
          </Button>
          <Button size="sm" className="flex-1" onClick={() => onExecute(activity)}>
            Executar
            <ArrowRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </CardContent>

      <OccurrenceModal
        open={occurrenceModalOpen}
        newOccurrence={newOccurrence}
        onChange={setNewOccurrence}
        onConfirm={handleQuickOccurrence}
        onCancel={() => setOccurrenceModalOpen(false)}
      />
    </Card>
  );
};

export default ActivityCard;
