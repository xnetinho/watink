import index from "./apple-index.json";

/**
 * Emojis desenhados com o conjunto de imagens da Apple (o mais próximo do WhatsApp Web, que se inspirou nele), em vez da
 * fonte do sistema: o mesmo emoji fica igual em Windows, macOS, Android e Linux, e o que se escolhe no seletor é o que
 * aparece na bolha. As imagens ficam no próprio app (public/emoji/apple), sem CDN, e carregam sob demanda.
 *
 * Fonte: emoji-datasource-apple 16.0.0 (MIT para o pacote; as imagens são arte da Apple, ver docs/dev/emoji.md).
 */

const BASE = "/emoji/apple/";

const files = new Set<string>(index.files);
const alias = index.alias as Record<string, string>;

const FE0F = /\ufe0f/g;

/** Sequência de code points em hexadecimal minúsculo, separados por "-" (ex.: "2764-fe0f", "0031-fe0f-20e3"). */
export const toUnified = (s: string): string =>
  Array.from(s)
    // O conjunto nomeia com no mínimo 4 dígitos (0031 e não 31; 00a9 e não a9); acima de 0xFFFF não há preenchimento.
    .map((c) => c.codePointAt(0)!.toString(16).padStart(4, "0"))
    .join("-");

/** URL da imagem de um emoji, ou null se o conjunto não tem esse emoji (aí a fonte do sistema o desenha). */
export function emojiImageUrl(emoji: string): string | null {
  const u = toUnified(emoji);
  const candidates = [u, toUnified(emoji.replace(FE0F, ""))];
  for (const c of candidates) {
    const file = alias[c] ?? `${c}.webp`;
    if (files.has(file)) return BASE + file;
  }
  return null;
}

/**
 * Casa um emoji completo: bandeiras (2 indicadores regionais), sequências com ZWJ (família, profissões), tom de pele,
 * teclas (1️⃣) e o emoji simples com ou sem seletor de variação. `\p{Extended_Pictographic}` sozinho cortaria as
 * sequências no meio, por isso o padrão consome a sequência inteira.
 */
const EMOJI_RE = new RegExp(
  [
    "\\p{Regional_Indicator}{2}",
    "[0-9#*]\\ufe0f?\\u20e3",
    "(?:\\p{Extended_Pictographic}|\\p{Emoji_Presentation})(?:\\ufe0f|\\p{Emoji_Modifier})?(?:\\u200d(?:\\p{Extended_Pictographic}|\\p{Emoji_Presentation})(?:\\ufe0f|\\p{Emoji_Modifier})?)*",
  ].join("|"),
  "gu",
);

export type EmojiPart = { type: "text"; value: string } | { type: "emoji"; value: string; url: string };

/** Quebra um texto em trechos de texto e emojis com imagem. Emoji sem imagem no conjunto continua como texto. */
export function splitEmojis(text: string): EmojiPart[] {
  if (!text) return [];
  const parts: EmojiPart[] = [];
  let last = 0;
  const push = (value: string) => {
    if (!value) return;
    const prev = parts[parts.length - 1];
    if (prev && prev.type === "text") prev.value += value;
    else parts.push({ type: "text", value });
  };
  for (const m of text.matchAll(EMOJI_RE)) {
    const at = m.index ?? 0;
    push(text.slice(last, at));
    const url = emojiImageUrl(m[0]);
    if (url) parts.push({ type: "emoji", value: m[0], url });
    else push(m[0]);
    last = at + m[0].length;
  }
  push(text.slice(last));
  return parts;
}
