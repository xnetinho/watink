import { describe, expect, it, vi } from "vitest";
import { VideoSink } from "../videoSink";
import type { VideoFrame } from "../videoFrame";

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

const frameOut = (timestamp?: number) => ({ displayWidth: 640, displayHeight: 480, timestamp, close: vi.fn() });
const mkCanvas = () => {
  const drawImage = vi.fn();
  const c = { width: 0, height: 0, getContext: () => ({ drawImage, setTransform: vi.fn() }) } as unknown as HTMLCanvasElement;
  return { c, drawImage };
};
const key: VideoFrame = { keyframe: true, rotation: 0, ts90k: 1, data: new Uint8Array([0, 0, 0, 1, 0x65, 1]) };

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
    const c = { width: 0, height: 0, getContext: () => ({ setTransform: vi.fn(), drawImage: () => { throw new Error("x"); } }) } as unknown as HTMLCanvasElement;
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

  // O celular em retrato manda a imagem deitada; a rotação vem em cada quadro (bits 1-2 de flags) e o
  // sink tem de girar o desenho. Sem isso a imagem aparecia virada para a esquerda.
  it("gira o desenho pela rotação do quadro e troca largura/altura do canvas nos quartos ímpares", () => {
    const sink = new VideoSink(Dec as never, Chunk as never);
    const setTransform = vi.fn();
    const drawImage = vi.fn();
    const c = { width: 0, height: 0, getContext: () => ({ setTransform, drawImage }) } as unknown as HTMLCanvasElement;
    sink.attach(c);
    sink.push({ ...key, rotation: 3 });
    Dec.last!.init.output(frameOut()); // 640×480
    expect([c.width, c.height]).toEqual([480, 640]);
    expect(setTransform).toHaveBeenCalledWith(0, -1, 1, 0, 0, 640);
    expect(drawImage).toHaveBeenCalledWith(expect.anything(), 0, 0);
  });

  it("rotação 0 mantém o tamanho e usa a matriz identidade", () => {
    const sink = new VideoSink(Dec as never, Chunk as never);
    const setTransform = vi.fn();
    const c = { width: 0, height: 0, getContext: () => ({ setTransform, drawImage: vi.fn() }) } as unknown as HTMLCanvasElement;
    sink.attach(c);
    sink.push({ ...key, rotation: 0 });
    Dec.last!.init.output(frameOut());
    expect([c.width, c.height]).toEqual([640, 480]);
    expect(setTransform).toHaveBeenCalledWith(1, 0, 0, 1, 0, 0);
  });

  it("quando o contato gira o aparelho, o desenho acompanha o quadro seguinte", () => {
    const sink = new VideoSink(Dec as never, Chunk as never);
    const setTransform = vi.fn();
    const c = { width: 0, height: 0, getContext: () => ({ setTransform, drawImage: vi.fn() }) } as unknown as HTMLCanvasElement;
    sink.attach(c);
    sink.push({ ...key, rotation: 0 });
    Dec.last!.init.output(frameOut());
    sink.push({ ...key, rotation: 1, ts90k: 2 });
    Dec.last!.init.output(frameOut());
    expect([c.width, c.height]).toEqual([480, 640]);
    expect(setTransform).toHaveBeenLastCalledWith(0, 1, -1, 0, 480, 0);
  });

  // O decoder devolve o quadro DEPOIS de recebê-lo. Se o contato gira o aparelho entre dois quadros em voo,
  // cada um deve ser desenhado com a SUA rotação, não com a do último que entrou.
  it("cada quadro em voo é desenhado com a sua própria rotação", () => {
    const sink = new VideoSink(Dec as never, Chunk as never);
    const setTransform = vi.fn();
    const c = { width: 0, height: 0, getContext: () => ({ setTransform, drawImage: vi.fn() }) } as unknown as HTMLCanvasElement;
    sink.attach(c);
    // dois quadros entram (rotação 0 e depois 1) antes de qualquer um sair do decoder
    sink.push({ ...key, rotation: 0, ts90k: 90_000 }); // timestamp 1 s
    sink.push({ ...key, rotation: 1, ts90k: 180_000, keyframe: false }); // timestamp 2 s
    // saem na ordem; o 1º tem de usar a identidade, o 2º a rotação de 90°
    Dec.last!.init.output(frameOut(1_000_000));
    expect(setTransform).toHaveBeenLastCalledWith(1, 0, 0, 1, 0, 0);
    Dec.last!.init.output(frameOut(2_000_000));
    expect(setTransform).toHaveBeenLastCalledWith(0, 1, -1, 0, 480, 0);
  });
});

