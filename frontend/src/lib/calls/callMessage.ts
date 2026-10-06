import { formatDuration } from "./format";

/** O que a mensagem de sistema de uma chamada leva em `dataJson` (definido no backend). */
export interface CallMessageData {
  callId: string;
  direction: "incoming" | "outgoing";
  status: string;
  endReason?: string;
  durationSec?: number;
  recordingStatus?: string;
  mosEstimated?: number | null;
  handledByName?: string;
  handledByUserId?: number;
}

/** Lê o dataJson (string ou objeto) de uma mensagem de chamada; null se não for válido. */
export function parseCallData(raw: unknown): CallMessageData | null {
  let obj: unknown = raw;
  if (typeof raw === "string") {
    try {
      obj = JSON.parse(raw);
    } catch {
      return null;
    }
  }
  if (!obj || typeof obj !== "object") return null;
  const d = obj as Record<string, unknown>;
  if (typeof d.callId !== "string" || !d.callId) return null;
  return {
    callId: d.callId,
    direction: d.direction === "outgoing" ? "outgoing" : "incoming",
    status: typeof d.status === "string" ? d.status : "ended",
    endReason: typeof d.endReason === "string" ? d.endReason : undefined,
    durationSec: typeof d.durationSec === "number" ? d.durationSec : undefined,
    recordingStatus: typeof d.recordingStatus === "string" ? d.recordingStatus : undefined,
    mosEstimated: typeof d.mosEstimated === "number" ? d.mosEstimated : null,
    handledByName: typeof d.handledByName === "string" ? d.handledByName : undefined,
    handledByUserId: typeof d.handledByUserId === "number" ? d.handledByUserId : undefined,
  };
}

export type CallTone = "success" | "error" | "warning" | "default";

/** Cor do resultado: atendida é sucesso; perdida/recusada/falha chamam atenção. */
export function callTone(status: string): CallTone {
  switch (status) {
    case "ended":
    case "active":
      return "success";
    case "missed":
      return "warning";
    case "rejected":
    case "failed":
    case "interrupted":
      return "error";
    default:
      return "default";
  }
}

/** Chave i18n do título da mensagem, a partir do status e da direção. */
export function callTitleKey(d: Pick<CallMessageData, "status" | "direction">): string {
  switch (d.status) {
    case "missed":
      return "calls.message.missed";
    case "rejected":
      return "calls.message.rejected";
    case "failed":
    case "interrupted":
      return "calls.message.interrupted";
    default:
      return d.direction === "outgoing" ? "calls.message.made" : "calls.message.received";
  }
}

/** Duração legível, ou null se a chamada não chegou a ser atendida. */
export function callDurationLabel(d: Pick<CallMessageData, "durationSec" | "status">): string | null {
  if (d.status === "missed" || d.status === "rejected") return null;
  if (d.durationSec == null) return null;
  return formatDuration(d.durationSec);
}

export type RecordingView = "none" | "ready" | "failed" | "deleted" | "recording";

export function recordingView(d: Pick<CallMessageData, "recordingStatus">): RecordingView {
  switch (d.recordingStatus) {
    case "ready":
      return "ready";
    case "failed":
      return "failed";
    case "deleted":
      return "deleted";
    case "recording":
      return "recording";
    default:
      return "none";
  }
}
