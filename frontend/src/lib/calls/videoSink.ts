import { VideoPlayback, type ChunkCtor, type DecoderCtor } from "./videoPlayback";
import type { VideoFrame } from "./videoFrame";
import { planRotation, type Rotation } from "./videoRotation";

/** Quadro decodificado: o recorte do `VideoFrame` do WebCodecs que o desenho usa. */
interface DecodedFrame {
  displayWidth: number;
  displayHeight: number;
  /** Timestamp em microssegundos (o mesmo que o chunk levou ao decoder). */
  timestamp?: number;
  close(): void;
}

/** Quantas rotações pendentes guardar (um quadro decodificado sai depois de entrar; a fila é curta). */
const MAX_PENDING_ROTATIONS = 64;

/**
 * Liga o vídeo do contato a um `<canvas>` SEM passar pelo estado do React: um re-render por quadro
 * (15-30 por segundo) seria péssimo. O painel registra o canvas; os quadros que chegam antes dele
 * existir são descartados (o decoder só liga no primeiro quadro-chave de qualquer forma).
 */
export class VideoSink {
  private canvas: HTMLCanvasElement | null = null;
  private playback: VideoPlayback | null = null;
  /** Rotação de cada quadro, por timestamp (µs): o decoder devolve o quadro depois e fora de contexto. */
  private rotations = new Map<number, Rotation>();
  private lastRotation: Rotation = 0;
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
    this.remember(frame);
    this.playback.push(frame);
  }

  private remember(frame: VideoFrame): void {
    this.lastRotation = frame.rotation;
    this.rotations.set(Math.round((frame.ts90k * 1_000_000) / 90_000), frame.rotation);
    if (this.rotations.size > MAX_PENDING_ROTATIONS) {
      const oldest = this.rotations.keys().next().value;
      if (oldest !== undefined) this.rotations.delete(oldest);
    }
  }

  close(): void {
    this.playback?.close();
    this.playback = null;
    this.canvas = null;
    this.rotations.clear();
  }

  private draw(frame: DecodedFrame): void {
    const c = this.canvas;
    try {
      if (c) {
        // A rotação veio com o quadro de entrada; se o timestamp não casar, vale a última conhecida.
        const key = frame.timestamp ?? -1;
        const rotation = this.rotations.get(key) ?? this.lastRotation;
        this.rotations.delete(key);
        const plan = planRotation(frame.displayWidth, frame.displayHeight, rotation);
        if (c.width !== plan.canvasWidth || c.height !== plan.canvasHeight) {
          c.width = plan.canvasWidth;
          c.height = plan.canvasHeight;
        }
        const ctx = c.getContext("2d");
        if (ctx) {
          ctx.setTransform(...plan.matrix);
          ctx.drawImage(frame as unknown as CanvasImageSource, 0, 0);
        }
      }
    } finally {
      frame.close();
    }
  }
}
