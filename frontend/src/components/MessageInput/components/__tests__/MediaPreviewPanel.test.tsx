import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";

beforeEach(() => {
  let n = 0;
  (URL as unknown as { createObjectURL: (f: File) => string }).createObjectURL = () => `blob:fake-${n++}`;
  (URL as unknown as { revokeObjectURL: (u: string) => void }).revokeObjectURL = () => undefined;
});

import MediaPreviewPanel, { formatSize } from "../MediaPreviewPanel";
import type { MediaItem } from "../../hooks/useMessageInput";

const file = (name: string, type: string, size = 100) => new File([new Uint8Array(size)], name, { type });
const item = (name: string, type: string, caption = "", extra: Partial<MediaItem> = {}): MediaItem => ({ file: file(name, type), caption, ...extra });

const setup = (medias: MediaItem[], over: Partial<React.ComponentProps<typeof MediaPreviewPanel>> = {}) => {
  const props = {
    medias,
    loading: false,
    onRemoveMedia: vi.fn(),
    onCaptionChange: vi.fn(),
    onAudioKindChange: vi.fn(),
    onAddMore: vi.fn(),
    onCancelAll: vi.fn(),
    onUpload: vi.fn(),
    ...over,
  };
  render(<MediaPreviewPanel {...props} />);
  return props;
};

describe("formatSize", () => {
  it("bytes, KB e MB (com vírgula, como no WhatsApp Web)", () => {
    expect(formatSize(512)).toBe("512 B");
    expect(formatSize(340 * 1024)).toBe("340 KB");
    expect(formatSize(1.2 * 1024 * 1024)).toBe("1,2 MB");
    expect(formatSize(-1)).toBe("");
  });
});

describe("pré-visualização estilo WhatsApp Web", () => {
  it("mostra o arquivo selecionado grande, o nome no topo e a legenda num campo próprio", () => {
    setup([item("foto.png", "image/png", "oi")]);
    expect(screen.getByTestId("media-preview-name").textContent).toBe("foto.png");
    expect(within(screen.getByTestId("media-preview-main")).getByAltText("foto.png")).toBeInTheDocument();
    expect((screen.getByTestId("media-preview-caption") as HTMLInputElement).value).toBe("oi");
  });

  it("a legenda é editada do arquivo SELECIONADO", () => {
    const p = setup([item("a.png", "image/png"), item("b.png", "image/png")]);
    fireEvent.click(screen.getByTestId("media-preview-thumb-1"));
    expect(screen.getByTestId("media-preview-name").textContent).toBe("b.png");
    fireEvent.change(screen.getByTestId("media-preview-caption"), { target: { value: "legenda da b" } });
    expect(p.onCaptionChange).toHaveBeenCalledWith(1, "legenda da b");
  });

  it("a fileira de miniaturas troca o arquivo grande e marca o selecionado", () => {
    setup([item("a.png", "image/png"), item("b.pdf", "application/pdf")]);
    expect(screen.getByTestId("media-preview-thumb-0").getAttribute("aria-current")).toBe("true");
    fireEvent.click(screen.getByTestId("media-preview-thumb-1"));
    expect(screen.getByTestId("media-preview-thumb-1").getAttribute("aria-current")).toBe("true");
    expect(screen.getByTestId("media-preview-thumb-0").getAttribute("aria-current")).toBe("false");
  });

  it("documento e áudio mostram nome e tamanho no lugar da imagem", () => {
    setup([{ file: file("contrato.pdf", "application/pdf", 340 * 1024), caption: "" }]);
    const main = screen.getByTestId("media-preview-main");
    expect(within(main).getByText("contrato.pdf")).toBeInTheDocument();
    expect(within(main).getByText("340 KB")).toBeInTheDocument();
  });

  it("vídeo toca na pré-visualização", () => {
    setup([item("v.mp4", "video/mp4")]);
    expect(screen.getByTestId("media-preview-main").querySelector("video")).not.toBeNull();
  });

  it("Enter na legenda envia; Shift+Enter não", () => {
    const p = setup([item("a.png", "image/png")]);
    const cap = screen.getByTestId("media-preview-caption");
    fireEvent.keyDown(cap, { key: "Enter", shiftKey: true });
    expect(p.onUpload).not.toHaveBeenCalled();
    fireEvent.keyDown(cap, { key: "Enter" });
    expect(p.onUpload).toHaveBeenCalledTimes(1);
  });

  it("botão de enviar, cancelar tudo e remover um arquivo", () => {
    const p = setup([item("a.png", "image/png"), item("b.png", "image/png")]);
    fireEvent.click(screen.getByTestId("media-preview-send"));
    expect(p.onUpload).toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("media-preview-remove-1"));
    expect(p.onRemoveMedia).toHaveBeenCalledWith(1);
    fireEvent.click(screen.getByTestId("media-preview-cancel"));
    expect(p.onCancelAll).toHaveBeenCalled();
  });

  it("enviando: nada é clicável nem editável (não envia duas vezes)", () => {
    const p = setup([item("a.png", "image/png")], { loading: true });
    expect((screen.getByTestId("media-preview-send") as HTMLButtonElement).disabled).toBe(true);
    expect((screen.getByTestId("media-preview-caption") as HTMLInputElement).disabled).toBe(true);
    expect(screen.queryByTestId("media-preview-remove-0")).toBeNull();
    fireEvent.keyDown(screen.getByTestId("media-preview-caption"), { key: "Enter" });
    expect(p.onUpload).not.toHaveBeenCalled();
  });

  it("remover o último arquivo da fileira não deixa a seleção fora da lista", () => {
    const medias = [item("a.png", "image/png"), item("b.png", "image/png")];
    const props = { medias, loading: false, onRemoveMedia: vi.fn(), onCaptionChange: vi.fn(), onAudioKindChange: vi.fn(), onAddMore: vi.fn(), onCancelAll: vi.fn(), onUpload: vi.fn() };
    const { rerender } = render(<MediaPreviewPanel {...props} />);
    fireEvent.click(screen.getByTestId("media-preview-thumb-1"));
    rerender(<MediaPreviewPanel {...props} medias={[medias[0]]} />);
    expect(screen.getByTestId("media-preview-name").textContent).toBe("a.png");
  });

  it("áudio mostra a escolha voz/arquivo", () => {
    const p = setup([item("nota.mp3", "audio/mpeg", "", { audioKind: "file" })]);
    expect(screen.getAllByRole("radio").length).toBeGreaterThan(0);
    fireEvent.click(screen.getAllByRole("radio")[0]);
    expect(p.onAudioKindChange).toHaveBeenCalled();
  });
});
