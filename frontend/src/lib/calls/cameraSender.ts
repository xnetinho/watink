import { encodeVideoFrame } from "./videoFrame";

/**
 * Câmera do operador → contato (fase 2 do plano add-whatsapp-video-calls). Captura com `getUserMedia`,
 * lê os quadros com `MediaStreamTrackProcessor`, codifica em H.264 Annex-B com `VideoEncoder` e entrega
 * cada access unit já no formato do fio (`encodeVideoFrame`) a `send`. O servidor só empacota e cifra:
 * quem codifica é o navegador.
 *
 * Tudo que é do navegador entra por `CameraDeps`, para testar sem câmera.
 */

/** Níveis da fase: 640x480 a 15 fps e 600 kbps cabem com folga no nível 3.1 e na saída do operador. */
export const CAMERA = {
  width: 640,
  height: 480,
  fps: 15,
  bitrate: 600_000,
  /** Intervalo máximo entre quadros-chave (o contato precisa de um para começar e para se recuperar). */
  keyframeEveryMs: 2000,
  /** Fila de codificação acima disto = o codificador não dá conta: descarta o quadro da câmera. */
  maxEncodeQueue: 4,
  /** Mesmo perfil do decodificador: H.264 Constrained Baseline nível 3.1, Annex-B com SPS/PPS inline. */
  codec: "avc1.42E01F",
} as const;

export type CameraFailure = "denied" | "unavailable" | "encoder";

export interface CameraFrameLike {
  timestamp: number;
  close(): void;
}

export interface EncodedChunkLike {
  type: "key" | "delta";
  timestamp: number;
  byteLength: number;
  copyTo(dest: Uint8Array): void;
}

export interface EncoderLike {
  encodeQueueSize: number;
  state: string;
  configure(config: Record<string, unknown>): void;
  encode(frame: CameraFrameLike, opts?: { keyFrame?: boolean }): void;
  close(): void;
}

export interface TrackLike {
  stop(): void;
}

export interface StreamLike {
  getVideoTracks(): TrackLike[];
  getTracks(): TrackLike[];
}

export interface ReaderLike {
  read(): Promise<{ done: boolean; value?: CameraFrameLike }>;
  cancel(): Promise<void> | void;
}

export interface CameraDeps {
  getStream(constraints: unknown): Promise<StreamLike>;
  /** Cria o leitor de quadros da faixa (MediaStreamTrackProcessor.readable.getReader()). */
  readerFor(track: TrackLike): ReaderLike;
  createEncoder(init: { output: (chunk: EncodedChunkLike) => void; error: (e: Error) => void }): EncoderLike;
  now(): number;
}

export interface CameraOptions {
  deps: CameraDeps;
  /** Recebe cada mensagem binária pronta para o WebSocket. */
  send: (msg: Uint8Array) => void;
  onFailure: (reason: CameraFailure) => void;
}

/** Classifica o erro do getUserMedia: permissão negada é diferente de câmera inexistente ou ocupada. */
export function classifyCameraError(err: unknown): CameraFailure {
  const name = (err as { name?: string } | null)?.name ?? "";
  if (name === "NotAllowedError" || name === "SecurityError" || name === "PermissionDeniedError") return "denied";
  return "unavailable";
}

/** Dependências de produção: as APIs do Chromium. Só chamar onde `detectVideoSupport().canSend`. */
export function browserCameraDeps(): CameraDeps {
  const w = globalThis as unknown as {
    MediaStreamTrackProcessor: new (init: { track: unknown }) => { readable: { getReader(): ReaderLike } };
    VideoEncoder: new (init: { output: (chunk: EncodedChunkLike) => void; error: (e: Error) => void }) => EncoderLike;
  };
  return {
    getStream: (constraints) => navigator.mediaDevices.getUserMedia(constraints as MediaStreamConstraints) as unknown as Promise<StreamLike>,
    readerFor: (track) => new w.MediaStreamTrackProcessor({ track }).readable.getReader(),
    createEncoder: (init) => new w.VideoEncoder(init),
    now: () => performance.now(),
  };
}

export class CameraSender {
  private stream: StreamLike | null = null;
  private encoder: EncoderLike | null = null;
  private reader: ReaderLike | null = null;
  private running = false;
  private forceKey = true;
  private lastKeyAt = 0;
  private lastFrameAt = 0;
  private orientation: 0 | 1 | 2 | 3 = 0;
  /** Relógio de 90 kHz dos quadros enviados: parte do primeiro quadro e acompanha o timestamp da câmera. */
  private baseTs: number | null = null;

  constructor(private readonly opts: CameraOptions) {}

  get active(): boolean {
    return this.running;
  }

  /** Gira a imagem que o contato verá: quartos de volta horários (0..3), como o CVO do WhatsApp. */
  setOrientation(quarters: 0 | 1 | 2 | 3): void {
    this.orientation = quarters;
  }

  /** O contato perdeu o vídeo (PLI): o próximo quadro codificado sai como quadro-chave. */
  requestKeyframe(): void {
    this.forceKey = true;
  }

  async start(): Promise<boolean> {
    if (this.running) return true;
    const { deps, onFailure } = this.opts;
    let stream: StreamLike;
    try {
      stream = await deps.getStream({
        video: { width: { ideal: CAMERA.width }, height: { ideal: CAMERA.height }, frameRate: { ideal: CAMERA.fps }, facingMode: "user" },
        audio: false,
      });
    } catch (err) {
      onFailure(classifyCameraError(err));
      return false;
    }
    const track = stream.getVideoTracks()[0];
    if (!track) {
      stream.getTracks().forEach((t) => t.stop());
      onFailure("unavailable");
      return false;
    }
    let encoder: EncoderLike;
    try {
      encoder = deps.createEncoder({
        output: (chunk) => this.onChunk(chunk),
        error: () => this.fail("encoder"),
      });
      encoder.configure({
        codec: CAMERA.codec,
        width: CAMERA.width,
        height: CAMERA.height,
        bitrate: CAMERA.bitrate,
        framerate: CAMERA.fps,
        latencyMode: "realtime",
        avc: { format: "annexb" },
      });
    } catch {
      stream.getTracks().forEach((t) => t.stop());
      onFailure("encoder");
      return false;
    }
    this.stream = stream;
    this.encoder = encoder;
    this.reader = deps.readerFor(track);
    this.running = true;
    this.forceKey = true;
    this.baseTs = null;
    this.lastFrameAt = -Infinity;
    this.lastKeyAt = -Infinity;
    void this.pump(this.reader);
    return true;
  }

  stop(): void {
    this.running = false;
    try {
      void this.reader?.cancel();
    } catch {
      /* já cancelado */
    }
    this.reader = null;
    this.stream?.getTracks().forEach((t) => t.stop());
    this.stream = null;
    if (this.encoder && this.encoder.state !== "closed") {
      try {
        this.encoder.close();
      } catch {
        /* já fechado */
      }
    }
    this.encoder = null;
  }

  private fail(reason: CameraFailure): void {
    if (!this.running) return;
    this.stop();
    this.opts.onFailure(reason);
  }

  private async pump(reader: ReaderLike): Promise<void> {
    const { deps } = this.opts;
    const minGap = 1000 / CAMERA.fps - 5;
    try {
      while (this.running) {
        const { done, value } = await reader.read();
        if (done || !value) break;
        const enc = this.encoder;
        const now = deps.now();
        const tooSoon = now - this.lastFrameAt < minGap;
        if (!this.running || !enc || enc.state === "closed" || tooSoon || enc.encodeQueueSize > CAMERA.maxEncodeQueue) {
          value.close();
          continue;
        }
        this.lastFrameAt = now;
        const key = this.forceKey || now - this.lastKeyAt >= CAMERA.keyframeEveryMs;
        try {
          enc.encode(value, key ? { keyFrame: true } : undefined);
          if (key) {
            this.forceKey = false;
            this.lastKeyAt = now;
          }
        } catch {
          this.fail("encoder");
        } finally {
          value.close();
        }
      }
    } catch {
      this.fail("unavailable");
    }
  }

  private onChunk(chunk: EncodedChunkLike): void {
    if (!this.running) return;
    const data = new Uint8Array(chunk.byteLength);
    chunk.copyTo(data);
    if (this.baseTs === null) this.baseTs = chunk.timestamp;
    const ts90k = Math.round(((chunk.timestamp - this.baseTs) * 90_000) / 1_000_000) >>> 0;
    this.opts.send(
      encodeVideoFrame({ keyframe: chunk.type === "key", rotation: this.orientation, ts90k, data }),
    );
  }
}
