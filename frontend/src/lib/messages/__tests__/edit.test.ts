import { describe, expect, it } from "vitest";
import { canEdit, EDIT_WINDOW_MS } from "../edit";

const now = Date.parse("2026-10-29T12:00:00Z");
const base = { fromMe: true, ack: 2, createdAt: new Date(now - 5 * 60_000).toISOString() };

describe("canEdit (quando a ação Editar aparece)", () => {
  it("mensagem nossa de texto, entregue, dentro do prazo", () => {
    expect(canEdit(base, "whatsapp", now)).toBe(true);
    expect(canEdit(base, undefined, now)).toBe(true);
  });

  it("ack 1 a 4 aparecem; 0 (não saiu) e 5 (falhou: o caminho é reenviar) não", () => {
    for (const ack of [1, 2, 3, 4]) expect(canEdit({ ...base, ack }, "whatsapp", now)).toBe(true);
    for (const ack of [0, 5, undefined]) expect(canEdit({ ...base, ack }, "whatsapp", now)).toBe(false);
  });

  it("o prazo é de 20 min: no limite ainda aparece, 1 ms depois não", () => {
    const at = (ms: number) => ({ ...base, createdAt: new Date(now - ms).toISOString() });
    expect(canEdit(at(EDIT_WINDOW_MS), "whatsapp", now)).toBe(true);
    expect(canEdit(at(EDIT_WINDOW_MS + 1), "whatsapp", now)).toBe(false);
  });

  it("só mensagem NOSSA e não apagada", () => {
    expect(canEdit({ ...base, fromMe: false }, "whatsapp", now)).toBe(false);
    expect(canEdit({ ...base, isDeleted: true }, "whatsapp", now)).toBe(false);
  });

  it("legenda de imagem, vídeo e documento sim; áudio, figurinha, enquete e outros não", () => {
    for (const mediaType of ["image", "video", "document", "chat", ""]) expect(canEdit({ ...base, mediaType }, "whatsapp", now)).toBe(true);
    for (const mediaType of ["audio", "sticker", "poll", "location", "vcard", "call"]) expect(canEdit({ ...base, mediaType }, "whatsapp", now)).toBe(false);
  });

  it("outros canais (Telegram etc.) não editam por aqui", () => {
    expect(canEdit(base, "telegram", now)).toBe(false);
  });

  it("sem data válida não arrisca mostrar", () => {
    expect(canEdit({ ...base, createdAt: undefined }, "whatsapp", now)).toBe(false);
    expect(canEdit({ ...base, createdAt: "lixo" }, "whatsapp", now)).toBe(false);
  });
});
