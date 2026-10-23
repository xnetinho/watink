import type { VideoFrame } from "./videoFrame";
import { VIDEO_DECODER_CONFIG } from "./videoSupport";

/** Recorte mínimo do `VideoDecoder` do WebCodecs usado aqui (permite injetar um falso nos testes). */
export interface DecoderLike {
  state: string;
  decodeQueueSize?: number;
  configure(config: { codec: string; optimizeForLatency?: boolean }): void;
  decode(chunk: unknown): void;
  close(): void;
}
export type DecoderCtor = new (init: { output: (frame: unknown) => void; error: (e: Error) => void }) => DecoderLike;
export type ChunkCtor = new (init: { type: "key" | "delta"; timestamp: number; data: Uint8Array }) => unknown;

/** Acima disto o decoder está atrasado: melhor pedir um quadro-chave do que acumular atraso. */
const MAX_DECODE_QUEUE = 8;

export interface VideoPlaybackOptions {
  DecoderCtor: DecoderCtor | undefined;
  ChunkCtor: ChunkCtor | undefined;
  /** Recebe cada `VideoFrame` decodificado; quem recebe deve chamar `frame.close()` depois de desenhar. */
  onFrame: (frame: unknown) => void;
}

/**
 * Reproduz o vídeo H.264 do contato com WebCodecs. Regras que o fluxo exige:
 *  - só começa a decodificar num quadro-chave (SPS/PPS vêm inline no Annex-B, sem `description`);
 *  - em erro do decoder, recria e volta a exigir quadro-chave (um delta com o estado quebrado só gera lixo);
 *  - com a fila de decodificação cheia, pede um quadro-chave em vez de acumular atraso.
 */
export class VideoPlayback {
  /** Chamado quando convém o contato mandar um quadro-chave (o business o repassa ao engine). */
  onNeedKeyframe?: () => void;
  /** Tamanho da fila de decodificação; sobrescrevível nos testes. */
  decodeQueueSize: () => number = () => this.decoder?.decodeQueueSize ?? 0;

  private decoder: DecoderLike | null = null;
  private needKey = true;
  private closed = false;

  constructor(private readonly opts: VideoPlaybackOptions) {}

  get supported(): boolean {
    return !!this.opts.DecoderCtor && !!this.opts.ChunkCtor;
  }

  push(frame: VideoFrame): void {
    if (this.closed || !this.supported) return;
    if (this.needKey && !frame.keyframe) return;
    if (!this.decoder) this.decoder = this.open();
    if (!this.decoder) return;
    if (!frame.keyframe && this.decodeQueueSize() > MAX_DECODE_QUEUE) {
      this.needKey = true;
      this.onNeedKeyframe?.();
      return;
    }
    this.needKey = false;
    const Chunk = this.opts.ChunkCtor as ChunkCtor;
    try {
      this.decoder.decode(new Chunk({ type: frame.keyframe ? "key" : "delta", timestamp: Math.round((frame.ts90k * 1_000_000) / 90_000), data: frame.data }));
    } catch {
      this.reset();
    }
  }

  close(): void {
    this.closed = true;
    this.teardown();
  }

  private open(): DecoderLike | null {
    const Ctor = this.opts.DecoderCtor as DecoderCtor;
    try {
      const d = new Ctor({
        output: (f) => this.opts.onFrame(f),
        error: () => this.reset(),
      });
      d.configure({ codec: VIDEO_DECODER_CONFIG.codec, optimizeForLatency: VIDEO_DECODER_CONFIG.optimizeForLatency });
      return d;
    } catch {
      return null;
    }
  }

  private reset(): void {
    this.teardown();
    this.needKey = true;
    this.onNeedKeyframe?.();
  }

  private teardown(): void {
    if (this.decoder && this.decoder.state !== "closed") {
      try {
        this.decoder.close();
      } catch {
        /* já fechado */
      }
    }
    this.decoder = null;
  }
}
