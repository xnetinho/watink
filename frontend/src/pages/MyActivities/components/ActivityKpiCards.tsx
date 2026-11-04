import React from "react";
import { CalendarClock, Loader2, AlertTriangle, CheckCircle2, Timer } from "lucide-react";
import { MetricCard } from "@/components/ui/metric-card";
import { ActivityKpis } from "../hooks/useMyActivities";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface ActivityKpiCardsProps {
  kpis: ActivityKpis;
}

// Grid canônico de KPI do produto — mesmo padrão de DashboardKpiRow.tsx.
// A Central "Grupos WhatsApp" NÃO tem KPIs; este bloco segue o Dashboard,
// não Grupos (ver docs/frontend/activities/OVERVIEW.md).
const ActivityKpiCards: React.FC<ActivityKpiCardsProps> = ({ kpis }) => (
  <div className="grid gap-4 grid-cols-2 md:grid-cols-5">
    <MetricCard label={tr("uiText.activities.kpiToday")} value={kpis.today} icon={<CalendarClock />} color="info" />
    <MetricCard label={tr("uiText.activities.kpiInProgress")} value={kpis.inProgress} icon={<Loader2 />} color="primary" />
    <MetricCard label={tr("uiText.activities.kpiOverdue")} value={kpis.overdue} icon={<AlertTriangle />} color="error" />
    <MetricCard label={tr("uiText.activities.kpiDoneWeek")} value={kpis.completedThisWeek} icon={<CheckCircle2 />} color="success" />
    <MetricCard
      label={tr("uiText.activities.kpiAvg")}
      value={kpis.avgExecutionMinutes > 0 ? `${Math.round(kpis.avgExecutionMinutes)}min` : "—"}
      icon={<Timer />}
      color="warning"
    />
  </div>
);

export default ActivityKpiCards;
