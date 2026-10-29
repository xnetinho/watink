import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

const post = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { post: (...a: unknown[]) => post(...a) } }));
const toastError = vi.fn();
vi.mock("../../../../errors/toastError", () => ({ default: (...a: unknown[]) => toastError(...a) }));

import { extensionFor, pickRecordingType, useAudioRecording } from "../useAudioRecording";

// MediaRecorder simulado: guarda o que foi gravado e entrega os pedaços só ao parar, como o de verdade.
class FakeRecorder {
  static instances: FakeRecorder[] = [];
  static supported = new Set(["audio/webm;codecs=opus"]);
  static isTypeSupported = (t: string) => FakeRecorder.supported.has(t);
  state: "inactive" | "recording" = "inactive";
  mimeType: string;
  ondataavailable: ((e: { data: Blob }) => void) | null = null;
  onstop: (() => void) | null = null;
  constructor(public stream: unknown, opts?: { mimeType?: string }) {
    this.mimeType = opts?.mimeType ?? "";
    FakeRecorder.instances.push(this);
  }
  start() {
    this.state = "recording";
  }
  stop() {
    this.state = "inactive";
    this.ondataavailable?.({ data: new Blob([new Uint8Array(4000)], { type: this.mimeType }) });
    this.onstop?.();
  }
}

const track = { stop: vi.fn() };

beforeEach(() => {
  post.mockReset().mockResolvedValue({});
  toastError.mockReset();
  track.stop.mockReset();
  FakeRecorder.instances = [];
  FakeRecorder.supported = new Set(["audio/webm;codecs=opus"]);
  vi.stubGlobal("MediaRecorder", FakeRecorder);
  Object.defineProperty(navigator, "mediaDevices", { configurable: true, value: { getUserMedia: vi.fn().mockResolvedValue({ getTracks: () => [track] }) } });
  vi.useFakeTimers();
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

const record = async (ms: number) => {
  const { result } = renderHook(() => useAudioRecording("7", vi.fn()));
  await act(async () => result.current.handleStartRecording());
  vi.advanceTimersByTime(ms);
  return result;
};

describe("gravação de voz do operador", () => {
  it("envia SEMPRE como mensagem de voz (audioKind=voice), sem legenda, com a extensão do formato gravado", async () => {
    const result = await record(3000);
    expect(result.current.recording).toBe(true);
    await act(async () => result.current.handleUploadAudio());

    expect(post).toHaveBeenCalledTimes(1);
    const [url, form] = post.mock.calls[0] as [string, FormData];
    expect(url).toBe("/messages/7");
    expect(form.get("audioKind")).toBe("voice");
    expect(form.get("body")).toBe("");
    const file = form.get("medias") as File;
    expect(file.name).toMatch(/^\d+\.webm$/);
    expect(file.size).toBe(4000);
    expect(result.current.recording).toBe(false);
  });

  it("libera o microfone depois de enviar (a luz do navegador apaga)", async () => {
    const result = await record(2000);
    await act(async () => result.current.handleUploadAudio());
    expect(track.stop).toHaveBeenCalled();
  });

  it("cancelar não envia nada e também libera o microfone", async () => {
    const result = await record(2000);
    await act(async () => result.current.handleCancelAudio());
    expect(post).not.toHaveBeenCalled();
    expect(track.stop).toHaveBeenCalled();
    expect(result.current.recording).toBe(false);
  });

  it("clique sem querer (menos de 0,5 s) não envia mensagem", async () => {
    const result = await record(100);
    await act(async () => result.current.handleUploadAudio());
    expect(post).not.toHaveBeenCalled();
    expect(result.current.recording).toBe(false);
  });

  it("microfone negado mostra o erro e não fica 'gravando'", async () => {
    (navigator.mediaDevices.getUserMedia as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error("Permission denied"));
    const { result } = renderHook(() => useAudioRecording("7", vi.fn()));
    await act(async () => result.current.handleStartRecording());
    expect(toastError).toHaveBeenCalledTimes(1);
    expect(result.current.recording).toBe(false);
  });

  it("falha no envio avisa, para de gravar e libera o microfone", async () => {
    post.mockRejectedValueOnce(new Error("rede"));
    const result = await record(2000);
    await act(async () => result.current.handleUploadAudio());
    expect(toastError).toHaveBeenCalledTimes(1);
    expect(result.current.recording).toBe(false);
    expect(track.stop).toHaveBeenCalled();
  });

  it("navegador sem MediaRecorder: erro claro em vez de travar", async () => {
    vi.stubGlobal("MediaRecorder", undefined);
    const { result } = renderHook(() => useAudioRecording("7", vi.fn()));
    await act(async () => result.current.handleStartRecording());
    expect(toastError).toHaveBeenCalledTimes(1);
    expect(result.current.recording).toBe(false);
  });
});

describe("formato da gravação", () => {
  it("escolhe o primeiro formato suportado (webm/Opus no Chrome, mp4 no Safari)", () => {
    expect(pickRecordingType((t) => t === "audio/webm;codecs=opus")).toBe("audio/webm;codecs=opus");
    expect(pickRecordingType((t) => t.startsWith("audio/mp4"))).toBe("audio/mp4;codecs=mp4a.40.2");
    expect(pickRecordingType(() => false)).toBe("");
  });
  it("extensão pelo mimetype", () => {
    expect(extensionFor("audio/webm;codecs=opus")).toBe("webm");
    expect(extensionFor("audio/ogg; codecs=opus")).toBe("ogg");
    expect(extensionFor("audio/mp4")).toBe("m4a");
    expect(extensionFor("")).toBe("audio");
  });
});
