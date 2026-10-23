/** Linha do histórico (GET /calls). Espelha models.CallLog do backend. */
export interface CallLogRow {
  id: number;
  callId: string;
  whatsappId: number;
  contactId: number | null;
  ticketId: number | null;
  direction: "incoming" | "outgoing";
  status: string;
  peerJid: string;
  callerPn: string;
  startedAt: string;
  answeredAt: string | null;
  endedAt: string | null;
  durationSec: number;
  handledByUserId: number | null;
  endReason: string;
  rttAvg: number | null;
  rttMax: number | null;
  lossAvg: number | null;
  lossMax: number | null;
  jitterAvg: number | null;
  jitterMax: number | null;
  mosEstimated: number | null;
  recordingStatus: string;
  recordingDurationSec: number;
}

export interface CallHistoryResponse {
  calls: CallLogRow[];
  total: number;
}

export interface CallFilters {
  status: string;
  direction: string;
  page: number;
  pageSize: number;
}

export const CALL_STATUSES = ["ended", "missed", "rejected", "failed", "interrupted", "active", "ringing"] as const;
