import { beforeEach, describe, expect, it, vi } from "vitest";
import { __clearWaveformCache, loadWaveform, mixToMono, peaksFromSamples, restingWaveform, WAVEFORM_BARS } from "../waveform";

describe("peaksFromSamples", () => {
  it("devolve sempre o número de barras pedido, normalizado em 0..1 (a mais alta vale 1)", () => {
    const s = new Float32Array(6400);
    s[100] = 0.2; // janela 1
    s[3200] = -0.8; // pico negativo conta pelo módulo
    const p = peaksFromSamples(s);
    expect(p).toHaveLength(WAVEFORM_BARS);
    expect(Math.max(...p)).toBe(1);
    expect(p[1]).toBeCloseTo(0.25, 5);
    expect(p[32]).toBe(1);
    expect(p.every((v) => v >= 0 && v <= 1)).toBe(true);
  });

  it("áudio alto não vira bloco: janela cheia de picos ainda é maior que a de fala baixa", () => {
    const s = new Float32Array(6400);
    for (let i = 0; i < 3200; i++) s[i] = i % 2 ? 1 : -1; // metade alta o tempo todo
    for (let i = 3200; i < 6400; i++) s[i] = i % 10 === 0 ? 1 : 0.1; // resto: picos isolados sobre ruído baixo
    const p = peaksFromSamples(s);
    expect(p[0]).toBe(1);
    expect(p[63]).toBeLessThan(0.5);
  });

  it("silêncio devolve barras zeradas, sem dividir por zero", () => {
    expect(peaksFromSamples(new Float32Array(1000)).every((v) => v === 0)).toBe(true);
  });

  it("menos amostras que barras e entrada vazia não estouram", () => {
    expect(peaksFromSamples(new Float32Array([0.5, 1]))).toHaveLength(WAVEFORM_BARS);
    expect(peaksFromSamples(new Float32Array(0))).toEqual(new Array(WAVEFORM_BARS).fill(0));
  });
});

describe("mixToMono / restingWaveform", () => {
  it("mono passa direto; estéreo faz a média", () => {
    const l = new Float32Array([1, 0]);
    expect(mixToMono([l])).toBe(l);
    expect(Array.from(mixToMono([new Float32Array([1, 0]), new Float32Array([0, 1])]))).toEqual([0.5, 0.5]);
  });
  it("onda de repouso tem o tamanho certo e é baixa", () => {
    const r = restingWaveform();
    expect(r).toHaveLength(WAVEFORM_BARS);
    expect(Math.max(...r)).toBeLessThan(0.3);
  });
});

const audioBuffer = (data: Float32Array[]) => ({ numberOfChannels: data.length, getChannelData: (c: number) => data[c] });
const okResponse = (bytes = 100, headers: Record<string, string> = {}) => ({
  ok: true,
  headers: { get: (k: string) => headers[k.toLowerCase()] ?? null },
  arrayBuffer: async () => new ArrayBuffer(bytes),
});

describe("loadWaveform", () => {
  beforeEach(() => __clearWaveformCache());

  it("baixa, decodifica e devolve as barras; reusa o cache na segunda chamada", async () => {
    const fetchFn = vi.fn().mockResolvedValue(okResponse());
    const decodeAudioData = vi.fn().mockResolvedValue(audioBuffer([new Float32Array(640).fill(0.5)]));
    const deps = { fetchFn: fetchFn as unknown as typeof fetch, makeContext: () => ({ decodeAudioData }) };
    const a = await loadWaveform("/a.ogg", 64, deps);
    const b = await loadWaveform("/a.ogg", 64, deps);
    expect(a).toHaveLength(64);
    expect(b).toBe(a);
    expect(fetchFn).toHaveBeenCalledTimes(1);
  });

  it("devolve null (e o player usa a onda de repouso) quando: HTTP falha, formato não decodifica, arquivo grande, sem Web Audio, sem URL", async () => {
    const mk = (fetchFn: unknown, ctx: unknown = { decodeAudioData: vi.fn().mockResolvedValue(audioBuffer([new Float32Array(10)])) }) =>
      ({ fetchFn: fetchFn as typeof fetch, makeContext: () => ctx as never });
    expect(await loadWaveform("/1", 64, mk(vi.fn().mockResolvedValue({ ok: false })))).toBeNull();
    expect(await loadWaveform("/2", 64, mk(vi.fn().mockResolvedValue(okResponse()), { decodeAudioData: vi.fn().mockRejectedValue(new Error("EncodingError")) }))).toBeNull();
    expect(await loadWaveform("/3", 64, mk(vi.fn().mockResolvedValue(okResponse(10, { "content-length": String(50 * 1024 * 1024) }))))).toBeNull();
    expect(await loadWaveform("/4", 64, mk(vi.fn().mockResolvedValue(okResponse(0))))).toBeNull();
    expect(await loadWaveform("/5", 64, { fetchFn: vi.fn().mockResolvedValue(okResponse()) as unknown as typeof fetch, makeContext: () => null })).toBeNull();
    expect(await loadWaveform("/6", 64, mk(vi.fn().mockRejectedValue(new Error("rede"))))).toBeNull();
    expect(await loadWaveform("", 64, mk(vi.fn()))).toBeNull();
  });

  it("fecha o contexto de áudio mesmo quando a decodificação falha (não vaza)", async () => {
    const close = vi.fn().mockResolvedValue(undefined);
    await loadWaveform("/x", 64, { fetchFn: vi.fn().mockResolvedValue(okResponse()) as unknown as typeof fetch, makeContext: () => ({ decodeAudioData: vi.fn().mockRejectedValue(new Error("x")), close }) });
    expect(close).toHaveBeenCalled();
  });
});
