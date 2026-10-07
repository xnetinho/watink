export type CallDirection = "incoming" | "outgoing";

/** Estados vistos pelo operador. */
export type CallPhase =
  | "idle"
  | "ringing" // chamada recebida tocando
  | "calling" // chamada originada, esperando o contato atender
  | "connecting" // atendida, ligando o áudio
  | "active"
  | "ended";

export interface CallContactInfo {
  id?: number;
  name?: string;
  number?: string;
  profilePicUrl?: string;
}

/** Corpo do evento `call.incoming` e do `POST /calls` / `accept`. */
export type CallMedia = "audio" | "video";

export interface CallEventPayload {
  callId: string;
  whatsappId: number;
  direction: CallDirection;
  /** Ausente (engine anterior ao vídeo) vale "audio". */
  media?: CallMedia;
  status?: string;
  contact?: CallContactInfo;
  ticketId?: number | null;
  startedAt?: string;
}

export interface CallQuality {
  rttMs: number | null;
  lossPct: number;
  jitterMs: number;
  level: 1 | 2 | 3;
  mosEstimated: number | null;
  alerts: string[];
  noPeerAudio: boolean;
  silentMs: number;
  txBytesPerSec?: number;
  rxBytesPerSec?: number;
  txLevel?: number;
  rxLevel?: number;
}

export interface ActiveCall {
  callId: string;
  direction: CallDirection;
  media: CallMedia;
  phase: CallPhase;
  whatsappId: number;
  contact: CallContactInfo;
  ticketId: number | null;
  /** Instante em que a mídia conectou de verdade: o cronômetro parte daqui. */
  connectedAt: number | null;
  muted: boolean;
  recording: boolean;
  quality: CallQuality | null;
  /** Motivo do fim, quando phase === "ended". */
  endReason: string | null;
  /** Falha de áudio (ex.: microfone negado), quando houver. */
  failure: string | null;
  /** O operador clicou em Encerrar: "user_ended" é dele, não do contato. */
  endedByMe?: boolean;
}
