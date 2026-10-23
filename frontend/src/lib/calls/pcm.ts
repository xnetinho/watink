// Utilitários puros de PCM para o áudio das chamadas. O formato do canal é fixo:
// mono, 16 kHz, Int16 little-endian, quadros de 20 ms (320 amostras = 640 bytes).

export const SAMPLE_RATE = 16000;
export const FRAME_SAMPLES = 320;
export const FRAME_BYTES = FRAME_SAMPLES * 2;

/** Reamostra por interpolação linear (ex.: 48 kHz do navegador para 16 kHz). */
export function resample(input: Float32Array, fromRate: number, toRate: number): Float32Array {
  if (fromRate === toRate || input.length === 0) return input;
  const ratio = fromRate / toRate;
  const outLen = Math.floor(input.length / ratio);
  const out = new Float32Array(outLen);
  for (let i = 0; i < outLen; i++) {
    const pos = i * ratio;
    const i0 = Math.floor(pos);
    const i1 = Math.min(i0 + 1, input.length - 1);
    const frac = pos - i0;
    out[i] = input[i0] * (1 - frac) + input[i1] * frac;
  }
  return out;
}

/** Float32 [-1, 1] para Int16, com limitação (nunca dá a volta). */
export function floatToInt16(input: Float32Array): Int16Array {
  const out = new Int16Array(input.length);
  for (let i = 0; i < input.length; i++) {
    const s = Math.max(-1, Math.min(1, input[i]));
    out[i] = s < 0 ? Math.round(s * 32768) : Math.round(s * 32767);
  }
  return out;
}

export function int16ToFloat(input: Int16Array): Float32Array {
  const out = new Float32Array(input.length);
  for (let i = 0; i < input.length; i++) out[i] = input[i] / 32768;
  return out;
}

/** Serializa Int16 como bytes little-endian, independente da ordem da máquina. */
export function int16ToBytes(input: Int16Array): ArrayBuffer {
  const buf = new ArrayBuffer(input.length * 2);
  const view = new DataView(buf);
  for (let i = 0; i < input.length; i++) view.setInt16(i * 2, input[i], true);
  return buf;
}

export function bytesToInt16(buf: ArrayBuffer): Int16Array {
  const n = Math.floor(buf.byteLength / 2);
  const view = new DataView(buf);
  const out = new Int16Array(n);
  for (let i = 0; i < n; i++) out[i] = view.getInt16(i * 2, true);
  return out;
}

/**
 * Acumula amostras de tamanho arbitrário e devolve quadros EXATOS de FRAME_SAMPLES.
 * O que sobra fica para a próxima chamada de push.
 */
export class Framer {
  private pending = new Int16Array(0);

  push(samples: Int16Array): Int16Array[] {
    const merged = new Int16Array(this.pending.length + samples.length);
    merged.set(this.pending, 0);
    merged.set(samples, this.pending.length);
    const frames: Int16Array[] = [];
    let offset = 0;
    while (merged.length - offset >= FRAME_SAMPLES) {
      frames.push(merged.slice(offset, offset + FRAME_SAMPLES));
      offset += FRAME_SAMPLES;
    }
    this.pending = merged.slice(offset);
    return frames;
  }

  /** Quantas amostras aguardam um quadro completo. */
  get buffered(): number {
    return this.pending.length;
  }
}

/** Nível RMS de 0 a 1. */
export function rms(input: Int16Array): number {
  if (input.length === 0) return 0;
  let sum = 0;
  for (let i = 0; i < input.length; i++) {
    const v = input[i] / 32768;
    sum += v * v;
  }
  return Math.sqrt(sum / input.length);
}
