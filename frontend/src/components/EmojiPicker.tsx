import React, { useMemo, useState } from "react";
import { Search } from "lucide-react";
import catalog from "../lib/emoji/catalog.json";
import { emojiImageUrl } from "../lib/emoji/emoji";
import { i18n } from "../translate/i18n";
import { cn } from "../lib/utils";

// i18n.t() deste projeto pode devolver null; a tela só quer string.
const tr = (key: string): string => String(i18n.t(key) ?? "");

type Entry = { c: string; n: string; k: string[] };
type Category = { id: string; emojis: Entry[] };
const CATEGORIES = catalog as Category[];

// Ícone de cada categoria: o primeiro emoji dela, desenhado pelo mesmo conjunto.
const tabIcon = (c: Category) => c.emojis[0]?.c ?? "🙂";

interface Props {
  onSelect: (emoji: { native: string }) => void;
}

/**
 * Seletor de emojis com as imagens do mesmo conjunto que desenha as mensagens (Apple, como o WhatsApp Web): o que se
 * escolhe aqui é idêntico ao que aparece na bolha. Mostra uma categoria por vez (busca varre todas), então nunca monta
 * os ~1.900 emojis de uma vez.
 */
export const EmojiPicker: React.FC<Props> = ({ onSelect }) => {
  const [active, setActive] = useState(CATEGORIES[0]?.id);
  const [query, setQuery] = useState("");

  const visible = useMemo<Entry[]>(() => {
    const q = query.trim().toLowerCase().replace(/\s+/g, "_");
    if (q) {
      return CATEGORIES.flatMap((c) => c.emojis).filter((e) => e.n.includes(q) || e.k.some((k) => k.includes(q))).slice(0, 200);
    }
    return CATEGORIES.find((c) => c.id === active)?.emojis ?? [];
  }, [active, query]);

  return (
    <div
      className="flex h-[360px] w-[352px] flex-col overflow-hidden rounded-xl border border-[hsl(var(--border-default))] bg-[hsl(var(--bg-surface))] shadow-lg"
      data-testid="emoji-picker"
    >
      <div className="flex items-center gap-1.5 border-b border-[hsl(var(--border-default))] px-2 py-1.5">
        <Search className="h-4 w-4 shrink-0 text-[hsl(var(--text-muted))]" />
        <input
          autoFocus
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={tr("messagesInput.emojiPicker.search")}
          aria-label={tr("messagesInput.emojiPicker.search")}
          className="w-full bg-transparent text-sm outline-none placeholder:text-[hsl(var(--text-muted))]"
        />
      </div>

      {!query && (
        <div className="flex items-center justify-between border-b border-[hsl(var(--border-default))] px-1 py-1" role="tablist">
          {CATEGORIES.map((c) => (
            <button
              key={c.id}
              type="button"
              role="tab"
              aria-selected={active === c.id}
              aria-label={tr(`messagesInput.emojiPicker.categories.${c.id}`)}
              title={tr(`messagesInput.emojiPicker.categories.${c.id}`)}
              onClick={() => setActive(c.id)}
              className={cn("rounded-md p-1 transition-colors hover:bg-[var(--border-subtle)]", active === c.id && "bg-[var(--border-subtle)]")}
            >
              <EmojiImg emoji={tabIcon(c)} size={20} />
            </button>
          ))}
        </div>
      )}

      <div className="flex-1 overflow-y-auto p-1.5" data-testid="emoji-grid">
        {!query && (
          <p className="px-1 pb-1 text-[11px] font-medium uppercase tracking-wide text-[hsl(var(--text-muted))]">
            {tr(`messagesInput.emojiPicker.categories.${active}`)}
          </p>
        )}
        {visible.length === 0 ? (
          <p className="p-4 text-center text-sm text-[hsl(var(--text-muted))]">{tr("messagesInput.emojiPicker.none")}</p>
        ) : (
          <div className="grid grid-cols-8 gap-0.5">
            {visible.map((e) => (
              <button
                key={e.c}
                type="button"
                title={e.n.replace(/_/g, " ")}
                aria-label={e.n.replace(/_/g, " ")}
                onClick={() => onSelect({ native: e.c })}
                className="flex h-9 w-9 items-center justify-center rounded-md transition-colors hover:bg-[var(--border-subtle)]"
              >
                <EmojiImg emoji={e.c} size={26} />
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

const EmojiImg: React.FC<{ emoji: string; size: number }> = ({ emoji, size }) => {
  const url = emojiImageUrl(emoji);
  if (!url) return <span style={{ fontSize: size * 0.85 }}>{emoji}</span>;
  return <img src={url} alt={emoji} width={size} height={size} loading="lazy" draggable={false} />;
};

export default EmojiPicker;
