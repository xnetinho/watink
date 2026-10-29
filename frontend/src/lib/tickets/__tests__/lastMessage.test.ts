import { describe, expect, it } from "vitest";
import { lastMessagePreview } from "../lastMessage";
import { i18n } from "../../../translate/i18n";

const inLang = (lng: string, ticket: { lastMessage?: string; lastMessageType?: string }) => {
  const prev = i18n.language;
  void i18n.changeLanguage(lng);
  const out = lastMessagePreview(ticket);
  void i18n.changeLanguage(prev);
  return out;
};

describe("prévia da última mensagem (marcador de tipo traduzido pela tela)", () => {
  it("texto real vence", () => {
    expect(lastMessagePreview({ lastMessage: "oi, tudo bem?" })).toBe("oi, tudo bem?");
    expect(lastMessagePreview({ lastMessage: "legenda", lastMessageType: "image" })).toBe("legenda");
  });
  it("marcador vira o rótulo no idioma de quem olha", () => {
    expect(inLang("pt", { lastMessageType: "image" })).toBe("📷 Foto");
    expect(inLang("en", { lastMessageType: "image" })).toBe("📷 Photo");
    expect(inLang("es", { lastMessageType: "poll" })).toBe("📊 Encuesta");
  });
  it("todos os tipos que o backend marca têm tradução nos 3 idiomas", () => {
    const kinds = ["image", "video", "audio", "document", "sticker", "location", "vcard", "poll", "event", "pix", "view_once", "unsupported", "call"];
    for (const lng of ["pt", "en", "es"]) {
      for (const k of kinds) {
        const t = inLang(lng, { lastMessageType: k });
        expect(t, `${lng}.${k}`).not.toContain("tickets.lastMessageKind");
        expect(t.length, `${lng}.${k}`).toBeGreaterThan(1);
      }
    }
  });
  it("tipo que ainda não existe (canal novo) cai no genérico, nunca na chave crua", () => {
    const t = inLang("pt", { lastMessageType: "tipo_do_futuro" });
    expect(t).toBe("📎 Mensagem");
    expect(t).not.toContain("tickets.");
  });
  it("sem nada devolve vazio (a lista mostra 'sem mensagens')", () => {
    expect(lastMessagePreview({})).toBe("");
  });
});
