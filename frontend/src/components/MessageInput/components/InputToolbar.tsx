import React, { Suspense, lazy } from "react";
import { Paperclip, Smile, MoreVertical, Sticker } from "lucide-react";
import { Switch } from "../../ui/switch";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../../ui/dropdown-menu";
import { i18n } from "../../../translate/i18n";


const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
// Seletor próprio, com as imagens do mesmo conjunto que desenha as mensagens (carregado só quando abre).
const EmojiPicker = lazy(() => import("../../EmojiPicker"));
// Biblioteca de figurinhas da empresa (também carregada só quando abre).
const StickerPicker = lazy(() => import("../../StickerPicker"));

interface InputToolbarProps {
  loading: boolean;
  recording: boolean;
  ticketStatus: string;
  showEmoji: boolean;
  signMessage: boolean;
  anchorElMenu: boolean;
  onToggleEmoji: () => void;
  onEmojiClose: () => void;
  onEmojiSelect: (e: { native: string }) => void;
  onMediaChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onSignChange: (checked: boolean) => void;
  onMenuOpenChange: (open: boolean) => void;
  /** Biblioteca de figurinhas: ausente = o canal não tem figurinha (o botão não aparece). */
  stickers?: { open: boolean; ticketId: string | undefined; onToggle: () => void; onClose: () => void };
}

const InputToolbar: React.FC<InputToolbarProps> = ({
  loading,
  recording,
  ticketStatus,
  showEmoji,
  signMessage,
  anchorElMenu,
  onToggleEmoji,
  onEmojiClose,
  onEmojiSelect,
  onMediaChange,
  onSignChange,
  onMenuOpenChange,
  stickers,
}) => {
  const disabled = loading || recording || ticketStatus !== "open";

  return (
    <>
      {/* Desktop toolbar */}
      <div className="hidden sm:flex items-center">
        <button
          type="button"
          className="p-2.5 rounded-full hover:bg-[var(--border-subtle)] transition-colors disabled:opacity-50"
          disabled={disabled}
          onClick={onToggleEmoji}
          aria-label="Emoji"
        >
          <Smile className="h-6 w-6 text-gray-500" />
        </button>

        {showEmoji && (
          <div className="absolute bottom-16 z-50">
            <div onMouseLeave={onEmojiClose}>
              <Suspense fallback={<div className="text-xs p-2">{String(i18n.t("messagesInput.emojiPicker.loading") ?? "")}</div>}>
                <EmojiPicker onSelect={onEmojiSelect} />
              </Suspense>
            </div>
          </div>
        )}

        {stickers && (
          <>
            <button
              type="button"
              className="p-2.5 rounded-full hover:bg-[var(--border-subtle)] transition-colors disabled:opacity-50"
              disabled={disabled}
              onClick={stickers.onToggle}
              aria-label={tr("messagesInput.stickers")}
              aria-pressed={stickers.open}
              title={tr("messagesInput.stickers")}
              data-testid="sticker-picker-button"
            >
              <Sticker className="h-6 w-6 text-gray-500" />
            </button>
            {stickers.open && (
              <div className="absolute bottom-16 z-50">
                <div onMouseLeave={stickers.onClose}>
                  <Suspense fallback={<div className="text-xs p-2">{tr("stickerPicker.loading")}</div>}>
                    <StickerPicker ticketId={stickers.ticketId} onSent={stickers.onClose} />
                  </Suspense>
                </div>
              </div>
            )}
          </>
        )}

        <input
          multiple
          type="file"
          id="upload-button"
          disabled={disabled}
          className="hidden"
          onChange={onMediaChange}
        />
        <label htmlFor="upload-button">
          <button
            type="button"
            className="p-2.5 rounded-full hover:bg-[var(--border-subtle)] transition-colors disabled:opacity-50 pointer-events-none"
            disabled={disabled}
            aria-label="Anexar arquivo"
            tabIndex={-1}
          >
            <Paperclip className="h-6 w-6 text-gray-500" />
          </button>
        </label>

        <label className="flex items-center gap-1.5 mr-1.5 text-gray-500 text-sm select-none cursor-pointer">
          <span className="text-xs">{i18n.t("messagesInput.signMessage")}</span>
          <Switch
            checked={Boolean(signMessage)}
            onCheckedChange={onSignChange}
            aria-label={i18n.t("messagesInput.signMessage") as string}
          />
        </label>
      </div>

      {/* Mobile toolbar */}
      <div className="flex sm:hidden">
        <DropdownMenu open={anchorElMenu} onOpenChange={onMenuOpenChange}>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              className="p-2 rounded-full hover:bg-[var(--border-subtle)] transition-colors"
              aria-label={tr("uiText.common.moreOptions")}
            >
              <MoreVertical className="h-5 w-5" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start">
            <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
              <button
                type="button"
                className="p-1 rounded-full hover:bg-[var(--border-subtle)] transition-colors disabled:opacity-50"
                disabled={disabled}
                onClick={() => {
                  onToggleEmoji();
                  onMenuOpenChange(false);
                }}
              >
                <Smile className="h-6 w-6 text-gray-500" />
              </button>
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
              <input
                multiple
                type="file"
                id="upload-button-mobile"
                disabled={disabled}
                className="hidden"
                onChange={(e) => {
                  onMediaChange(e);
                  onMenuOpenChange(false);
                }}
              />
              <label htmlFor="upload-button-mobile" className="cursor-pointer">
                <Paperclip className="h-6 w-6 text-gray-500" />
              </label>
            </DropdownMenuItem>
            <DropdownMenuItem onSelect={(e) => e.preventDefault()}>
              <label className="flex items-center gap-1.5 text-sm text-gray-500 cursor-pointer">
                <span className="text-xs">{i18n.t("messagesInput.signMessage")}</span>
                <Switch
                  checked={Boolean(signMessage)}
                  onCheckedChange={(checked) => {
                    onSignChange(checked);
                    onMenuOpenChange(false);
                  }}
                />
              </label>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </>
  );
};

export default InputToolbar;
