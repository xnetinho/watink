// Forma de onda de um áudio, calculada no navegador a partir do próprio arquivo. O WhatsApp Web desenha a onda que vem
// na mensagem; o Watink não a guarda (enviadas e recebidas), então ela é derivada do arquivo e funciona para qualquer
// áudio, antigo ou novo. Resultado em cache por URL: reabrir o chat não decodifica de novo.

/** Quantas barras o player desenha (o cliente oficial usa 64). */
export const WAVEFORM_BARS = 64;

/** Acima disso não decodifica (um áudio de 15 MB vira centenas de MB de PCM na memória): fica com a onda de repouso. */
const MAX_DECODE_BYTES = 12 * 1024 * 1024;

const cache = new Map<string, number[]>();

/**
 * Reduz amostras PCM (canal único, -1..1) a `bars` valores de 0..1, pelo valor RMS (energia média) de cada janela, como o WhatsApp, e normalizados pelo maior; o pico achata áudio alto.
 * Áudio mudo devolve barras zeradas (nunca divide por zero); menos amostras que barras repete a última janela.
 */
export function peaksFromSamples(samples: Float32Array, bars: number = WAVEFORM_BARS): number[] {
  const out = new Array<number>(bars).fill(0);
  const n = samples.length;
  if (n === 0 || bars <= 0) return out;
  const win = n / bars;
  let max = 0;
  for (let i = 0; i < bars; i++) {
    const start = Math.floor(i * win);
    const end = Math.max(start + 1, Math.min(n, Math.floor((i + 1) * win)));
    let sum = 0;
    let count = 0;
    for (let j = start; j < end && j < n; j++) {
      sum += samples[j] * samples[j];
      count++;
    }
    const peak = count ? Math.sqrt(sum / count) : 0;
    out[i] = peak;
    if (peak > max) max = peak;
  }
  if (max === 0) return out;
  return out.map((p) => p / max);
}

/** Mistura os canais em um só (média), para o pico refletir o que se ouve. */
export function mixToMono(channels: Float32Array[]): Float32Array {
  if (channels.length === 1) return channels[0];
  const len = channels[0]?.length ?? 0;
  const mono = new Float32Array(len);
  for (const ch of channels) for (let i = 0; i < len; i++) mono[i] += ch[i] / channels.length;
  return mono;
}

/** Onda "de repouso": barras baixas e parelhas, usada enquanto carrega ou quando não dá para decodificar. */
export const restingWaveform = (bars: number = WAVEFORM_BARS): number[] => new Array<number>(bars).fill(0.18);

type DecodeContext = { decodeAudioData(data: ArrayBuffer): Promise<AudioBuffer>; close?: () => Promise<void> };

/**
 * Baixa o áudio e calcula a onda. Devolve `null` quando não dá (sem Web Audio, arquivo grande demais, formato que o
 * navegador não decodifica, rede): quem chama mostra a onda de repouso e o player continua funcionando normalmente.
 */
export async function loadWaveform(
  url: string,
  bars: number = WAVEFORM_BARS,
  deps: { fetchFn?: typeof fetch; makeContext?: () => DecodeContext | null } = {},
): Promise<number[] | null> {
  const key = `${url}#${bars}`;
  const hit = cache.get(key);
  if (hit) return hit;

  const fetchFn = deps.fetchFn ?? (typeof fetch === "function" ? fetch : undefined);
  const makeContext =
    deps.makeContext ??
    (() => {
      const Ctor = (globalThis as { AudioContext?: new () => DecodeContext; webkitAudioContext?: new () => DecodeContext }).AudioContext ??
        (globalThis as { webkitAudioContext?: new () => DecodeContext }).webkitAudioContext;
      return Ctor ? new Ctor() : null;
    });
  if (!fetchFn || !url) return null;

  let ctx: DecodeContext | null = null;
  try {
    const res = await fetchFn(url);
    if (!res.ok) return null;
    const declared = Number(res.headers?.get?.("content-length") ?? 0);
    if (declared > MAX_DECODE_BYTES) return null;
    const buf = await res.arrayBuffer();
    if (buf.byteLength === 0 || buf.byteLength > MAX_DECODE_BYTES) return null;

    ctx = makeContext();
    if (!ctx) return null;
    const audio = await ctx.decodeAudioData(buf);
    const channels: Float32Array[] = [];
    for (let c = 0; c < audio.numberOfChannels; c++) channels.push(audio.getChannelData(c));
    if (channels.length === 0) return null;
    const peaks = peaksFromSamples(mixToMono(channels), bars);
    cache.set(key, peaks);
    return peaks;
  } catch {
    return null;
  } finally {
    void ctx?.close?.().catch(() => undefined);
  }
}

/** Só para testes. */
export const __clearWaveformCache = (): void => cache.clear();
