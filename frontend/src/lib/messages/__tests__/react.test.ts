import { describe, expect, it } from "vitest";
import { canReact } from "../react";

describe("canReact", () => {
  it("texto, mídia e figurinha aceitam reação", () => {
    for (const mediaType of [undefined, "", "chat", "image", "audio", "sticker", "poll"]) {
      expect(canReact({ id: "M1", mediaType })).toBe(true);
    }
  });
  it("registro de chamada e mensagem apagada não aceitam", () => {
    expect(canReact({ id: "call:ABC", mediaType: "call" })).toBe(false);
    expect(canReact({ id: "M1", isDeleted: true })).toBe(false);
    expect(canReact({})).toBe(false);
  });
});
