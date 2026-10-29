import { describe, expect, it } from "vitest";
import { canForward } from "../forward";

describe("canForward", () => {
  it("texto e mídia baixada podem", () => {
    expect(canForward({ id: "1", mediaType: "chat" })).toBe(true);
    expect(canForward({ id: "1" })).toBe(true);
    expect(canForward({ id: "1", mediaType: "image", mediaUrl: "/public/media/a.png" })).toBe(true);
    expect(canForward({ id: "1", mediaType: "sticker", mediaUrl: "/public/media/a.webp" })).toBe(true);
  });
  it("mídia pendente, apagada, sem id e cartões não podem", () => {
    expect(canForward({ id: "1", mediaType: "image", mediaUrl: "" })).toBe(false);
    expect(canForward({ id: "1", isDeleted: true })).toBe(false);
    expect(canForward({ mediaType: "chat" })).toBe(false);
    for (const t of ["poll", "location", "vcard", "pix", "event", "call", "view_once", "unsupported"]) {
      expect(canForward({ id: "1", mediaType: t, mediaUrl: "/x" }), t).toBe(false);
    }
  });
});
