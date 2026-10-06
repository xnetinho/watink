import React, { createContext, useCallback, useContext, useEffect, useMemo, useReducer, useRef } from "react";
import { AuthContext } from "../Auth/AuthContext";
import { check } from "../../components/Can";
import api from "../../services/api";
import { subscribeToSocket } from "../../services/sse-client";
import { callsReducer, initialCallsState, type CallsState } from "./callReducer";
import type { ActiveCall, CallEventPayload, CallQuality } from "./types";
import { useCallAudio, type AudioFailure, type CallTelemetry } from "../../lib/calls/useCallAudio";
import { notify } from "../../lib/notify";
import { t } from "../../lib/calls/t";

const PAUSE_KEY = "wt:calls:paused";

export interface CallsContextValue {
  state: CallsState;
  canReceive: boolean;
  canPlace: boolean;
  ringing: ActiveCall[];
  active: ActiveCall | null;
  paused: boolean;
  accept: (callId: string) => Promise<void>;
  reject: (callId: string) => Promise<void>;
  place: (ticketId: number) => Promise<void>;
  end: () => Promise<void>;
  setMuted: (muted: boolean) => void;
  setPaused: (paused: boolean) => void;
  startRecording: () => Promise<void>;
  stopRecording: () => Promise<void>;
  dismiss: () => void;
  audioLevels: { tx: number; rx: number };
}

const noop = async () => undefined;

export const CallsContext = createContext<CallsContextValue>({
  state: initialCallsState(),
  canReceive: false,
  canPlace: false,
  ringing: [],
  active: null,
  paused: false,
  accept: noop,
  reject: noop,
  place: noop,
  end: noop,
  setMuted: () => undefined,
  setPaused: () => undefined,
  startRecording: noop,
  stopRecording: noop,
  dismiss: () => undefined,
  audioLevels: { tx: 0, rx: 0 },
});

export const useCalls = () => useContext(CallsContext);

function readPaused(): boolean {
  try {
    return localStorage.getItem(PAUSE_KEY) === "true";
  } catch {
    return false;
  }
}

/** Interpreta o call.quality do servidor no formato da tela. */
export function toQuality(p: Record<string, unknown>): CallQuality {
  return {
    rttMs: typeof p.rttMs === "number" ? p.rttMs : null,
    lossPct: Number(p.lossPct ?? 0),
    jitterMs: Number(p.jitterMs ?? 0),
    level: ([1, 2, 3].includes(Number(p.level)) ? Number(p.level) : 3) as 1 | 2 | 3,
    mosEstimated: typeof p.mosEstimated === "number" ? p.mosEstimated : null,
    alerts: Array.isArray(p.alerts) ? (p.alerts as string[]) : [],
    noPeerAudio: Boolean(p.noPeerAudio),
    silentMs: Number(p.silentMs ?? 0),
  };
}

export const CallsProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user } = useContext(AuthContext);
  const canReceive = check(user, "calls:receive");
  const canPlace = check(user, "calls:place");
  const myId = Number(user?.id ?? 0);

  const [state, dispatch] = useReducer(callsReducer, undefined, () => initialCallsState(readPaused()));
  const stateRef = useRef(state);
  stateRef.current = state;

  // Avisa o servidor do estado de pausa para ele não contar este operador como
  // elegível. Ao reconectar/montar reenvia: o servidor guarda isso em memória.
  const announcePause = useCallback(
    (paused: boolean) => {
      if (!canReceive) return;
      api.put("/calls/pause", { paused }).catch(() => undefined);
    },
    [canReceive],
  );

  useEffect(() => {
    announcePause(readPaused());
  }, [announcePause]);

  useEffect(() => {
    if (!user || (!canReceive && !canPlace)) return undefined;
    return subscribeToSocket({
      "call.incoming": (p: CallEventPayload) => {
        if (canReceive) dispatch({ type: "incoming", payload: p });
      },
      "call.answered": (p: { callId: string; handledByUserId: number }) =>
        dispatch({ type: "answered", callId: p.callId, byMe: p.handledByUserId === myId }),
      "call.state": (p: { callId: string; state: string }) => dispatch({ type: "state", callId: p.callId, state: p.state }),
      "call.ended": (p: { callId: string; endReason?: string }) =>
        dispatch({ type: "ended", callId: p.callId, endReason: p.endReason }),
      "call.quality": (p: Record<string, unknown> & { callId: string }) =>
        dispatch({ type: "quality", callId: p.callId, quality: toQuality(p) }),
      "call.recording": (p: { callId: string; recording: boolean }) =>
        dispatch({ type: "recording", callId: p.callId, recording: p.recording }),
    });
  }, [user, canReceive, canPlace, myId]);

  const accept = useCallback(async (callId: string) => {
    try {
      const { data } = await api.post(`/calls/${encodeURIComponent(callId)}/accept`, {});
      dispatch({
        type: "accepted",
        payload: {
          callId,
          whatsappId: data.whatsappId,
          direction: "incoming",
          ticketId: data.ticketId ?? null,
          contact: stateRef.current.ringing[callId]?.contact,
        },
      });
    } catch (err) {
      const code = (err as { response?: { data?: { code?: string } } })?.response?.data?.code;
      dispatch({ type: "answered", callId, byMe: false });
      if (code === "ALREADY_ANSWERED") notify.warning(t("calls.alreadyAnswered"));
      else notify.error(err);
    }
  }, []);

  const reject = useCallback(async (callId: string) => {
    try {
      await api.post(`/calls/${encodeURIComponent(callId)}/reject`, {});
    } catch (err) {
      notify.error(err);
    }
    dispatch({ type: "ended", callId, endReason: "declined" });
  }, []);

  const place = useCallback(async (ticketId: number) => {
    try {
      const { data } = await api.post("/calls", { ticketId });
      dispatch({
        type: "originate",
        payload: { callId: data.callId, whatsappId: data.whatsappId, direction: "outgoing", ticketId, contact: { number: data.callerPn } },
      });
    } catch (err) {
      notify.error(err);
    }
  }, []);

  const end = useCallback(async () => {
    const a = stateRef.current.active;
    if (!a) return;
    if (a.phase === "ended") {
      dispatch({ type: "dismiss" });
      return;
    }
    try {
      await api.post(`/calls/${encodeURIComponent(a.callId)}/end`, {});
    } catch (err) {
      notify.error(err);
    }
  }, []);

  const setMuted = useCallback((muted: boolean) => dispatch({ type: "mute", muted }), []);

  const setPaused = useCallback(
    (paused: boolean) => {
      try {
        localStorage.setItem(PAUSE_KEY, String(paused));
      } catch {
        /* preferência local indisponível: segue só em memória */
      }
      dispatch({ type: "pause", paused });
      announcePause(paused);
    },
    [announcePause],
  );

  const startRecording = useCallback(async () => {
    const a = stateRef.current.active;
    if (!a) return;
    try {
      await api.post(`/calls/${encodeURIComponent(a.callId)}/recording/start`, {});
    } catch (err) {
      notify.error(err);
    }
  }, []);

  const stopRecording = useCallback(async () => {
    const a = stateRef.current.active;
    if (!a) return;
    try {
      await api.post(`/calls/${encodeURIComponent(a.callId)}/recording/stop`, {});
    } catch (err) {
      notify.error(err);
    }
  }, []);

  const dismiss = useCallback(() => dispatch({ type: "dismiss" }), []);

  const active = state.active;
  const onFailure = useCallback((reason: AudioFailure) => {
    const a = stateRef.current.active;
    if (!a) return;
    dispatch({ type: "failure", callId: a.callId, reason });
    if (reason !== "socket") {
      api.post(`/calls/${encodeURIComponent(a.callId)}/end`, {}).catch(() => undefined);
    }
  }, []);
  const onTelemetry = useCallback((t: CallTelemetry) => {
    const a = stateRef.current.active;
    if (!a) return;
    const prev = a.quality;
    dispatch({
      type: "quality",
      callId: a.callId,
      quality: {
        rttMs: t.rttMs,
        lossPct: t.lossPct,
        jitterMs: t.jitterMs,
        level: prev?.level ?? 3,
        mosEstimated: prev?.mosEstimated ?? null,
        alerts: prev?.alerts ?? [],
        noPeerAudio: t.noPeerAudio,
        silentMs: t.silentMs,
        txBytesPerSec: t.txBytesPerSec,
        rxBytesPerSec: t.rxBytesPerSec,
        txLevel: t.txLevel,
        rxLevel: t.rxLevel,
      },
    });
  }, []);

  const audio = useCallAudio({
    callId: active?.callId ?? null,
    enabled: !!active && (active.phase === "connecting" || active.phase === "active" || active.phase === "calling"),
    muted: active?.muted ?? false,
    onFailure,
    onTelemetry,
  });

  const value = useMemo<CallsContextValue>(
    () => ({
      state,
      canReceive,
      canPlace,
      ringing: Object.values(state.ringing),
      active,
      paused: state.paused,
      accept,
      reject,
      place,
      end,
      setMuted,
      setPaused,
      startRecording,
      stopRecording,
      dismiss,
      audioLevels: audio.levels,
    }),
    [state, canReceive, canPlace, active, accept, reject, place, end, setMuted, setPaused, startRecording, stopRecording, dismiss, audio.levels],
  );

  return <CallsContext.Provider value={value}>{children}</CallsContext.Provider>;
};
