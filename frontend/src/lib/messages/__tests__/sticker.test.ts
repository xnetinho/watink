import { describe, expect, it } from "vitest";
import { canSaveSticker } from "../sticker";

describe("canSaveSticker", () => {
  const ok = { id: "S1", mediaType: "sticker", mediaUrl: "/public/media/a.webp" };

  it("figurinha com arquivo guardado pode ir para a biblioteca, recebida ou enviada", () => {
    expect(canSaveSticker(ok)).toBe(true);
    expect(canSaveSticker(ok, "whatsapp")).toBe(true);
  });

  it("só no WhatsApp: outros canais têm outro mecanismo de figurinha", () => {
    expect(canSaveSticker(ok, "telegram")).toBe(false);
  });

  it("sem arquivo guardado, apagada ou sem id, não", () => {
    expect(canSaveSticker({ ...ok, mediaUrl: "" })).toBe(false);
    expect(canSaveSticker({ ...ok, mediaUrl: undefined })).toBe(false);
    expect(canSaveSticker({ ...ok, isDeleted: true })).toBe(false);
    expect(canSaveSticker({ mediaType: "sticker", mediaUrl: "/x.webp" })).toBe(false);
  });

  it("outros tipos de mídia não são figurinha", () => {
    for (const mediaType of ["image", "video", "audio", "document", "chat", ""]) {
      expect(canSaveSticker({ ...ok, mediaType })).toBe(false);
    }
  });
});
