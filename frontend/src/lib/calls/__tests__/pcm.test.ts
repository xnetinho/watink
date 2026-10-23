import { describe, expect, it } from "vitest";
import {
  FRAME_BYTES,
  FRAME_SAMPLES,
  Framer,
  bytesToInt16,
  floatToInt16,
  int16ToBytes,
  int16ToFloat,
  resample,
  rms,
} from "../pcm";

function sine(n: number, hz: number, rate: number, amp = 0.5): Float32Array {
  const out = new Float32Array(n);
  for (let i = 0; i < n; i++) out[i] = amp * Math.sin((2 * Math.PI * hz * i) / rate);
  return out;
}

// frequência dominante por cruzamentos de zero
function zeroCrossHz(x: Float32Array, rate: number): number {
  let c = 0;
  for (let i = 1; i < x.length; i++) if (x[i - 1] < 0 !== x[i] < 0) c++;
  return c / 2 / (x.length / rate);
}

describe("resample", () => {
  it("48 kHz -> 16 kHz preserva a duração e o tom", () => {
    const input = sine(48000, 440, 48000);
    const out = resample(input, 48000, 16000);
    expect(out.length).toBe(16000);
    expect(zeroCrossHz(out, 16000)).toBeGreaterThan(430);
    expect(zeroCrossHz(out, 16000)).toBeLessThan(450);
  });

  it("44,1 kHz -> 16 kHz preserva o tom", () => {
    const out = resample(sine(44100, 1000, 44100), 44100, 16000);
    expect(Math.abs(zeroCrossHz(out, 16000) - 1000)).toBeLessThan(20);
  });

  it("16 kHz -> 48 kHz (reprodução) triplica o número de amostras", () => {
    const out = resample(sine(16000, 440, 16000), 16000, 48000);
    expect(out.length).toBe(48000);
  });

  it("taxa igual devolve a mesma entrada sem copiar", () => {
    const input = sine(100, 440, 16000);
    expect(resample(input, 16000, 16000)).toBe(input);
  });

  it("entrada vazia não quebra", () => {
    expect(resample(new Float32Array(0), 48000, 16000).length).toBe(0);
  });
});

describe("floatToInt16 / int16ToFloat", () => {
  it("limita em vez de dar a volta", () => {
    const out = floatToInt16(new Float32Array([2, -2, 1, -1, 0]));
    expect(Array.from(out)).toEqual([32767, -32768, 32767, -32768, 0]);
  });

  it("ida e volta mantém o sinal dentro de 1 LSB", () => {
    const input = sine(1000, 440, 16000, 0.7);
    const back = int16ToFloat(floatToInt16(input));
    let maxErr = 0;
    for (let i = 0; i < input.length; i++) maxErr = Math.max(maxErr, Math.abs(input[i] - back[i]));
    expect(maxErr).toBeLessThan(2 / 32768);
  });
});

describe("int16ToBytes / bytesToInt16", () => {
  it("é little-endian, byte a byte", () => {
    const bytes = new Uint8Array(int16ToBytes(new Int16Array([0x0102, -2])));
    expect(Array.from(bytes)).toEqual([0x02, 0x01, 0xfe, 0xff]);
  });

  it("ida e volta é idêntica", () => {
    const input = new Int16Array([0, 1, -1, 32767, -32768, 1234]);
    expect(Array.from(bytesToInt16(int16ToBytes(input)))).toEqual(Array.from(input));
  });

  it("um quadro tem exatamente 640 bytes", () => {
    expect(int16ToBytes(new Int16Array(FRAME_SAMPLES)).byteLength).toBe(FRAME_BYTES);
    expect(FRAME_BYTES).toBe(640);
  });

  it("byte sobrando no fim é ignorado", () => {
    expect(bytesToInt16(new ArrayBuffer(5)).length).toBe(2);
  });
});

describe("Framer", () => {
  it("entrega só quadros exatos de 320 e guarda o resto", () => {
    const f = new Framer();
    expect(f.push(new Int16Array(100))).toHaveLength(0);
    expect(f.buffered).toBe(100);
    const frames = f.push(new Int16Array(600)); // 700 no total = 2 quadros + 60
    expect(frames).toHaveLength(2);
    frames.forEach((fr) => expect(fr.length).toBe(FRAME_SAMPLES));
    expect(f.buffered).toBe(60);
  });

  it("preserva a ordem das amostras entre as chamadas", () => {
    const f = new Framer();
    const input = new Int16Array(FRAME_SAMPLES * 2);
    input.forEach((_, i) => (input[i] = i));
    const a = f.push(input.slice(0, 500));
    const b = f.push(input.slice(500));
    const all = [...a, ...b];
    expect(all).toHaveLength(2);
    expect(all[0][0]).toBe(0);
    expect(all[0][FRAME_SAMPLES - 1]).toBe(FRAME_SAMPLES - 1);
    expect(all[1][0]).toBe(FRAME_SAMPLES);
  });

  it("128 amostras do AudioWorklet viram quadros de 320 sem perder nenhuma", () => {
    const f = new Framer();
    let total = 0;
    for (let i = 0; i < 100; i++) total += f.push(new Int16Array(128)).length * FRAME_SAMPLES;
    expect(total + f.buffered).toBe(12800);
  });
});

describe("rms", () => {
  it("silêncio é 0 e vazio não quebra", () => {
    expect(rms(new Int16Array(100))).toBe(0);
    expect(rms(new Int16Array(0))).toBe(0);
  });

  it("onda quadrada de meia escala dá 0,5", () => {
    const x = new Int16Array(100).fill(16384);
    expect(rms(x)).toBeCloseTo(0.5, 3);
  });
});
