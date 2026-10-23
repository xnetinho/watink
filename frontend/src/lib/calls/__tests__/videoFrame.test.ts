import { describe, expect, it } from "vitest";
import { decodeVideoFrame, encodeVideoFrame, isVideoFrame } from "../videoFrame";

describe("videoFrame", () => {
  it("round-trip preserva quadro-chave, timestamp e bytes", () => {
    const data = new Uint8Array([0, 0, 0, 1, 0x65, 9, 8, 7]);
    for (const keyframe of [true, false]) {
      const msg = encodeVideoFrame({ keyframe, rotation: 0, ts90k: 0xdeadbeef, data });
      expect(isVideoFrame(msg)).toBe(true);
      const f = decodeVideoFrame(msg);
      expect(f?.keyframe).toBe(keyframe);
      expect(f?.ts90k).toBe(0xdeadbeef);
      expect(Array.from(f?.data ?? [])).toEqual(Array.from(data));
    }
  });

  it("bytes idênticos aos do engine (vetor fixo, para os dois lados nunca divergirem)", () => {
    const msg = encodeVideoFrame({ keyframe: true, rotation: 0, ts90k: 0x01020304, data: new Uint8Array([0xaa, 0xbb]) });
    expect(Array.from(msg)).toEqual([0xff, 0x56, 0x44, 0x01, 0x01, 0x01, 0x02, 0x03, 0x04, 0xaa, 0xbb]);
  });

  it("PCM nunca é confundido com vídeo", () => {
    for (const pcm of [new Uint8Array(640), new Uint8Array(640).fill(0xff), new Uint8Array([0xff]), new Uint8Array([0xff, 0x56, 0x44]), new Uint8Array([0xff, 0x56, 0x44, 0x01])]) {
      expect(isVideoFrame(pcm)).toBe(false);
      expect(decodeVideoFrame(pcm)).toBeNull();
    }
  });

  it("só o prefixo, sem corpo, não é um quadro", () => {
    const msg = encodeVideoFrame({ keyframe: true, rotation: 0, ts90k: 1, data: new Uint8Array([1]) });
    expect(isVideoFrame(msg.subarray(0, 9))).toBe(false);
  });

  it("a rotação (0..3) viaja ao lado do quadro-chave, com os MESMOS bytes do engine", () => {
    // vetor idêntico ao TestVideoFrameWireVector do engine: ts=0x01020304, keyframe, rotação 3
    const msg = encodeVideoFrame({ keyframe: true, rotation: 3, ts90k: 0x01020304, data: new Uint8Array([0xaa, 0xbb]) });
    expect(Array.from(msg)).toEqual([0xff, 0x56, 0x44, 0x01, 0x07, 0x01, 0x02, 0x03, 0x04, 0xaa, 0xbb]);
    const f = decodeVideoFrame(msg);
    expect(f?.rotation).toBe(3);
    expect(f?.keyframe).toBe(true);
    for (const rotation of [0, 1, 2, 3] as const) {
      for (const keyframe of [true, false]) {
        const r = decodeVideoFrame(encodeVideoFrame({ keyframe, rotation, ts90k: 5, data: new Uint8Array([1]) }));
        expect([r?.keyframe, r?.rotation]).toEqual([keyframe, rotation]);
      }
    }
  });

  it("bits reservados de flags não vazam para a rotação nem para o quadro-chave", () => {
    const msg = encodeVideoFrame({ keyframe: false, rotation: 0, ts90k: 1, data: new Uint8Array([1]) });
    msg[4] |= 0xf0;
    const f = decodeVideoFrame(msg);
    expect(f?.keyframe).toBe(false);
    expect(f?.rotation).toBe(0);
  });
});

