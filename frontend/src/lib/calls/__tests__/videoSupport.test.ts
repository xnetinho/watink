import { describe, expect, it } from "vitest";
import { detectVideoSupport, VIDEO_DECODER_CONFIG } from "../videoSupport";

const fn = () => undefined;

describe("detectVideoSupport", () => {
  it("Chromium completo: recebe e envia", () => {
    expect(detectVideoSupport({ VideoDecoder: fn, VideoEncoder: fn, MediaStreamTrackProcessor: fn })).toEqual({ canReceive: true, canSend: true });
  });

  it("decodifica mas não tem MediaStreamTrackProcessor (ex.: Firefox/Safari recentes): só recebe", () => {
    expect(detectVideoSupport({ VideoDecoder: fn, VideoEncoder: fn })).toEqual({ canReceive: true, canSend: false });
  });

  it("sem WebCodecs: nem recebe nem envia", () => {
    expect(detectVideoSupport({})).toEqual({ canReceive: false, canSend: false });
  });

  it("sem window (SSR/teste): nada", () => {
    expect(detectVideoSupport(undefined)).toEqual({ canReceive: false, canSend: false });
  });

  it("não se deixa enganar por um valor que não é função", () => {
    expect(detectVideoSupport({ VideoDecoder: true as unknown, VideoEncoder: fn, MediaStreamTrackProcessor: fn })).toEqual({
      canReceive: false,
      canSend: false,
    });
  });

  it("o decoder pede H.264 Constrained Baseline 3.1 de baixa latência", () => {
    expect(VIDEO_DECODER_CONFIG.codec).toBe("avc1.42E01F");
    expect(VIDEO_DECODER_CONFIG.optimizeForLatency).toBe(true);
  });
});
