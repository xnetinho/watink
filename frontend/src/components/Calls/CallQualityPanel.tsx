import React, { useState } from "react";
import { Activity, AlertTriangle, ChevronDown, ChevronUp, Info } from "lucide-react";
import { cn } from "@/lib/utils";
import { t } from "@/lib/calls/t";
import type { CallQuality } from "@/context/Calls/types";
import { formatBitrate, levelPercent } from "@/lib/calls/format";

const LEVEL_KEY: Record<1 | 2 | 3, string> = { 3: "good", 2: "fair", 1: "poor" };
const LEVEL_TONE: Record<1 | 2 | 3, string> = {
  3: "text-status-success-text",
  2: "text-status-warning-text",
  1: "text-status-error-text",
};

/** Três barras de sinal; as acesas dependem do nível. */
export const SignalBars: React.FC<{ level: 1 | 2 | 3 }> = ({ level }) => (
  <span className="inline-flex items-end gap-0.5" role="img" aria-label={t(`calls.quality.${LEVEL_KEY[level]}`)} data-level={level}>
    {[1, 2, 3].map((i) => (
      <span
        key={i}
        className={cn("w-1 rounded-sm", i <= level ? "bg-current" : "bg-muted")}
        style={{ height: `${6 + i * 4}px` }}
      />
    ))}
  </span>
);

const Row: React.FC<{ label: string; value: React.ReactNode }> = ({ label, value }) => (
  <div className="flex items-center justify-between gap-4 text-xs">
    <span className="text-muted-foreground">{label}</span>
    <span className="font-medium tabular-nums">{value}</span>
  </div>
);

const LevelBar: React.FC<{ label: string; level: number | undefined }> = ({ label, level }) => (
  <div className="space-y-1">
    <span className="text-xs text-muted-foreground">{label}</span>
    <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${levelPercent(level)}%` }} />
    </div>
  </div>
);

/**
 * Indicador de sinal sempre visível e painel de detalhes expansível, atualizado a
 * cada segundo pela telemetria. O índice é sempre rotulado "estimado" e a perda
 * declara o que mede (só o áudio recebido).
 */
const CallQualityPanel: React.FC<{ quality: CallQuality | null; extraDelayMs?: number }> = ({ quality, extraDelayMs = 0 }) => {
  const [open, setOpen] = useState(false);
  if (!quality) return null;
  const level = quality.level;
  const rtt = quality.rttMs == null ? t("calls.quality.notMeasured") : `${Math.round(quality.rttMs + extraDelayMs)} ms`;

  return (
    <div className="w-full space-y-2" data-testid="call-quality">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn("flex w-full items-center justify-between gap-2 rounded-md px-1 py-1 text-sm", LEVEL_TONE[level])}
        aria-expanded={open}
      >
        <span className="inline-flex items-center gap-2">
          <Activity className="h-4 w-4" />
          <SignalBars level={level} />
          <span className="font-medium">{t(`calls.quality.${LEVEL_KEY[level]}`)}</span>
        </span>
        {open ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
      </button>

      {(quality.noPeerAudio || level === 1) && (
        <div role="alert" className="flex items-start gap-2 rounded-md bg-status-warning-bg px-3 py-2 text-xs text-status-warning-text" data-testid="call-quality-alert">
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{quality.noPeerAudio ? t("calls.quality.noPeerAudio") : t("calls.quality.degraded")}</span>
        </div>
      )}

      {open && (
        <div className="space-y-2 rounded-md bg-muted/40 p-3" data-testid="call-quality-details">
          <Row
            label={`${t("calls.quality.index")} (${t("calls.quality.estimated")})`}
            value={quality.mosEstimated == null ? "—" : quality.mosEstimated.toFixed(1).replace(".", ",")}
          />
          <Row label={t("calls.quality.rtt")} value={rtt} />
          <Row label={t("calls.quality.loss")} value={`${quality.lossPct.toFixed(1).replace(".", ",")}%`} />
          <Row label={t("calls.quality.jitter")} value={`${Math.round(quality.jitterMs)} ms`} />
          <Row label={t("calls.quality.bitrateTx")} value={formatBitrate(quality.txBytesPerSec)} />
          <Row label={t("calls.quality.bitrateRx")} value={formatBitrate(quality.rxBytesPerSec)} />
          <LevelBar label={t("calls.quality.levelTx")} level={quality.txLevel} />
          <LevelBar label={t("calls.quality.levelRx")} level={quality.rxLevel} />
          <p className="flex items-start gap-1.5 pt-1 text-[0.7rem] leading-snug text-muted-foreground">
            <Info className="mt-0.5 h-3 w-3 shrink-0" />
            <span>
              {t("calls.quality.indexNote")} {t("calls.quality.lossNote")}
            </span>
          </p>
        </div>
      )}
    </div>
  );
};

export default CallQualityPanel;
