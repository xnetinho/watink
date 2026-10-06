import { describe, expect, it } from "vitest";
import { classifyMicError } from "../useCallAudio";
import { CAPTURE_PROCESSOR, PLAYBACK_PROCESSOR, WORKLET_SOURCES } from "../worklet";

describe("classifyMicError", () => {
  it("permissão negada é diferente de sem dispositivo", () => {
    expect(classifyMicError({ name: "NotAllowedError" })).toBe("mic_denied");
    expect(classifyMicError({ name: "SecurityError" })).toBe("mic_denied");
    expect(classifyMicError({ name: "PermissionDeniedError" })).toBe("mic_denied");
    expect(classifyMicError({ name: "NotFoundError" })).toBe("mic_unavailable");
    expect(classifyMicError({ name: "NotReadableError" })).toBe("mic_unavailable");
  });
  it("erro estranho ou nulo cai em indisponível, nunca quebra", () => {
    expect(classifyMicError(null)).toBe("mic_unavailable");
    expect(classifyMicError(undefined)).toBe("mic_unavailable");
    expect(classifyMicError("x")).toBe("mic_unavailable");
  });
});

// O código dos worklets roda fora do bundler, como texto: aqui se garante que é
// JavaScript sintaticamente válido e que registra o nome que o hook espera.
describe("código dos worklets", () => {
  it.each([
    ["captura", WORKLET_SOURCES.CAPTURE, CAPTURE_PROCESSOR],
    ["reprodução", WORKLET_SOURCES.PLAYBACK, PLAYBACK_PROCESSOR],
  ])("%s é JS válido e registra o processador certo", (_n, src, name) => {
    expect(() => new Function("AudioWorkletProcessor", "registerProcessor", "sampleRate", src)).not.toThrow();
    const registered: string[] = [];
    class Base { port = { postMessage: () => undefined, onmessage: null as unknown } }
    new Function("AudioWorkletProcessor", "registerProcessor", "sampleRate", src)(Base, (n: string) => registered.push(n), 48000);
    expect(registered).toEqual([name]);
  });

  it("a reprodução toca o que chegou, na ordem, e silêncio quando acaba", () => {
    let Proc: new () => { port: { onmessage: ((e: { data: Float32Array }) => void) | null }; process: (i: unknown, o: Float32Array[][]) => boolean };
    class Base { port: { onmessage: ((e: { data: Float32Array }) => void) | null } = { onmessage: null } }
    new Function("AudioWorkletProcessor", "registerProcessor", "sampleRate", WORKLET_SOURCES.PLAYBACK)(
      Base,
      (_n: string, c: typeof Proc) => (Proc = c),
      48000,
    );
    const p = new Proc!();
    p.port.onmessage!({ data: new Float32Array([1, 2, 3]) });
    p.port.onmessage!({ data: new Float32Array([4, 5]) });
    const out = new Float32Array(8).fill(9);
    p.process(null, [[out]]);
    expect(Array.from(out)).toEqual([1, 2, 3, 4, 5, 0, 0, 0]);
  });

  it("a captura devolve uma CÓPIA do bloco (o buffer do worklet é reutilizado)", () => {
    let Proc: new () => { port: { postMessage: (d: Float32Array) => void }; process: (i: Float32Array[][]) => boolean };
    class Base { port = { postMessage: (_d: Float32Array) => undefined } }
    new Function("AudioWorkletProcessor", "registerProcessor", "sampleRate", WORKLET_SOURCES.CAPTURE)(
      Base,
      (_n: string, c: typeof Proc) => (Proc = c),
      48000,
    );
    const p = new Proc!();
    const sent: Float32Array[] = [];
    p.port.postMessage = (d) => sent.push(d);
    const block = new Float32Array([0.1, 0.2, 0.3]);
    p.process([[block]]);
    block.fill(0);
    expect(Array.from(sent[0])).toEqual([expect.closeTo(0.1), expect.closeTo(0.2), expect.closeTo(0.3)]);
  });
});
