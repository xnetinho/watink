import React, { useEffect, useState } from "react";
import { Circle, Mic, MicOff, PhoneOff, ShieldAlert, TriangleAlert, X } from "lucide-react";
import { Avatar } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { t } from "@/lib/calls/t";
import { useCalls } from "@/context/Calls/CallsContext";
import { displayName, elapsedSeconds, formatDuration } from "@/lib/calls/format";
import type { ActiveCall } from "@/context/Calls/types";
import CallQualityPanel from "./CallQualityPanel";

/** Texto do estado atual da chamada. */
export function phaseLabel(call: ActiveCall, now: number): string {
  switch (call.phase) {
    case "calling":
      return t("calls.active.calling");
    case "connecting":
      return t("calls.active.connecting");
    case "active":
      return formatDuration(elapsedSeconds(call.connectedAt, now));
    case "ended":
      return t("calls.active.ended");
    default:
      return "";
  }
}

const FAILURE_KEY: Record<string, string> = {
  mic_denied: "calls.active.micDenied",
  mic_unavailable: "calls.active.micUnavailable",
  unsupported: "calls.active.unsupported",
  socket: "calls.active.socketLost",
};

/**
 * Tela da chamada em curso, fixa no canto, visível em qualquer página. O
 * cronômetro parte do instante em que a mídia conectou de verdade (não do toque).
 */
const ActiveCallPanel: React.FC<{ recordingAvailable?: boolean; recordingMode?: string }> = ({
  recordingAvailable = false,
  recordingMode = "off",
}) => {
  const { active, end, setMuted, startRecording, stopRecording, dismiss } = useCalls();
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (!active || active.phase !== "active") return undefined;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [active]);

  if (!active) return null;

  const name = displayName(active.contact, t("calls.unknownContact"));
  const ended = active.phase === "ended";
  const canMute = active.phase === "active" || active.phase === "connecting";
  const canRecord = recordingAvailable && recordingMode === "optional" && active.phase === "active";
  const failure = active.failure ? t(FAILURE_KEY[active.failure] ?? "calls.active.socketLost") : null;

  return (
    <div
      className="fixed bottom-4 right-4 z-40 w-80 space-y-3 rounded-2xl bg-card p-4 shadow-[0px_8px_30px_rgba(0,0,0,0.16)]"
      role="region"
      aria-label={t("calls.title")}
      data-testid="active-call"
      data-phase={active.phase}
    >
      <div className="flex items-center gap-3">
        <Avatar size="lg" src={active.contact.profilePicUrl} name={name} />
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-semibold">{name}</p>
          <p className={cn("text-sm tabular-nums", ended ? "text-muted-foreground" : "text-primary")} data-testid="call-status">
            {phaseLabel(active, now)}
          </p>
        </div>
        {ended && (
          <Button variant="ghost" size="icon" onClick={dismiss} aria-label={t("calls.active.close")}>
            <X />
          </Button>
        )}
      </div>

      {active.recording && (
        <div className="flex items-center gap-2 rounded-md bg-status-error-bg px-3 py-1.5 text-xs font-medium text-status-error-text" data-testid="recording-indicator">
          <Circle className="h-3 w-3 animate-pulse fill-current" />
          {t("calls.active.recording")}
        </div>
      )}

      {failure && (
        <div role="alert" className="flex items-start gap-2 rounded-md bg-status-error-bg px-3 py-2 text-xs text-status-error-text" data-testid="call-failure">
          <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{failure}</span>
        </div>
      )}

      {active.muted && !ended && (
        <p className="text-xs text-muted-foreground" data-testid="muted-label">
          {t("calls.active.muted")}
        </p>
      )}

      {active.phase === "active" && <CallQualityPanel quality={active.quality} />}

      <p className="flex items-start gap-1.5 text-[0.7rem] leading-snug text-muted-foreground" data-testid="risk-notice">
        <ShieldAlert className="mt-0.5 h-3 w-3 shrink-0" />
        <span>{t("calls.active.riskNotice")}</span>
      </p>

      {!ended && (
        <div className="flex items-center justify-center gap-2">
          <Button
            variant="secondary"
            size="icon"
            disabled={!canMute}
            onClick={() => setMuted(!active.muted)}
            aria-pressed={active.muted}
            aria-label={active.muted ? t("calls.active.unmute") : t("calls.active.mute")}
            data-testid="mute-call"
          >
            {active.muted ? <MicOff /> : <Mic />}
          </Button>
          {canRecord && (
            <Button
              variant="secondary"
              onClick={() => void (active.recording ? stopRecording() : startRecording())}
              data-testid="record-call"
            >
              <Circle className={cn(active.recording && "fill-current text-destructive")} />
              {active.recording ? t("calls.active.stopRecord") : t("calls.active.record")}
            </Button>
          )}
          <Button variant="destructive" onClick={() => void end()} data-testid="end-call">
            <PhoneOff />
            {t("calls.active.hangUp")}
          </Button>
        </div>
      )}
    </div>
  );
};

export default ActiveCallPanel;
