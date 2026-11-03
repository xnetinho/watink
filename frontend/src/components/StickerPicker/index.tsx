/* @jsxImportSource react */
import React, { useCallback, useEffect, useState } from "react";
import { Loader2, Trash2 } from "lucide-react";
import api from "../../services/api";
import toastError from "../../errors/toastError";
import { getBackendUrl } from "../../helpers/urlUtils";
import { i18n } from "../../translate/i18n";
import { cn } from "../../lib/utils";

const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

export interface LibrarySticker {
  id: number;
  mediaUrl: string;
  animated: boolean;
}

interface StickerPickerProps {
  /** Ticket aberto: a figurinha escolhida é enviada nele. */
  ticketId: string | undefined;
  /** Chamado depois de enviar (ou ao remover o último), para o compositor fechar o painel. */
  onSent: () => void;
}

/**
 * Biblioteca de figurinhas da EMPRESA (todos os atendentes e conexões veem as mesmas). Um clique envia a figurinha no ticket
 * aberto; a lixeira tira da biblioteca. As figurinhas entram por "Salvar figurinha" no menu de uma mensagem.
 */
const StickerPicker: React.FC<StickerPickerProps> = ({ ticketId, onSent }) => {
  const [items, setItems] = useState<LibrarySticker[] | null>(null);
  const [sendingId, setSendingId] = useState<number | null>(null);

  const load = useCallback(() => {
    api
      .get<LibrarySticker[]>("/stickers")
      .then(({ data }) => setItems(Array.isArray(data) ? data : []))
      .catch((err) => {
        setItems([]);
        toastError(err);
      });
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const send = (item: LibrarySticker) => {
    if (!ticketId || sendingId !== null) return;
    setSendingId(item.id);
    api
      .post(`/messages/${ticketId}`, { stickerId: item.id })
      .then(onSent)
      .catch(toastError)
      .finally(() => setSendingId(null));
  };

  const remove = (e: React.MouseEvent, item: LibrarySticker) => {
    e.stopPropagation();
    api
      .delete(`/stickers/${item.id}`)
      .then(() => setItems((prev) => (prev ?? []).filter((s) => s.id !== item.id)))
      .catch(toastError);
  };

  return (
    <div
      className="w-[340px] max-h-[320px] overflow-y-auto rounded-lg border border-[var(--border-subtle)] bg-[hsl(var(--bg-surface))] p-2 shadow-lg"
      data-testid="sticker-picker"
    >
      {items === null ? (
        <div className="flex items-center justify-center gap-2 p-6 text-sm text-[hsl(var(--text-muted))]">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          {tr("stickerPicker.loading")}
        </div>
      ) : items.length === 0 ? (
        <p className="p-6 text-center text-sm text-[hsl(var(--text-muted))]" data-testid="sticker-picker-empty">
          {tr("stickerPicker.empty")}
        </p>
      ) : (
        <div className="grid grid-cols-4 gap-2">
          {items.map((s) => (
            <div key={s.id} className="group relative">
              <button
                type="button"
                onClick={() => send(s)}
                disabled={sendingId !== null}
                aria-label={tr("stickerPicker.send")}
                title={tr("stickerPicker.send")}
                className={cn(
                  "flex h-[72px] w-[72px] items-center justify-center rounded-md hover:bg-[var(--border-subtle)] disabled:opacity-50",
                  sendingId === s.id && "animate-pulse"
                )}
                data-testid="sticker-picker-item"
              >
                <img src={getBackendUrl(s.mediaUrl) ?? ""} alt="" loading="lazy" className="h-16 w-16 object-contain" />
              </button>
              <button
                type="button"
                onClick={(e) => remove(e, s)}
                aria-label={tr("stickerPicker.remove")}
                title={tr("stickerPicker.remove")}
                className="absolute right-0 top-0 hidden rounded-full bg-[hsl(var(--bg-surface))] p-1 text-[hsl(var(--text-muted))] shadow group-hover:flex hover:text-red-500"
                data-testid="sticker-picker-remove"
              >
                <Trash2 className="h-3 w-3" aria-hidden="true" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default StickerPicker;
