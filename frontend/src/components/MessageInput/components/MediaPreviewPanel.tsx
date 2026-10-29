import React, { useEffect, useMemo, useState } from "react";
import { FileText, Film, Loader2, Music, Plus, Send, X } from "lucide-react";
import { i18n } from "../../../translate/i18n";
import { MediaItem } from "../hooks/useMessageInput";
import { AudioKindPicker } from "../../AudioKindPicker";
import { cn } from "../../../lib/utils";

const tr = (key: string, opts: Record<string, string | number> = {}): string => String(i18n.t(key, opts) ?? "");

interface MediaPreviewPanelProps {
  medias: MediaItem[];
  loading: boolean;
  onRemoveMedia: (index: number) => void;
  onCaptionChange: (index: number, value: string) => void;
  onAudioKindChange: (index: number, kind: "voice" | "file") => void;
  onAddMore: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onCancelAll: () => void;
  onUpload: () => void;
}

/** "1,2 MB" / "340 KB": o tamanho que o WhatsApp Web mostra abaixo do nome do arquivo. */
export function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1).replace(".", ",")} MB`;
}

type Kind = "image" | "video" | "audio" | "file";
const kindOf = (f: File): Kind => {
  if (f.type.startsWith("image/")) return "image";
  if (f.type.startsWith("video/")) return "video";
  if (f.type.startsWith("audio/")) return "audio";
  return "file";
};

/** Uma URL de pré-visualização por arquivo, criada uma vez e liberada ao sair (a tela antiga criava uma a cada renderização). */
function usePreviewUrls(medias: MediaItem[]): string[] {
  const files = medias.map((m) => m.file);
  const urls = useMemo(
    () => files.map((f) => (kindOf(f) === "image" || kindOf(f) === "video" ? URL.createObjectURL(f) : "")),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [files.length, ...files]
  );
  useEffect(() => () => urls.forEach((u) => u && URL.revokeObjectURL(u)), [urls]);
  return urls;
}

/** O que aparece grande (ou pequeno, na miniatura) para cada tipo de arquivo. */
const Thumb: React.FC<{ file: File; url: string; big?: boolean }> = ({ file, url, big = false }) => {
  const kind = kindOf(file);
  if (kind === "image") return <img src={url} alt={file.name} className={cn("object-contain", big ? "max-h-full max-w-full" : "h-full w-full object-cover")} />;
  if (kind === "video") {
    return big ? (
      <video src={url} controls className="max-h-full max-w-full" />
    ) : (
      <div className="flex h-full w-full items-center justify-center bg-black/40">
        <Film className="h-5 w-5 text-white" aria-hidden="true" />
      </div>
    );
  }
  const Icon = kind === "audio" ? Music : FileText;
  return (
    <div className={cn("flex flex-col items-center justify-center gap-2 text-[hsl(var(--text-muted))]", big ? "h-full" : "h-full w-full")}>
      <Icon className={big ? "h-24 w-24" : "h-6 w-6"} aria-hidden="true" />
      {big && (
        <>
          <span className="max-w-[80%] truncate text-base font-medium text-[hsl(var(--foreground))]">{file.name}</span>
          <span className="text-sm">{formatSize(file.size)}</span>
        </>
      )}
    </div>
  );
};

/**
 * Pré-visualização antes de enviar, no estilo do WhatsApp Web: o arquivo selecionado grande no centro, a legenda num campo
 * arredondado logo abaixo, a fileira de miniaturas (com "+" para adicionar mais) e o botão redondo de enviar.
 */
const MediaPreviewPanel: React.FC<MediaPreviewPanelProps> = ({
  medias,
  loading,
  onRemoveMedia,
  onCaptionChange,
  onAudioKindChange,
  onAddMore,
  onCancelAll,
  onUpload,
}) => {
  const [selected, setSelected] = useState(0);
  const urls = usePreviewUrls(medias);

  // Remover o último, ou mais arquivos que o selecionado, não pode deixar o índice fora da lista.
  const index = Math.min(selected, medias.length - 1);
  useEffect(() => {
    if (selected !== index) setSelected(index);
  }, [selected, index]);

  const current = medias[index];
  if (!current) return null;

  const onCaptionKey = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey && !loading) {
      e.preventDefault();
      onUpload();
    }
  };

  return (
    <div
      className="flex w-full flex-col border-t border-[var(--border-divider)] bg-[hsl(var(--bg-surface-alt))]"
      data-testid="media-preview"
    >
      <div className="flex items-center justify-between px-3 py-2">
        <button
          type="button"
          onClick={onCancelAll}
          disabled={loading}
          className="rounded-full p-1.5 text-[hsl(var(--text-muted))] transition-colors hover:bg-[var(--border-subtle)] disabled:opacity-50"
          aria-label={tr("messagesInput.preview.cancel")}
          data-testid="media-preview-cancel"
        >
          <X className="h-5 w-5" />
        </button>
        <span className="truncate px-2 text-sm font-medium" data-testid="media-preview-name">
          {current.file.name}
        </span>
        <span className="w-8" />
      </div>

      <div className="relative flex h-[38vh] min-h-[200px] items-center justify-center overflow-hidden px-4" data-testid="media-preview-main">
        {loading && <Loader2 className="absolute h-8 w-8 animate-spin text-[hsl(var(--status-success))]" aria-hidden="true" />}
        <div className={cn("flex h-full w-full items-center justify-center", loading && "opacity-40")}>
          <Thumb file={current.file} url={urls[index]} big />
        </div>
      </div>

      {current.audioKind && (
        <div className="flex justify-center px-4 pb-2">
          <AudioKindPicker value={current.audioKind} onChange={(k) => onAudioKindChange(index, k)} name={`audio-kind-${index}`} className="flex flex-col gap-1 text-xs" />
        </div>
      )}

      <div className="flex items-center gap-2 px-4 pb-2">
        <input
          type="text"
          value={current.caption}
          onChange={(e) => onCaptionChange(index, e.target.value)}
          onKeyDown={onCaptionKey}
          disabled={loading}
          placeholder={tr("messagesInput.preview.caption")}
          aria-label={tr("messagesInput.preview.caption")}
          className="h-11 min-w-0 flex-1 rounded-full border border-[var(--border-subtle)] bg-[hsl(var(--bg-surface))] px-4 text-[15px] outline-none focus:border-[hsl(var(--action-primary))]"
          data-testid="media-preview-caption"
        />
      </div>

      <div className="flex items-center gap-2 px-4 pb-3">
        <div className="flex min-w-0 flex-1 items-center gap-2 overflow-x-auto py-1" data-testid="media-preview-strip">
          {medias.map((m, i) => (
            <div key={`${m.file.name}-${i}`} className="relative shrink-0">
              <button
                type="button"
                onClick={() => setSelected(i)}
                className={cn(
                  "block h-14 w-14 overflow-hidden rounded-lg border-2 transition-colors",
                  i === index ? "border-[hsl(var(--action-primary))]" : "border-transparent opacity-70 hover:opacity-100"
                )}
                aria-label={m.file.name}
                aria-current={i === index}
                data-testid={`media-preview-thumb-${i}`}
              >
                <Thumb file={m.file} url={urls[i]} />
              </button>
              {!loading && (
                <button
                  type="button"
                  onClick={() => onRemoveMedia(i)}
                  className="absolute -right-1 -top-1 rounded-full bg-[hsl(var(--bg-surface))] p-0.5 text-[hsl(var(--text-muted))] shadow hover:text-[hsl(var(--status-error))]"
                  aria-label={tr("messagesInput.preview.remove")}
                  data-testid={`media-preview-remove-${i}`}
                >
                  <X className="h-3 w-3" />
                </button>
              )}
            </div>
          ))}

          {!loading && (
            <label
              htmlFor="add-more-media"
              className="flex h-14 w-14 shrink-0 cursor-pointer items-center justify-center rounded-lg border-2 border-dashed border-[var(--border-divider)] text-[hsl(var(--text-muted))] transition-colors hover:border-[hsl(var(--action-primary))] hover:text-[hsl(var(--action-primary))]"
              aria-label={tr("messagesInput.preview.addMore")}
              title={tr("messagesInput.preview.addMore")}
            >
              <Plus className="h-6 w-6" />
              <input multiple type="file" id="add-more-media" className="hidden" onChange={onAddMore} />
            </label>
          )}
        </div>

        <button
          type="button"
          onClick={onUpload}
          disabled={loading}
          className="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-[hsl(var(--status-success))] text-white shadow transition-opacity hover:opacity-90 disabled:opacity-50"
          aria-label={tr("messagesInput.preview.send")}
          data-testid="media-preview-send"
        >
          <Send className="h-5 w-5" />
        </button>
      </div>
    </div>
  );
};

export default MediaPreviewPanel;
