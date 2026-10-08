import React, { useState } from "react";
import { PhoneIncoming, PhoneOutgoing, Play, Trash2 } from "lucide-react";
import { format, parseISO } from "date-fns";
import { Button } from "@/components/ui/button";
import { DataTable, type DataTableColumn } from "@/components/ui/data-table";
import { StatusChip } from "@/components/ui/status-chip";
import { SignalBars } from "@/components/Calls/CallQualityPanel";
import api from "@/services/api";
import toastError from "@/errors/toastError";
import { t } from "@/lib/calls/t";
import { callTone } from "@/lib/calls/callMessage";
import { formatDuration } from "@/lib/calls/format";
import type { CallLogRow } from "../types";
import { formatMos, summaryLevel } from "../quality";

interface Props {
  rows: CallLogRow[];
  loading: boolean;
  error: boolean;
  onRetry: () => void;
  canDelete: boolean;
  onDelete: (row: CallLogRow) => void;
  contactName?: (row: CallLogRow) => string;
}

function startedLabel(iso: string): string {
  try {
    return format(parseISO(iso), "dd/MM/yyyy HH:mm");
  } catch {
    return iso;
  }
}

/** Player da gravação: a URL assinada só é pedida ao clicar (expira; a escuta é auditada). */
const RecordingCell: React.FC<{ row: CallLogRow; canDelete: boolean; onDelete: (r: CallLogRow) => void }> = ({
  row,
  canDelete,
  onDelete,
}) => {
  const [url, setUrl] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  if (row.recordingStatus === "failed") return <span className="text-xs text-status-error-text">{t("calls.message.recordingFailed")}</span>;
  if (row.recordingStatus === "deleted") return <span className="text-xs text-muted-foreground">{t("calls.message.recordingDeleted")}</span>;
  if (row.recordingStatus !== "ready") return <span className="text-muted-foreground">—</span>;

  const listen = async () => {
    setBusy(true);
    try {
      const { data } = await api.get<{ url: string }>(`/calls/${encodeURIComponent(row.callId)}/recording`);
      setUrl(data.url);
    } catch (err) {
      toastError(err);
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex items-center gap-1">
      {url ? (
        <audio controls src={url} className="h-8 w-44" data-testid="history-player" />
      ) : (
        <Button variant="ghost" size="sm" disabled={busy} onClick={() => void listen()} data-testid="history-listen">
          <Play />
          {t("calls.history.listen")}
        </Button>
      )}
      {canDelete && (
        <Button variant="destructive-ghost" size="icon" onClick={() => onDelete(row)} aria-label={t("calls.history.delete")} data-testid="history-delete">
          <Trash2 />
        </Button>
      )}
    </div>
  );
};

export const CallsTable: React.FC<Props> = ({ rows, loading, error, onRetry, canDelete, onDelete, contactName }) => {
  const columns: DataTableColumn<CallLogRow>[] = [
    {
      key: "direction",
      header: t("calls.history.direction"),
      cell: (r) => (
        <span className="inline-flex items-center gap-1.5 text-sm">
          {r.direction === "outgoing" ? <PhoneOutgoing className="h-4 w-4" /> : <PhoneIncoming className="h-4 w-4" />}
          {r.direction === "outgoing" ? t("calls.history.directionOutgoing") : t("calls.history.directionIncoming")}
        </span>
      ),
    },
    {
      key: "contact",
      header: t("calls.history.contact"),
      cell: (r) => contactName?.(r) || r.callerPn || r.peerJid.split("@")[0] || "—",
    },
    {
      key: "status",
      header: t("calls.history.status"),
      cell: (r) => <StatusChip status={callTone(r.status)} size="sm" label={t(`calls.status.${r.status}`)} />,
    },
    { key: "startedAt", header: t("calls.history.startedAt"), cell: (r) => startedLabel(r.startedAt) },
    {
      key: "duration",
      header: t("calls.history.duration"),
      className: "tabular-nums",
      cell: (r) => (r.status === "missed" || r.status === "rejected" ? "—" : formatDuration(r.durationSec)),
    },
    {
      key: "quality",
      header: t("calls.history.quality"),
      cell: (r) => {
        const level = summaryLevel(r);
        if (!level) return <span className="text-muted-foreground">—</span>;
        return (
          <span className="inline-flex items-center gap-2" data-testid="history-quality" data-level={level}>
            <SignalBars level={level} />
            <span className="text-xs tabular-nums text-muted-foreground" title={t("calls.quality.estimated")}>
              {formatMos(r.mosEstimated)}
            </span>
          </span>
        );
      },
    },
    {
      key: "recording",
      header: t("calls.history.recording"),
      cell: (r) => <RecordingCell row={r} canDelete={canDelete} onDelete={onDelete} />,
    },
  ];

  return (
    <DataTable
      columns={columns}
      data={rows}
      getRowKey={(r) => r.id}
      loading={loading}
      error={error}
      onRetry={onRetry}
      emptyTitle={t("calls.history.empty")}
    />
  );
};

export default CallsTable;
