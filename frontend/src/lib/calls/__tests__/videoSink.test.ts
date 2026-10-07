import { describe, expect, it, vi } from "vitest";
import { VideoSink } from "../videoSink";

class Dec {
  static last: Dec | null = null;
  state = "unconfigured";
  chunks: unknown[] = [];
  constructor(public init: { output: (f: unknown) => void; error: (e: Error) => void }) { Dec.last = this; }
  configure() { this.state = "configured"; }
  decode(c: unknown) { this.chunks.push(c); }
  close() { this.state = "closed"; }
}
class Chunk { constructor(public i: unknown) {} }

const frameOut = () => ({ displayWidth: 640, displayHeight: 480, close: vi.fn() });
const mkCanvas = () => {
  const drawImage = vi.fn();
  const c = { width: 0, height: 0, getContext: () => ({ drawImage }) } as unknown as HTMLCanvasElement;
  return { c, drawImage };
};
const key = { keyframe: true, ts90k: 1, data: new Uint8Array([0, 0, 0, 1, 0x65, 1]) };

describe("VideoSink", () => {
  it("desenha o quadro decodificado no canvas, ajusta o tamanho e libera o quadro", () => {
    const sink = new VideoSink(Dec as never, Chunk as never);
    const { c, drawImage } = mkCanvas();
    sink.attach(c);
    sink.push(key);
    const f = frameOut();
    Dec.last!.init.output(f);
    expect(c.width).toBe(640);
    expect(c.height).toBe(480);
    expect(drawImage).toHaveBeenCalledTimes(1);
    expect(f.close).toHaveBeenCalledTimes(1);
  });

  it("libera o quadro mesmo se desenhar falhar (senão vaza memória de GPU)", () => {
    const sink = new VideoSink(Dec as never, Chunk as never);
    const c = { width: 0, height: 0, getContext: () => ({ drawImage: () => { throw new Error("x"); } }) } as unknown as HTMLCanvasElement;
    sink.attach(c);
    sink.push(key);
    const f = frameOut();
    expect(() => Dec.last!.init.output(f)).toThrow();
    expect(f.close).toHaveBeenCalledTimes(1);
  });

  it("sem canvas registrado, descarta o quadro sem criar decoder", () => {
    Dec.last = null;
    const sink = new VideoSink(Dec as never, Chunk as never);
    sink.push(key);
    expect(Dec.last).toBeNull();
  });

  it("sem suporte do navegador, não faz nada e informa", () => {
    const sink = new VideoSink(undefined, undefined);
    expect(sink.supported).toBe(false);
    sink.attach({} as HTMLCanvasElement);
    expect(() => sink.push(key)).not.toThrow();
  });

  it("o pedido de quadro-chave do decoder sobe pelo sink", () => {
    const sink = new VideoSink(Dec as never, Chunk as never);
    const need = vi.fn();
    sink.onNeedKeyframe = need;
    sink.attach(mkCanvas().c);
    sink.push(key);
    Dec.last!.init.error(new Error("boom"));
    expect(need).toHaveBeenCalled();
  });

  it("close libera o decoder e esquece o canvas", () => {
    const sink = new VideoSink(Dec as never, Chunk as never);
    sink.attach(mkCanvas().c);
    sink.push(key);
    const d = Dec.last!;
    sink.close();
    expect(d.state).toBe("closed");
    Dec.last = null;
    sink.push(key);
    expect(Dec.last).toBeNull();
  });
});
