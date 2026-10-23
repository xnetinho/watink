import { beforeEach, describe, expect, it, vi } from "vitest";
import { VideoPlayback } from "../videoPlayback";

// Decoder falso: registra o que recebe e deixa o teste disparar erro/saída.
interface FakeChunk { type: "key" | "delta"; timestamp: number; data: Uint8Array }
class FakeDecoder {
  static instances: FakeDecoder[] = [];
  state: "unconfigured" | "configured" | "closed" = "unconfigured";
  chunks: FakeChunk[] = [];
  config: unknown = null;
  constructor(public init: { output: (f: unknown) => void; error: (e: Error) => void }) {
    FakeDecoder.instances.push(this);
  }
  configure(c: unknown) { this.config = c; this.state = "configured"; }
  decode(c: FakeChunk) { this.chunks.push(c); }
  close() { this.state = "closed"; }
}
class FakeChunkCtor {
  type: "key" | "delta"; timestamp: number; data: Uint8Array;
  constructor(i: { type: "key" | "delta"; timestamp: number; data: Uint8Array }) { this.type = i.type; this.timestamp = i.timestamp; this.data = i.data; }
}

const au = (b: number) => new Uint8Array([0, 0, 0, 1, b, 1, 2, 3]);
const make = () => {
  const frames: unknown[] = [];
  const p = new VideoPlayback({
    DecoderCtor: FakeDecoder as unknown as ConstructorParameters<typeof VideoPlayback>[0]["DecoderCtor"],
    ChunkCtor: FakeChunkCtor as unknown as ConstructorParameters<typeof VideoPlayback>[0]["ChunkCtor"],
    onFrame: (f) => frames.push(f),
  });
  return { p, frames, dec: () => FakeDecoder.instances[FakeDecoder.instances.length - 1] };
};

describe("VideoPlayback", () => {
  beforeEach(() => { FakeDecoder.instances = []; });

  it("descarta quadros delta até o primeiro quadro-chave", () => {
    const { p, dec } = make();
    p.push({ keyframe: false, rotation: 0, ts90k: 1000, data: au(0x41) });
    p.push({ keyframe: false, rotation: 0, ts90k: 2000, data: au(0x41) });
    expect(dec()?.chunks ?? []).toHaveLength(0);
    p.push({ keyframe: true, rotation: 0, ts90k: 3000, data: au(0x65) });
    p.push({ keyframe: false, rotation: 0, ts90k: 4000, data: au(0x41) });
    expect(dec().chunks.map((c) => c.type)).toEqual(["key", "delta"]);
  });

  it("configura o decoder para H.264 Constrained Baseline 3.1", () => {
    const { p, dec } = make();
    p.push({ keyframe: true, rotation: 0, ts90k: 1, data: au(0x65) });
    expect((dec().config as { codec: string }).codec).toBe("avc1.42E01F");
  });

  it("converte o relógio de 90 kHz para microssegundos (o que o WebCodecs espera)", () => {
    const { p, dec } = make();
    p.push({ keyframe: true, rotation: 0, ts90k: 90000, data: au(0x65) });
    expect(dec().chunks[0].timestamp).toBe(1_000_000);
  });

  it("erro do decoder: recria e volta a exigir quadro-chave (não decodifica delta com estado quebrado)", () => {
    const { p, dec } = make();
    const need = vi.fn();
    p.onNeedKeyframe = need;
    p.push({ keyframe: true, rotation: 0, ts90k: 1, data: au(0x65) });
    const first = dec();
    first.init.error(new Error("boom"));
    expect(first.state).toBe("closed");
    expect(need).toHaveBeenCalledTimes(1);

    // um delta logo depois do erro: o decoder velho está fechado e NÃO se cria outro só por causa dele
    p.push({ keyframe: false, rotation: 0, ts90k: 2, data: au(0x41) });
    expect(FakeDecoder.instances).toHaveLength(1);
    expect(first.chunks.map((c) => c.type)).toEqual(["key"]);

    // só o próximo quadro-chave cria o decoder novo e recomeça
    p.push({ keyframe: true, rotation: 0, ts90k: 3, data: au(0x65) });
    expect(FakeDecoder.instances).toHaveLength(2);
    expect(dec().chunks.map((c) => c.type)).toEqual(["key"]);
  });

  it("fila de decodificação cheia: pede quadro-chave em vez de acumular atraso", () => {
    const { p } = make();
    const need = vi.fn();
    p.onNeedKeyframe = need;
    p.push({ keyframe: true, rotation: 0, ts90k: 1, data: au(0x65) });
    p.decodeQueueSize = () => 50;
    p.push({ keyframe: false, rotation: 0, ts90k: 2, data: au(0x41) });
    expect(need).toHaveBeenCalled();
  });

  it("close libera o decoder e ignora quadros depois", () => {
    const { p, dec } = make();
    p.push({ keyframe: true, rotation: 0, ts90k: 1, data: au(0x65) });
    const d = dec();
    p.close();
    expect(d.state).toBe("closed");
    p.push({ keyframe: true, rotation: 0, ts90k: 2, data: au(0x65) });
    expect(FakeDecoder.instances.length).toBe(1);
  });

  it("sem suporte (sem construtor de decoder): não quebra e não decodifica", () => {
    const frames: unknown[] = [];
    const p = new VideoPlayback({ DecoderCtor: undefined, ChunkCtor: undefined, onFrame: (f) => frames.push(f) });
    expect(() => p.push({ keyframe: true, rotation: 0, ts90k: 1, data: au(0x65) })).not.toThrow();
    expect(p.supported).toBe(false);
  });
});
