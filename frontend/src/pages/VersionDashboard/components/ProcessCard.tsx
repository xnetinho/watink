import React from "react";
import { SystemStats } from "../types";
import { formatBytes } from "../utils";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface ProcessCardProps {
  stats: SystemStats | null;
}

const ProcessCard: React.FC<ProcessCardProps> = ({ stats }) => {
  const items = [
    {
      label: tr("uiText.versionDashboard.cpuProc"),
      value: `${stats?.process?.cpuUsage?.toFixed(2)}%`,
    },
    {
      label: tr("uiText.versionDashboard.heap"),
      value: formatBytes(stats?.process?.memoryUsed || 0),
    },
    {
      label: "Goroutines",
      value: String(stats?.process?.numGoroutine ?? "-"),
    },
    { label: tr("uiText.versionDashboard.threads"), value: "-" },
  ];

  return (
    <div className="rounded-xl border border-border bg-card p-4 shadow-sm">
      <h2 className="mb-3 font-semibold">Processo Backend (Go)</h2>
      <div className="grid grid-cols-2 gap-3">
        {items.map((item) => (
          <div key={item.label}>
            <p className="text-xs text-muted-foreground">{item.label}</p>
            <p className="text-base font-bold">{item.value}</p>
          </div>
        ))}
      </div>
    </div>
  );
};

export default ProcessCard;
