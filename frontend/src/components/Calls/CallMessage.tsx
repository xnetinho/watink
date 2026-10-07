import React, { useState } from "react";
import { PhoneIncoming, PhoneMissed, PhoneOff, PhoneOutgoing, Video } from "lucide-react";
import { StatusChip } from "@/components/ui/status-chip";
import { cn } from "@/lib/utils";
import api from "@/services/api";
import toastError from "@/errors/toastError";
import { t } from "@/lib/calls/t";
import {
  callDurationLabel,
  callTitleKey,
  callTone,
  parseCallData,
  recordingView,
  type CallMessageData,
} from "@/lib/calls/callMessage";

const Icon: React.FC<{ d: CallMessageData }> = ({ d }) => {
  const cls = "h-5 w-5 shrink-0";
  if (d.status === "missed") return <PhoneMissed className={cls} />;
  if (d.status === "rejected" || d.status === "failed" || d.status === "interrupted") return <PhoneOff className={cls} />;
  if (d.media === "video") return <Video className={cls} />;
  return d.direction === "outgoing" ? <PhoneOutgoing className={cls} /> : <PhoneIncoming className={cls} />;
};

/**
 * Mensagem de sistema de uma chamada no histórico do ticket: direção, resultado,
 * quem atendeu, duração e, quando houver, o player da gravação. A URL do áudio só é
 * pedida ao clicar (é assinada, expira e a escuta fica registrada em auditoria).
 */
const CallMessage: React.FC<{ dataJson: unknown; body?: string }> = ({ dataJson, body }) => {
  const d = parseCallData(dataJson);
  const [url, setUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  if (!d) return <span className="text-sm">{body}</span>;

  const duration = callDurationLabel(d);
  const rec = recordingView(d);
  const reason = d.endReason ? t(`calls.endReason.${d.endReason}`) : "";

  const listen = async () => {
    setLoading(true);
    try {
      const { data } = await api.get<{ url: string }>(`/calls/${encodeURIComponent(d.callId)}/recording`);
      setUrl(data.url);
    } catch (err) {
      toastError(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-w-[220px] flex-col gap-2 py-1" data-testid="call-message" data-status={d.status}>
      <div className="flex items-center gap-2">
        <span className={cn("text-primary", d.status === "missed" && "text-status-warning-text", callTone(d.status) === "error" && "text-status-error-text")}>
          <Icon d={d} />
        </span>
        <div className="min-w-0">
          <p className="text-sm font-semibold leading-tight">{t(callTitleKey(d))}</p>
          <p className="text-xs text-muted-foreground">
            {[duration && `${t("calls.message.duration")}: ${duration}`, d.handledByName && `${t("calls.message.handledBy")} ${d.handledByName}`]
              .filter(Boolean)
              .join(" · ")}
          </p>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <StatusChip status={callTone(d.status)} size="sm" label={t(`calls.status.${d.status}`)} />
        {reason && d.status !== "ended" && <span className="text-xs text-muted-foreground">{reason}</span>}
      </div>

      {rec === "ready" &&
        (url ? (
          <audio controls src={url} className="h-9 w-full" data-testid="call-recording-player" />
        ) : (
          <button
            type="button"
            onClick={() => void listen()}
            disabled={loading}
            className="self-start text-xs font-medium text-primary underline-offset-2 hover:underline disabled:opacity-50"
            data-testid="call-recording-listen"
          >
            {t("calls.message.listen")}
          </button>
        ))}
      {rec === "failed" && <p className="text-xs text-status-error-text" data-testid="call-recording-failed">{t("calls.message.recordingFailed")}</p>}
      {rec === "deleted" && <p className="text-xs text-muted-foreground" data-testid="call-recording-deleted">{t("calls.message.recordingDeleted")}</p>}
    </div>
  );
};

export default CallMessage;
