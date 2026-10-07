/**
 * Formato do quadro de VÍDEO no WebSocket da chamada (o mesmo do engine, `calls/videowire.go`):
 *
 *   [FF 56 44 01][flags 1B][ts90k 4B BE][access unit Annex-B…]
 *
 * flags bit0 = quadro-chave. O áudio PCM continua sendo uma mensagem binária de 640 bytes, sem
 * cabeçalho; o vídeo é reconhecido pelo prefixo.
 */
export const VIDEO_HEADER_LEN = 9;
const MAGIC = [0xff, 0x56, 0x44, 0x01] as const;

export interface VideoFrame {
  keyframe: boolean;
  /** Timestamp em 90 kHz. */
  ts90k: number;
  /** Access unit H.264 em Annex-B (com os start codes). */
  data: Uint8Array;
}

/** A mensagem binária é um quadro de vídeo? (PCM nunca é.) */
export function isVideoFrame(msg: Uint8Array): boolean {
  return msg.length > VIDEO_HEADER_LEN && msg[0] === MAGIC[0] && msg[1] === MAGIC[1] && msg[2] === MAGIC[2] && msg[3] === MAGIC[3];
}

/** Lê o quadro; devolve null se a mensagem não for de vídeo. O `data` aponta para dentro de `msg`. */
export function decodeVideoFrame(msg: Uint8Array): VideoFrame | null {
  if (!isVideoFrame(msg)) return null;
  const ts90k = ((msg[5] << 24) | (msg[6] << 16) | (msg[7] << 8) | msg[8]) >>> 0;
  return { keyframe: (msg[4] & 0x01) !== 0, ts90k, data: msg.subarray(VIDEO_HEADER_LEN) };
}

/** Monta a mensagem de um quadro (usado pelo envio, na fase 2, e pelos testes). */
export function encodeVideoFrame(frame: VideoFrame): Uint8Array {
  const out = new Uint8Array(VIDEO_HEADER_LEN + frame.data.length);
  out.set(MAGIC, 0);
  out[4] = frame.keyframe ? 1 : 0;
  out[5] = (frame.ts90k >>> 24) & 0xff;
  out[6] = (frame.ts90k >>> 16) & 0xff;
  out[7] = (frame.ts90k >>> 8) & 0xff;
  out[8] = frame.ts90k & 0xff;
  out.set(frame.data, VIDEO_HEADER_LEN);
  return out;
}
