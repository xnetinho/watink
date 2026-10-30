import React, { useCallback, useRef, useState } from "react";
import { UploadCloud } from "lucide-react";
import { i18n } from "../../translate/i18n";
import { dragHasFiles, filesFromDrop } from "../../lib/media/dropFiles";

const tr = (key: string): string => String(i18n.t(key) ?? "");

interface Props {
  /** Chamado com os arquivos soltos. */
  onFiles: (files: File[]) => void;
  /** Desligado quando não dá para enviar (ticket fechado, canal bloqueado...): o arraste passa direto. */
  disabled?: boolean;
  children: React.ReactNode;
  className?: string;
}

/**
 * Faz a conversa inteira aceitar arquivos arrastados, como o WhatsApp Web: ao arrastar sobre ela aparece uma camada
 * "Solte os arquivos aqui" e, ao soltar, os arquivos vão para a pré-visualização com legenda.
 *
 * O `dragenter`/`dragleave` dispara a cada elemento filho que o ponteiro cruza; contar as entradas evita a camada piscar.
 */
const DropZone: React.FC<Props> = ({ onFiles, disabled = false, children, className }) => {
  const [active, setActive] = useState(false);
  const depth = useRef(0);

  const reset = useCallback(() => {
    depth.current = 0;
    setActive(false);
  }, []);

  const onDragEnter = (e: React.DragEvent) => {
    if (disabled || !dragHasFiles(e.dataTransfer)) return;
    e.preventDefault();
    depth.current += 1;
    setActive(true);
  };

  const onDragOver = (e: React.DragEvent) => {
    if (disabled || !dragHasFiles(e.dataTransfer)) return;
    e.preventDefault(); // sem isto o navegador não aceita o drop
    e.dataTransfer.dropEffect = "copy";
  };

  const onDragLeave = (e: React.DragEvent) => {
    if (disabled || !dragHasFiles(e.dataTransfer)) return;
    depth.current = Math.max(0, depth.current - 1);
    if (depth.current === 0) setActive(false);
  };

  const onDrop = (e: React.DragEvent) => {
    if (disabled || !dragHasFiles(e.dataTransfer)) return;
    e.preventDefault();
    const files = filesFromDrop(e.dataTransfer);
    reset();
    if (files.length) onFiles(files);
  };

  return (
    <div
      className={className}
      onDragEnter={onDragEnter}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
      data-testid="drop-zone"
    >
      {children}
      {active && (
        <div
          className="pointer-events-none absolute inset-0 z-40 flex flex-col items-center justify-center gap-2 border-2 border-dashed border-[hsl(var(--action-primary))] bg-[hsl(var(--bg-surface)/0.92)] text-[hsl(var(--action-primary))]"
          data-testid="drop-overlay"
          role="status"
        >
          <UploadCloud className="h-14 w-14" aria-hidden="true" />
          <span className="text-lg font-medium">{tr("messagesInput.drop.title")}</span>
          <span className="text-sm text-[hsl(var(--text-muted))]">{tr("messagesInput.drop.hint")}</span>
        </div>
      )}
    </div>
  );
};

export default DropZone;
