import React from "react";
import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import DropZone from "../DropZone";

const files = [new File(["x"], "foto.png", { type: "image/png" })];
const withFiles = { dataTransfer: { types: ["Files"], files, items: [{ kind: "file", getAsFile: () => files[0], webkitGetAsEntry: () => ({ isDirectory: false }) }], dropEffect: "" } };
const withText = { dataTransfer: { types: ["text/plain"], files: [], items: [] } };

describe("DropZone (arrastar e soltar na conversa)", () => {
  it("arrastar arquivo sobre a conversa mostra a camada 'Solte os arquivos aqui'", () => {
    render(<DropZone onFiles={vi.fn()}><div>conversa</div></DropZone>);
    expect(screen.queryByTestId("drop-overlay")).toBeNull();
    fireEvent.dragEnter(screen.getByTestId("drop-zone"), withFiles);
    expect(screen.getByTestId("drop-overlay")).toBeInTheDocument();
    expect(screen.getByText("Solte os arquivos aqui")).toBeInTheDocument();
  });

  it("soltar entrega os arquivos e some com a camada", () => {
    const onFiles = vi.fn();
    render(<DropZone onFiles={onFiles}><div>conversa</div></DropZone>);
    const zone = screen.getByTestId("drop-zone");
    fireEvent.dragEnter(zone, withFiles);
    fireEvent.drop(zone, withFiles);
    expect(onFiles).toHaveBeenCalledTimes(1);
    expect(onFiles.mock.calls[0][0].map((f: File) => f.name)).toEqual(["foto.png"]);
    expect(screen.queryByTestId("drop-overlay")).toBeNull();
  });

  it("a camada não pisca ao cruzar elementos filhos (conta entradas e saídas)", () => {
    render(<DropZone onFiles={vi.fn()}><span data-testid="filho">x</span></DropZone>);
    const zone = screen.getByTestId("drop-zone");
    fireEvent.dragEnter(zone, withFiles); // entra na zona
    fireEvent.dragEnter(screen.getByTestId("filho"), withFiles); // entra num filho
    fireEvent.dragLeave(zone, withFiles); // sai da zona para o filho: ainda dentro
    expect(screen.getByTestId("drop-overlay")).toBeInTheDocument();
    fireEvent.dragLeave(screen.getByTestId("filho"), withFiles); // sai de vez
    expect(screen.queryByTestId("drop-overlay")).toBeNull();
  });

  it("arrastar texto (não arquivo) não faz nada", () => {
    const onFiles = vi.fn();
    render(<DropZone onFiles={onFiles}><div>conversa</div></DropZone>);
    const zone = screen.getByTestId("drop-zone");
    fireEvent.dragEnter(zone, withText);
    expect(screen.queryByTestId("drop-overlay")).toBeNull();
    fireEvent.drop(zone, withText);
    expect(onFiles).not.toHaveBeenCalled();
  });

  it("desligada (ticket fechado / canal bloqueado) o arraste passa direto", () => {
    const onFiles = vi.fn();
    render(<DropZone onFiles={onFiles} disabled><div>conversa</div></DropZone>);
    const zone = screen.getByTestId("drop-zone");
    fireEvent.dragEnter(zone, withFiles);
    fireEvent.drop(zone, withFiles);
    expect(screen.queryByTestId("drop-overlay")).toBeNull();
    expect(onFiles).not.toHaveBeenCalled();
  });
});
