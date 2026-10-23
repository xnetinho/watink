import type { ActiveCall, CallEventPayload, CallPhase, CallQuality } from "./types";

/**
 * Estado do módulo de chamadas. Uma só chamada ativa por operador; a oferta que
 * toca (incoming) e a ativa coexistem só até o operador decidir: atender troca o
 * "ringing" por "connecting".
 */
export interface CallsState {
  /** A chamada em curso do próprio operador (atendida ou originada). */
  active: ActiveCall | null;
  /** Ofertas tocando agora, por callId (pode haver mais de uma em conexões diferentes). */
  ringing: Record<string, ActiveCall>;
  paused: boolean;
}

export const initialCallsState = (paused = false): CallsState => ({ active: null, ringing: {}, paused });

export type CallsAction =
  | { type: "incoming"; payload: CallEventPayload }
  | { type: "answered"; callId: string; byMe: boolean }
  | { type: "ended"; callId: string; endReason?: string }
  | { type: "originate"; payload: CallEventPayload }
  | { type: "accepted"; payload: CallEventPayload }
  | { type: "state"; callId: string; state: string }
  | { type: "quality"; callId: string; quality: CallQuality }
  | { type: "mute"; muted: boolean }
  | { type: "camera"; on: boolean; failure?: string }
  | { type: "recording"; callId: string; recording: boolean }
  | { type: "failure"; callId: string; reason: string }
  | { type: "endRequested"; callId: string }
  | { type: "dismiss" }
  | { type: "pause"; paused: boolean };

function fromPayload(p: CallEventPayload, phase: CallPhase): ActiveCall {
  return {
    callId: p.callId,
    direction: p.direction,
    media: p.media === "video" ? "video" : "audio",
    phase,
    whatsappId: p.whatsappId,
    contact: p.contact ?? {},
    ticketId: p.ticketId ?? null,
    connectedAt: null,
    muted: false,
    camera: false,
    cameraFailure: null,
    recording: false,
    quality: null,
    endReason: null,
    failure: null,
  };
}

export function callsReducer(state: CallsState, action: CallsAction): CallsState {
  switch (action.type) {
    case "incoming": {
      // Pausado: o servidor já não conta este operador, mas se um toque escapar
      // (corrida entre o aviso e a oferta) ele simplesmente não aparece.
      if (state.paused) return state;
      const id = action.payload.callId;
      if (state.ringing[id] || state.active?.callId === id) return state;
      return { ...state, ringing: { ...state.ringing, [id]: fromPayload(action.payload, "ringing") } };
    }

    case "answered": {
      // Outro operador atendeu: o toque some. Se fui eu, o accepted cuida do resto.
      if (action.byMe) return state;
      if (!state.ringing[action.callId]) return state;
      const { [action.callId]: _gone, ...rest } = state.ringing;
      return { ...state, ringing: rest };
    }

    case "accepted": {
      const id = action.payload.callId;
      const { [id]: _gone, ...rest } = state.ringing;
      const base = state.ringing[id] ?? fromPayload(action.payload, "connecting");
      const alreadyConnected = base.connectedAt != null;
      return {
        ...state,
        ringing: rest,
        active: { ...base, phase: alreadyConnected ? "active" : "connecting", contact: action.payload.contact ?? base.contact },
      };
    }

    case "originate":
      return { ...state, active: fromPayload(action.payload, "calling") };

    case "state": {
      if (action.state !== "active") return state;
      // Chamada recebida: o engine pode conectar a mídia e o SSE entregar o "active" ANTES de o
      // POST /accept responder (que é quando a chamada vira "active" no estado). Ela ainda está só
      // em "ringing": guarda o instante nela, e "accepted" o aproveita em vez de ficar em
      // "connecting" para sempre com o cronômetro desligado.
      const early = state.ringing[action.callId];
      if (early && !state.active) {
        return { ...state, ringing: { ...state.ringing, [action.callId]: { ...early, connectedAt: early.connectedAt ?? Date.now() } } };
      }
      if (!state.active || state.active.callId !== action.callId) return state;
      if (state.active.phase === "active") return state;
      return { ...state, active: { ...state.active, phase: "active", connectedAt: Date.now() } };
    }

    case "quality": {
      if (!state.active || state.active.callId !== action.callId) return state;
      return { ...state, active: { ...state.active, quality: action.quality } };
    }

    case "mute":
      return state.active ? { ...state, active: { ...state.active, muted: action.muted } } : state;

    case "camera":
      return state.active
        ? { ...state, active: { ...state.active, camera: action.on, cameraFailure: action.failure ?? null } }
        : state;

    case "recording": {
      if (!state.active || state.active.callId !== action.callId) return state;
      return { ...state, active: { ...state.active, recording: action.recording } };
    }

    case "failure": {
      if (!state.active || state.active.callId !== action.callId) return state;
      if (action.reason === "socket" && state.active.phase === "ended") return state;
      return { ...state, active: { ...state.active, failure: action.reason } };
    }

    case "endRequested": {
      if (!state.active || state.active.callId !== action.callId) return state;
      return { ...state, active: { ...state.active, endedByMe: true } };
    }

    case "ended": {
      let next = state;
      if (state.ringing[action.callId]) {
        const { [action.callId]: _gone, ...rest } = state.ringing;
        next = { ...next, ringing: rest };
      }
      if (next.active?.callId === action.callId) {
        const failure = next.active.failure === "socket" ? null : next.active.failure;
        next = { ...next, active: { ...next.active, phase: "ended", endReason: action.endReason ?? null, failure } };
      }
      return next;
    }

    case "dismiss":
      return state.active?.phase === "ended" ? { ...state, active: null } : state;

    case "pause":
      // Pausar também derruba os toques que já estão na tela.
      return { ...state, paused: action.paused, ringing: action.paused ? {} : state.ringing };
  }
}
