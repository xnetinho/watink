/** O arraste carrega arquivos (e não, por exemplo, texto selecionado ou uma imagem da própria página)? */
export function dragHasFiles(dt: Pick<DataTransfer, "types"> | null | undefined): boolean {
  if (!dt || !dt.types) return false;
  return Array.from(dt.types).includes("Files");
}

/** Arquivos de um drop. Pastas e itens sem arquivo (texto, links) ficam de fora. */
export function filesFromDrop(dt: Pick<DataTransfer, "files" | "items"> | null | undefined): File[] {
  if (!dt) return [];
  const out: File[] = [];
  if (dt.items && dt.items.length) {
    for (const item of Array.from(dt.items)) {
      if (item.kind !== "file") continue;
      const entry = (item as DataTransferItem & { webkitGetAsEntry?: () => { isDirectory?: boolean } | null }).webkitGetAsEntry?.();
      if (entry?.isDirectory) continue; // arrastar uma pasta não envia nada: o servidor lê um arquivo por vez
      const f = item.getAsFile();
      if (f) out.push(f);
    }
    if (out.length) return out;
  }
  return Array.from(dt.files ?? []).filter((f) => f.size > 0 || f.type !== "");
}
