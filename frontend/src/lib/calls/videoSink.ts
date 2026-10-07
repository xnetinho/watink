import { VideoPlayback, type ChunkCtor, type DecoderCtor } from "./videoPlayback";
import type { VideoFrame } from "./videoFrame";

/** Quadro decodificado: o recorte do `VideoFrame` do WebCodecs que o desenho usa. */
interface DecodedFrame {
  displayWidth: number;
  displayHeight: number;
  close(): void;
}

/**
 * Liga o vídeo do contato a um `<canvas>` SEM passar pelo estado do React: um re-render por quadro
 * (15-30 por segundo) seria péssimo. O painel registra o canvas; os quadros que chegam antes dele
 * existir são descartados (o decoder só liga no primeiro quadro-chave de qualquer forma).
 */
export class VideoSink {
  private canvas: HTMLCanvasElement | null = null;
  private playback: VideoPlayback | null = null;
  /** Avisa o business que convém o contato mandar um quadro-chave. */
  onNeedKeyframe?: () => void;

  constructor(
    private readonly DecoderCtor: DecoderCtor | undefined = (globalThis as unknown as { VideoDecoder?: DecoderCtor }).VideoDecoder,
    private readonly ChunkCtor: ChunkCtor | undefined = (globalThis as unknown as { EncodedVideoChunk?: ChunkCtor }).EncodedVideoChunk,
  ) {}

  get supported(): boolean {
    return !!this.DecoderCtor && !!this.ChunkCtor;
  }

  /** Chamado pelo painel com o canvas (ou null ao sair). */
  attach(canvas: HTMLCanvasElement | null): void {
    this.canvas = canvas;
  }

  push(frame: VideoFrame): void {
    if (!this.canvas || !this.supported) return;
    if (!this.playback) {
      this.playback = new VideoPlayback({ DecoderCtor: this.DecoderCtor, ChunkCtor: this.ChunkCtor, onFrame: (f) => this.draw(f as DecodedFrame) });
      this.playback.onNeedKeyframe = () => this.onNeedKeyframe?.();
    }
    this.playback.push(frame);
  }

  close(): void {
    this.playback?.close();
    this.playback = null;
    this.canvas = null;
  }

  private draw(frame: DecodedFrame): void {
    const c = this.canvas;
    try {
      if (c) {
        if (c.width !== frame.displayWidth || c.height !== frame.displayHeight) {
          c.width = frame.displayWidth;
          c.height = frame.displayHeight;
        }
        c.getContext("2d")?.drawImage(frame as unknown as CanvasImageSource, 0, 0);
      }
    } finally {
      frame.close();
    }
  }
}
