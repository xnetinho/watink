import { describe, expect, it } from "vitest";
import { emojiImageUrl, splitEmojis, toUnified } from "../emoji";
import index from "../apple-index.json";

const kinds = (s: string) => splitEmojis(s).map((p) => (p.type === "emoji" ? `E:${p.value}` : `T:${p.value}`));

describe("emoji como imagem (conjunto Apple)", () => {
  it("o emoji do pedido (😢) tem imagem", () => {
    expect(emojiImageUrl("😢")).toBe("/emoji/apple/1f622.webp");
  });

  it("converte a sequência de code points", () => {
    expect(toUnified("😢")).toBe("1f622");
    expect(toUnified("❤️")).toBe("2764-fe0f");
    expect(toUnified("1️⃣")).toBe("0031-fe0f-20e3"); // 4 dígitos mínimos, como nos nomes dos arquivos
    expect(toUnified("©")).toBe("00a9");
    expect(toUnified("👨‍👩‍👧")).toBe("1f468-200d-1f469-200d-1f467");
  });

  it("texto sem emoji volta intacto, e vazio vira lista vazia", () => {
    expect(kinds("oi, tudo bem?")).toEqual(["T:oi, tudo bem?"]);
    expect(splitEmojis("")).toEqual([]);
  });

  it("separa texto e emoji sem perder nada (ida e volta)", () => {
    const s = "Obrigado 😢 pelo apoio 🙏!";
    const joined = splitEmojis(s).map((p) => p.value).join("");
    expect(joined).toBe(s);
    expect(kinds(s)).toEqual(["T:Obrigado ", "E:😢", "T: pelo apoio ", "E:🙏", "T:!"]);
  });

  it("emojis colados são dois emojis, não um", () => {
    expect(kinds("😢😭")).toEqual(["E:😢", "E:😭"]);
  });

  it("emojis de code point baixo (©, ®, ™, ↔) também têm imagem", () => {
    for (const e of ["©️", "®️", "™️", "↔️", "☀️", "✅"]) expect(emojiImageUrl(e), e).not.toBeNull();
  });

  it("coração com e sem seletor de variação usam a mesma imagem", () => {
    expect(emojiImageUrl("❤️")).toBe(emojiImageUrl("❤"));
    expect(emojiImageUrl("❤️")).not.toBeNull();
  });

  it("sequência ZWJ (família) é UM emoji", () => {
    expect(kinds("👨‍👩‍👧")).toEqual(["E:👨‍👩‍👧"]);
  });

  it("tom de pele é UM emoji", () => {
    expect(kinds("👍🏽")).toEqual(["E:👍🏽"]);
  });

  it("bandeira (dois indicadores regionais) é UM emoji", () => {
    expect(kinds("🇧🇷")).toEqual(["E:🇧🇷"]);
  });

  it("tecla numerada é um emoji, mas dígito solto é texto", () => {
    expect(kinds("1️⃣")).toEqual(["E:1️⃣"]);
    expect(kinds("123 #1 *")).toEqual(["T:123 #1 *"]);
  });

  it("emoji que o conjunto não tem continua como texto (a fonte do sistema o desenha), sem sumir", () => {
    const rare = "\u{1FAE8}"; // shaking face (Emoji 15): pode não existir em versões antigas do conjunto
    const parts = splitEmojis(`a${rare}b`);
    expect(parts.map((p) => p.value).join("")).toBe(`a${rare}b`);
  });

  it("o índice não aponta para imagem inexistente e todo arquivo é .webp", () => {
    expect(index.files.length).toBeGreaterThan(3000);
    expect(index.files.every((f: string) => f.endsWith(".webp"))).toBe(true);
    for (const f of Object.values(index.alias as Record<string, string>)) expect(index.files).toContain(f);
  });
});
