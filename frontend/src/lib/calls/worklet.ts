// Código dos AudioWorklets das chamadas. Roda no thread de áudio, em outro
// contexto, então vai como TEXTO carregado por Blob URL: assim não existe arquivo
// .js em src/ (política do projeto) e não depende do bundler para empacotar.

/** Captura: converte o microfone em blocos Float32 e os devolve ao thread principal. */
const CAPTURE = `
class CaptureProcessor extends AudioWorkletProcessor {
  process(inputs) {
    const ch = inputs[0] && inputs[0][0];
    if (ch && ch.length) this.port.postMessage(ch.slice(0));
    return true;
  }
}
registerProcessor('call-capture', CaptureProcessor);
`;

/**
 * Reprodução: consome Float32 já na taxa do contexto, empurrado pelo thread
 * principal. Fila limitada (~1 s) para um thread principal travado não acumular
 * atraso; sem dados toca silêncio.
 */
const PLAYBACK = `
class PlaybackProcessor extends AudioWorkletProcessor {
  constructor() {
    super();
    this.queue = [];
    this.pos = 0;
    this.cap = sampleRate;
    this.have = 0;
    this.port.onmessage = (e) => {
      this.queue.push(e.data);
      this.have += e.data.length;
      while (this.have > this.cap && this.queue.length > 1) {
        const old = this.queue.shift();
        this.have -= old.length - this.pos;
        this.pos = 0;
      }
    };
  }
  process(_i, outputs) {
    const out = outputs[0][0];
    let n = 0;
    while (n < out.length && this.queue.length) {
      const cur = this.queue[0];
      const take = Math.min(out.length - n, cur.length - this.pos);
      out.set(cur.subarray(this.pos, this.pos + take), n);
      n += take;
      this.pos += take;
      this.have -= take;
      if (this.pos >= cur.length) { this.queue.shift(); this.pos = 0; }
    }
    for (; n < out.length; n++) out[n] = 0;
    return true;
  }
}
registerProcessor('call-playback', PlaybackProcessor);
`;

export const CAPTURE_PROCESSOR = "call-capture";
export const PLAYBACK_PROCESSOR = "call-playback";

function blobUrl(code: string): string {
  return URL.createObjectURL(new Blob([code], { type: "application/javascript" }));
}

/** Registra os dois processadores no contexto de áudio. */
export async function loadCallWorklets(ctx: AudioContext): Promise<void> {
  const urls = [blobUrl(CAPTURE), blobUrl(PLAYBACK)];
  try {
    for (const u of urls) await ctx.audioWorklet.addModule(u);
  } finally {
    urls.forEach((u) => URL.revokeObjectURL(u));
  }
}

export const WORKLET_SOURCES = { CAPTURE, PLAYBACK };
