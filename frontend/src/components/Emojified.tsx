import React from "react";
import { splitEmojis } from "../lib/emoji/emoji";
import { cn } from "../lib/utils";

interface Props {
  children?: string | null;
  className?: string;
}

/**
 * Texto com os emojis desenhados como imagem (conjunto Apple, como o WhatsApp Web). Use no lugar de imprimir a string
 * direto sempre que o texto puder conter emoji (nomes, reações, prévias, mensagens). O texto original fica em `alt`, então
 * copiar, selecionar e leitores de tela continuam funcionando.
 */
export const Emojified: React.FC<Props> = ({ children, className }) => {
  const text = children ?? "";
  const parts = React.useMemo(() => splitEmojis(text), [text]);
  if (!text) return null;
  return (
    <>
      {parts.map((p, i) =>
        p.type === "text" ? (
          <React.Fragment key={i}>{p.value}</React.Fragment>
        ) : (
          <img
            key={i}
            src={p.url}
            alt={p.value}
            draggable={false}
            loading="lazy"
            data-testid="emoji-img"
            className={cn("inline-block h-[1.25em] w-[1.25em] -translate-y-px align-middle", className)}
          />
        ),
      )}
    </>
  );
};

export default Emojified;
