import React from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "../../../components/ui/card";
import TicketsInfo from "../../../components/Dashboard/Widgets/TicketsInfo";
import PerformanceMetrics from "../../../components/Dashboard/Widgets/PerformanceMetrics";
import type { WidgetConfig } from "../dashboardTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface DashboardWidgetsProps {
  sortedWidgets: WidgetConfig[];
  userQueueIds: number[];
}

const DashboardWidgets: React.FC<DashboardWidgetsProps> = ({
  sortedWidgets,
  userQueueIds,
}) => {
  return (
    <>
      {sortedWidgets.map((widget) => {
        if (!widget.visible) return null;

        switch (widget.id) {
          case "performance_metrics":
            return (
              <Card key={widget.id}>
                <CardHeader>
                  <CardTitle>Performance por Equipe</CardTitle>
                  <CardDescription>
                    {tr("uiText.dashboard.teamMetrics")}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <PerformanceMetrics />
                </CardContent>
              </Card>
            );
          case "tickets_info":
            return (
              <Card key={widget.id}>
                <CardHeader>
                  <CardTitle>{tr("uiText.dashboard.ticketSummary")}</CardTitle>
                  <CardDescription>
                    {tr("uiText.dashboard.ticketSummaryHint")}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <TicketsInfo userQueueIds={userQueueIds} />
                </CardContent>
              </Card>
            );
          case "attendance_chart":
            // Já renderizado na linha principal — ocultar aqui
            return null;
          default:
            return null;
        }
      })}
    </>
  );
};

export default DashboardWidgets;
