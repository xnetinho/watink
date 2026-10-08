import { describe, expect, it, vi } from "vitest";
import {
  CAMERA,
  CameraSender,
  classifyCameraError,
  type CameraDeps,
  type CameraFailure,
  type CameraFrameLike,
  type EncodedChunkLike,
  type EncoderLike,
  type ReaderLike,
} from "../cameraSender";
import { decodeVideoFrame } from "../videoFrame";

/** Câmera falsa: o teste empurra quadros; `read()` espera o próximo. */
class FakeReader implements ReaderLike {
  private queue: CameraFrameLike[] = [];
  private waiter: ((r: { done: boolean; value?: CameraFrameLike }) => void) | null = null;
  cancelled = false;
  push(f: CameraFrameLike) {
    if (this.waiter) {
      const w = this.waiter;
      this.waiter = null;
      w({ done: false, value: f });
    } else this.queue.push(f);
  }
  read() {
    const f = this.queue.shift();
    if (f) return Promise.resolve({ done: false, value: f });
    if (this.cancelled) return Promise.resolve({ done: true });
    return new Promise<{ done: boolean; value?: CameraFrameLike }>((res) => { this.waiter = res; });
  }
  cancel() {
    this.cancelled = true;
    if (this.waiter) { this.waiter({ done: true }); this.waiter = null; }
  }
}

class FakeEncoder implements EncoderLike {
  encodeQueueSize = 0;
  state = "unconfigured";
  config: Record<string, unknown> | null = null;
  encoded: { keyFrame: boolean }[] = [];
  closed = false;
  throwOnEncode = false;
  constructor(public init: { output: (c: EncodedChunkLike) => void; error: (e: Error) => void }) {}
  configure(c: Record<string, unknown>) { this.config = c; this.state = "configured"; }
  encode(_f: CameraFrameLike, o?: { keyFrame?: boolean }) {
    if (this.throwOnEncode) throw new Error("boom");
    this.encoded.push({ keyFrame: !!o?.keyFrame });
  }
  close() { this.closed = true; this.state = "closed"; }
  emit(type: "key" | "delta", timestampUs: number, bytes: number[]) {
    const data = new Uint8Array(bytes);
    this.init.output({ type, timestamp: timestampUs, byteLength: data.length, copyTo: (d) => d.set(data) });
  }
}

const frame = (): CameraFrameLike & { closed: boolean } => {
  const f = { timestamp: 0, closed: false, close() { f.closed = true; } };
  return f;
};

function rig(opts: { denyWith?: string; noTrack?: boolean } = {}) {
  // Cada start() recebe um leitor novo, como o MediaStreamTrackProcessor real (um leitor cancelado morre).
  let reader = new FakeReader();
  const readers: FakeReader[] = [reader];
  const stops: number[] = [];
  let enc!: FakeEncoder;
  let t = 1000;
  const deps: CameraDeps = {
    getStream: vi.fn(async () => {
      if (opts.denyWith) throw Object.assign(new Error("x"), { name: opts.denyWith });
      const track = { stop: () => stops.push(1) };
      return { getVideoTracks: () => (opts.noTrack ? [] : [track]), getTracks: () => [track] };
    }),
    readerFor: () => {
      if (readers.length > 1 || reader.cancelled) { reader = new FakeReader(); }
      readers.push(reader);
      return reader;
    },
    createEncoder: (init) => (enc = new FakeEncoder(init)),
    now: () => t,
  };
  const sent: Uint8Array[] = [];
  const failures: CameraFailure[] = [];
  const sender = new CameraSender({ deps, send: (m) => sent.push(m), onFailure: (r) => failures.push(r) });
  return { sender, get reader() { return reader; }, deps, stops, sent, failures, enc: () => enc, advance: (ms: number) => { t += ms; } };
}

const tick = () => new Promise((r) => setTimeout(r, 0));

describe("classifyCameraError", () => {
  it("separa permissão negada de câmera ausente ou ocupada", () => {
    expect(classifyCameraError({ name: "NotAllowedError" })).toBe("denied");
    expect(classifyCameraError({ name: "SecurityError" })).toBe("denied");
    expect(classifyCameraError({ name: "NotFoundError" })).toBe("unavailable");
    expect(classifyCameraError({ name: "NotReadableError" })).toBe("unavailable");
    expect(classifyCameraError(null)).toBe("unavailable");
  });
});

describe("CameraSender", () => {
  it("pede a câmera em 640x480/15 fps e configura H.264 Annex-B em tempo real", async () => {
    const r = rig();
    expect(await r.sender.start()).toBe(true);
    const c = (r.deps.getStream as ReturnType<typeof vi.fn>).mock.calls[0][0] as { audio: boolean; video: { width: { ideal: number } } };
    expect(c.audio).toBe(false);
    expect(c.video.width.ideal).toBe(640);
    expect(r.enc().config).toMatchObject({ codec: "avc1.42E01F", width: 640, height: 480, bitrate: 600000, latencyMode: "realtime", avc: { format: "annexb" } });
    r.sender.stop();
  });

  it("o primeiro quadro codificado sai como quadro-chave, e o seguinte como delta", async () => {
    const r = rig();
    await r.sender.start();
    r.reader.push(frame());
    await tick();
    r.advance(70);
    r.reader.push(frame());
    await tick();
    expect(r.enc().encoded.map((e) => e.keyFrame)).toEqual([true, false]);
    r.sender.stop();
  });

  it("gera um quadro-chave a cada 2 s mesmo sem ninguém pedir", async () => {
    const r = rig();
    await r.sender.start();
    r.reader.push(frame());
    await tick();
    r.advance(1000);
    r.reader.push(frame());
    await tick();
    r.advance(1100);
    r.reader.push(frame());
    await tick();
    expect(r.enc().encoded.map((e) => e.keyFrame)).toEqual([true, false, true]);
    r.sender.stop();
  });

  it("requestKeyframe força o próximo quadro a ser quadro-chave (PLI do contato)", async () => {
    const r = rig();
    await r.sender.start();
    r.reader.push(frame());
    await tick();
    r.advance(70);
    r.sender.requestKeyframe();
    r.reader.push(frame());
    await tick();
    expect(r.enc().encoded.map((e) => e.keyFrame)).toEqual([true, true]);
    r.sender.stop();
  });

  it("descarta quadros que chegam rápido demais para os 15 fps e fecha todos os quadros", async () => {
    const r = rig();
    await r.sender.start();
    const a = frame(), b = frame(), c = frame();
    r.reader.push(a);
    await tick();
    r.advance(10);
    r.reader.push(b);
    await tick();
    r.advance(70);
    r.reader.push(c);
    await tick();
    expect(r.enc().encoded).toHaveLength(2);
    expect([a.closed, b.closed, c.closed]).toEqual([true, true, true]);
    r.sender.stop();
  });

  it("com o codificador atrasado descarta o quadro da câmera em vez de acumular", async () => {
    const r = rig();
    await r.sender.start();
    r.enc().encodeQueueSize = CAMERA.maxEncodeQueue + 1;
    const f = frame();
    r.reader.push(f);
    await tick();
    expect(r.enc().encoded).toHaveLength(0);
    expect(f.closed).toBe(true);
    r.sender.stop();
  });

  it("entrega cada quadro codificado no formato do fio, com timestamp de 90 kHz relativo ao primeiro", async () => {
    const r = rig();
    await r.sender.start();
    r.sender.setOrientation(1);
    r.enc().emit("key", 5_000_000, [0, 0, 0, 1, 0x65, 9]);
    r.enc().emit("delta", 5_066_667, [0, 0, 0, 1, 0x41, 8]);
    const [k, d] = r.sent.map((m) => decodeVideoFrame(m)!);
    expect(k.keyframe).toBe(true);
    expect(k.ts90k).toBe(0);
    expect(k.rotation).toBe(1);
    expect(Array.from(k.data)).toEqual([0, 0, 0, 1, 0x65, 9]);
    expect(d.keyframe).toBe(false);
    expect(d.ts90k).toBe(6000);
    r.sender.stop();
  });

  it("permissão negada não liga nada e avisa", async () => {
    const r = rig({ denyWith: "NotAllowedError" });
    expect(await r.sender.start()).toBe(false);
    expect(r.failures).toEqual(["denied"]);
    expect(r.sender.active).toBe(false);
  });

  it("câmera inexistente avisa 'unavailable'", async () => {
    const r = rig({ denyWith: "NotFoundError" });
    expect(await r.sender.start()).toBe(false);
    expect(r.failures).toEqual(["unavailable"]);
  });

  it("stream sem faixa de vídeo libera o que abriu e avisa", async () => {
    const r = rig({ noTrack: true });
    expect(await r.sender.start()).toBe(false);
    expect(r.failures).toEqual(["unavailable"]);
    expect(r.stops).toHaveLength(1);
  });

  it("erro do codificador desliga a câmera e avisa uma vez", async () => {
    const r = rig();
    await r.sender.start();
    r.enc().init.error(new Error("hw"));
    r.enc().init.error(new Error("hw"));
    expect(r.failures).toEqual(["encoder"]);
    expect(r.sender.active).toBe(false);
    expect(r.enc().closed).toBe(true);
  });

  it("exceção ao codificar desliga a câmera", async () => {
    const r = rig();
    await r.sender.start();
    r.enc().throwOnEncode = true;
    r.reader.push(frame());
    await tick();
    expect(r.failures).toEqual(["encoder"]);
    expect(r.sender.active).toBe(false);
  });

  it("stop libera câmera, leitor e codificador, e nada mais sai depois", async () => {
    const r = rig();
    await r.sender.start();
    r.sender.stop();
    expect(r.stops).toHaveLength(1);
    expect(r.reader.cancelled).toBe(true);
    expect(r.enc().closed).toBe(true);
    expect(r.sender.active).toBe(false);
    r.enc().emit("key", 0, [1]);
    expect(r.sent).toHaveLength(0);
  });

  it("start duas vezes não abre a câmera duas vezes", async () => {
    const r = rig();
    await r.sender.start();
    await r.sender.start();
    expect(r.deps.getStream).toHaveBeenCalledTimes(1);
    r.sender.stop();
  });

  it("depois de parar dá para ligar de novo, e o primeiro quadro volta a ser quadro-chave", async () => {
    const r = rig();
    await r.sender.start();
    r.reader.push(frame());
    await tick();
    r.sender.stop();
    expect(await r.sender.start()).toBe(true);
    r.advance(10);
    r.reader.push(frame());
    await tick();
    expect(r.enc().encoded.map((e) => e.keyFrame)).toEqual([true]);
    r.sender.stop();
  });
});
