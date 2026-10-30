import React from "react";
import StatCard from "./StatCard";
import { SystemStats } from "../types";
import { formatBytes, formatUptime } from "../utils";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface SystemStatsGridProps {
  stats: SystemStats | null;
}

const SystemStatsGrid: React.FC<SystemStatsGridProps> = ({ stats }) => {
  const memPct = stats?.memoryTotal
    ? ((stats.memoryUsed || 0) / stats.memoryTotal) * 100
    : 0;

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
      <StatCard
        title={tr("uiText.versionDashboard.cpu")}
        value={`${stats?.cpuUsage?.toFixed(1)}%`}
        progress={stats?.cpuUsage || 0}
        progressColor={(stats?.cpuUsage || 0) > 80 ? "destructive" : "primary"}
      />
      <StatCard
        title={tr("uiText.versionDashboard.ram")}
        value={`${formatBytes(stats?.memoryUsed || 0)} / ${formatBytes(stats?.memoryTotal || 0)}`}
        progress={memPct}
        progressColor={memPct > 80 ? "destructive" : "primary"}
        caption={tr("uiText.versionDashboard.memInUse", { pct: memPct.toFixed(1) })}
      />
      <StatCard
        title={tr("uiText.versionDashboard.uptime")}
        value={formatUptime(stats?.uptime || 0)}
        caption={`Desde: ${new Date(
          ((stats?.timestamp || 0) - (stats?.uptime || 0)) * 1000
        ).toLocaleString()}`}
      />
    </div>
  );
};

export default SystemStatsGrid;
