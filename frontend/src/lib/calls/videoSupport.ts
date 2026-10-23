/**
 * Suporte do navegador a videochamada. Receber exige só decodificar H.264 com WebCodecs
 * (`VideoDecoder`); enviar a câmera (fase 2) exigirá também `VideoEncoder` e
 * `MediaStreamTrackProcessor`, que hoje só o Chromium (Chrome, Edge, Brave) tem.
 */
export interface VideoSupport {
  /** Dá para mostrar o vídeo do contato. */
  canReceive: boolean;
  /** Dá para enviar a câmera (fase 2). */
  canSend: boolean;
}

type Win = {
  VideoDecoder?: unknown;
  VideoEncoder?: unknown;
  MediaStreamTrackProcessor?: unknown;
};

export function detectVideoSupport(w: Win | undefined = typeof window === "undefined" ? undefined : (window as unknown as Win)): VideoSupport {
  const canReceive = !!w && typeof w.VideoDecoder === "function";
  const canSend = canReceive && typeof w?.VideoEncoder === "function" && typeof w?.MediaStreamTrackProcessor === "function";
  return { canReceive, canSend };
}

/** Configuração do decodificador: H.264 Constrained Baseline nível 3.1, Annex-B com SPS/PPS inline. */
export const VIDEO_DECODER_CONFIG = {
  codec: "avc1.42E01F",
  optimizeForLatency: true,
} as const;
