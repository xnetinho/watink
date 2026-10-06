import { describe, expect, it } from "vitest";
import { hasMediaBubble } from "../MessageBubble";
import type { Message } from "../../types";

const msg = (over: Partial<Message>): Message => ({ id: "1", body: "x", fromMe: false, createdAt: new Date().toISOString(), ...over });

// Esta decisão é o que faz a mensagem de chamada APARECER no ticket: sem ela o
// MessageBubble nunca chama o MessageMedia, e a chamada some do histórico.
describe("hasMediaBubble", () => {
  it("mensagem de chamada tem bolha própria mesmo sem mediaUrl", () => {
    expect(hasMediaBubble(msg({ mediaType: "call" }))).toBe(true);
  });
  it("texto puro não tem bolha de mídia", () => {
    expect(hasMediaBubble(msg({}))).toBe(false);
    expect(hasMediaBubble(msg({ mediaType: "chat" }))).toBe(false);
  });
  it("os tipos de mídia que já existiam continuam valendo", () => {
    for (const t of ["image", "video", "audio", "document", "sticker", "location", "vcard", "carousel"]) {
      expect(hasMediaBubble(msg({ mediaType: t })), t).toBe(true);
    }
  });
  it("qualquer mensagem com mediaUrl tem bolha", () => {
    expect(hasMediaBubble(msg({ mediaUrl: "http://x/y.png" }))).toBe(true);
  });
});
