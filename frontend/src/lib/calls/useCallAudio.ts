import { useCallback, useEffect, useRef, useState } from "react";
import { getBackendUrl } from "../../config";
import { JitterBuffer } from "./jitterBuffer";
import {
  FRAME_SAMPLES,
  Framer,
  SAMPLE_RATE,
  bytesToInt16,
  floatToInt16,
  int16ToBytes,
  int16ToFloat,
  resample,
  rms,
} from "./pcm";
import { CAPTURE_PROCESSOR, PLAYBACK_PROCESSOR, loadCallWorklets } from "./worklet";

export type AudioFailure = "mic_denied" | "mic_unavailable" | "socket" | "unsupported";

export interface CallTelemetry {
  type: "quality";
  callId: string;
  rttMs: number | null;
  lossPct: number;
  jitterMs: number;
  txBytesPerSec: number;
  rxBytesPerSec: number;
  txLevel: number;
  rxLevel: number;
  relayConnected: boolean;
  relayDrops: number;
  silentMs: number;
  noPeerAudio: boolean;
}

export interface UseCallAudioOptions {
  callId: string | null;
  /** Só conecta quando verdadeiro (chamada atendida e conectando/ativa). */
  enabled: boolean;
  muted: boolean;
  onFailure: (reason: AudioFailure) => void;
  onTelemetry?: (t: CallTelemetry) => void;
}

function getToken(): string {
  const raw = localStorage.getItem("token");
  if (!raw) return "";
  try {
    return JSON.parse(raw) as string;
  } catch {
    return raw;
  }
}

function audioUrl(callId: string): string {
  const base = getBackendUrl().replace(/^http/, "ws");
  return `${base}/api/v1/calls/${encodeURIComponent(callId)}/audio?token=${encodeURIComponent(getToken())}`;
}

/** Classifica o erro do getUserMedia: permissão negada é diferente de sem dispositivo. */
export function classifyMicError(err: unknown): AudioFailure {
  const name = (err as { name?: string } | null)?.name ?? "";
  if (name === "NotAllowedError" || name === "SecurityError" || name === "PermissionDeniedError") return "mic_denied";
  return "mic_unavailable";
}

/**
 * Liga o microfone e o alto-falante ao WebSocket de áudio da chamada: captura em
 * 16 kHz mono Int16 com quadros de 20 ms, e reproduz o que chega com um jitter
 * buffer de ~60 ms. Encerra tudo ao desmontar ou quando `enabled` volta a falso.
 */
export function useCallAudio({ callId, enabled, muted, onFailure, onTelemetry }: UseCallAudioOptions) {
  const mutedRef = useRef(muted);
  const failureRef = useRef(onFailure);
  const telemetryRef = useRef(onTelemetry);
  const [connected, setConnected] = useState(false);
  const [levels, setLevels] = useState({ tx: 0, rx: 0 });

  useEffect(() => {
    mutedRef.current = muted;
  }, [muted]);
  useEffect(() => {
    failureRef.current = onFailure;
    telemetryRef.current = onTelemetry;
  }, [onFailure, onTelemetry]);

  const cleanupRef = useRef<(() => void) | null>(null);
  const teardown = useCallback(() => {
    cleanupRef.current?.();
    cleanupRef.current = null;
    setConnected(false);
  }, []);

  useEffect(() => {
    if (!enabled || !callId) return undefined;
    if (typeof AudioContext === "undefined" || !navigator.mediaDevices?.getUserMedia) {
      failureRef.current("unsupported");
      return undefined;
    }

    let disposed = false;
    const jitter = new JitterBuffer();
    const framer = new Framer();
    let ws: WebSocket | null = null;
    let stream: MediaStream | null = null;
    let ctx: AudioContext | null = null;
    let playTimer: ReturnType<typeof setInterval> | null = null;
    let lastTx = 0;
    let lastRx = 0;

    const stop = () => {
      disposed = true;
      if (playTimer) clearInterval(playTimer);
      try { ws?.close(); } catch { /* já fechado */ }
      stream?.getTracks().forEach((t) => t.stop());
      void ctx?.close().catch(() => undefined);
    };
    cleanupRef.current = stop;

    (async () => {
      try {
        stream = await navigator.mediaDevices.getUserMedia({
          audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
        });
      } catch (err) {
        if (!disposed) failureRef.current(classifyMicError(err));
        return;
      }
      if (disposed) { stream.getTracks().forEach((t) => t.stop()); return; }

      ctx = new AudioContext();
      try {
        await loadCallWorklets(ctx);
      } catch {
        if (!disposed) failureRef.current("unsupported");
        return;
      }
      if (disposed) return;

      const source = ctx.createMediaStreamSource(stream);
      const capture = new AudioWorkletNode(ctx, CAPTURE_PROCESSOR);
      const playback = new AudioWorkletNode(ctx, PLAYBACK_PROCESSOR, { outputChannelCount: [1] });
      source.connect(capture);
      playback.connect(ctx.destination);

      ws = new WebSocket(audioUrl(callId));
      ws.binaryType = "arraybuffer";
      ws.onopen = () => { if (!disposed) setConnected(true); };
      ws.onerror = () => { if (!disposed) failureRef.current("socket"); };
      ws.onclose = () => { if (!disposed) { setConnected(false); failureRef.current("socket"); } };
      ws.onmessage = (ev) => {
        if (typeof ev.data === "string") {
          try {
            const t = JSON.parse(ev.data) as CallTelemetry;
            if (t.type === "quality") telemetryRef.current?.(t);
          } catch { /* mensagem de controle inválida: ignora */ }
          return;
        }
        const pcm = bytesToInt16(ev.data as ArrayBuffer);
        lastRx = rms(pcm);
        for (let i = 0; i + FRAME_SAMPLES <= pcm.length; i += FRAME_SAMPLES) jitter.push(pcm.slice(i, i + FRAME_SAMPLES));
      };

      const sampleRate = ctx.sampleRate;
      capture.port.onmessage = (ev: MessageEvent<Float32Array>) => {
        if (mutedRef.current || !ws || ws.readyState !== WebSocket.OPEN) { lastTx = 0; return; }
        const pcm16 = floatToInt16(resample(ev.data, sampleRate, SAMPLE_RATE));
        lastTx = rms(pcm16);
        for (const frame of framer.push(pcm16)) ws.send(int16ToBytes(frame));
      };

      // A cada 20 ms entrega um quadro do jitter buffer ao alto-falante.
      playTimer = setInterval(() => {
        const frame = jitter.pull();
        if (frame) playback.port.postMessage(resample(int16ToFloat(frame), SAMPLE_RATE, sampleRate));
      }, 20);
      const meter = setInterval(() => { if (!disposed) setLevels({ tx: lastTx, rx: lastRx }); }, 200);
      const prevStop = cleanupRef.current;
      cleanupRef.current = () => { clearInterval(meter); prevStop?.(); };
    })();

    return () => {
      stop();
      cleanupRef.current = null;
    };
  }, [enabled, callId]);

  return { connected, levels, stop: teardown };
}
