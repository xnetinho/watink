import { FRAME_SAMPLES, SAMPLE_RATE } from "./pcm";

/** Duração de um quadro em ms. */
const FRAME_MS = (FRAME_SAMPLES / SAMPLE_RATE) * 1000;

export interface JitterBufferOptions {
  /** Reserva antes de começar a tocar (padrão ~60 ms). */
  targetMs?: number;
  /** Acima disso o excedente mais antigo é descartado para o atraso não acumular (~200 ms). */
  maxMs?: number;
}

/**
 * Jitter buffer fixo: guarda quadros recebidos e só começa a entregar depois de
 * juntar `targetMs`. Se o atraso passar de `maxMs`, descarta os quadros MAIS
 * ANTIGOS: ouvir o presente picotado é melhor que ouvir o passado com atraso.
 */
export class JitterBuffer {
  private queue: Int16Array[] = [];
  private primed = false;
  private readonly targetFrames: number;
  private readonly maxFrames: number;
  dropped = 0;
  underruns = 0;

  constructor(opts: JitterBufferOptions = {}) {
    this.targetFrames = Math.max(1, Math.round((opts.targetMs ?? 60) / FRAME_MS));
    this.maxFrames = Math.max(this.targetFrames, Math.round((opts.maxMs ?? 200) / FRAME_MS));
  }

  push(frame: Int16Array): void {
    this.queue.push(frame);
    while (this.queue.length > this.maxFrames) {
      this.queue.shift();
      this.dropped++;
    }
  }

  /** Próximo quadro a tocar, ou null se ainda enchendo ou vazio (underrun). */
  pull(): Int16Array | null {
    if (!this.primed) {
      if (this.queue.length < this.targetFrames) return null;
      this.primed = true;
    }
    const frame = this.queue.shift();
    if (!frame) {
      this.underruns++;
      this.primed = false;
      return null;
    }
    return frame;
  }

  get length(): number {
    return this.queue.length;
  }

  /** Atraso atual em ms (o que o navegador soma ao RTT mostrado). */
  get delayMs(): number {
    return this.queue.length * FRAME_MS;
  }

  clear(): void {
    this.queue = [];
    this.primed = false;
  }
}
