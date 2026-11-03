import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import StickerPicker from "../index";
import { i18n } from "@/translate/i18n";

const get = vi.fn();
const post = vi.fn();
const del = vi.fn();
vi.mock("../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), post: (...a: unknown[]) => post(...a), delete: (...a: unknown[]) => del(...a) } }));
const toastError = vi.fn();
vi.mock("../../../errors/toastError", () => ({ default: (...a: unknown[]) => toastError(...a) }));
void i18n.changeLanguage("pt");

const lib = (n: number) => Array.from({ length: n }, (_, i) => ({ id: i + 1, mediaUrl: `/public/media/${i + 1}.webp`, animated: false }));

beforeEach(() => {
  get.mockReset();
  post.mockReset().mockResolvedValue({});
  del.mockReset().mockResolvedValue({});
  toastError.mockReset();
});

describe("StickerPicker", () => {
  it("carrega a biblioteca da empresa e mostra uma figurinha por item", async () => {
    get.mockResolvedValue({ data: lib(3) });
    render(<StickerPicker ticketId="9" onSent={vi.fn()} />);
    expect(await screen.findAllByTestId("sticker-picker-item")).toHaveLength(3);
    expect(get).toHaveBeenCalledWith("/stickers");
  });

  it("biblioteca vazia explica como salvar a primeira", async () => {
    get.mockResolvedValue({ data: [] });
    render(<StickerPicker ticketId="9" onSent={vi.fn()} />);
    expect(await screen.findByTestId("sticker-picker-empty")).toHaveTextContent(/Salvar figurinha/);
  });

  it("um clique envia a figurinha no ticket aberto pelo ID da biblioteca (nunca por URL)", async () => {
    get.mockResolvedValue({ data: lib(2) });
    const onSent = vi.fn();
    render(<StickerPicker ticketId="9" onSent={onSent} />);
    fireEvent.click((await screen.findAllByTestId("sticker-picker-item"))[1]);
    await waitFor(() => expect(post).toHaveBeenCalledWith("/messages/9", { stickerId: 2 }));
    await waitFor(() => expect(onSent).toHaveBeenCalled());
  });

  it("falha no envio avisa e NÃO fecha o painel", async () => {
    get.mockResolvedValue({ data: lib(1) });
    post.mockRejectedValue(new Error("falhou"));
    const onSent = vi.fn();
    render(<StickerPicker ticketId="9" onSent={onSent} />);
    fireEvent.click(await screen.findByTestId("sticker-picker-item"));
    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(onSent).not.toHaveBeenCalled();
  });

  it("remover tira da lista, sem enviar", async () => {
    get.mockResolvedValue({ data: lib(2) });
    render(<StickerPicker ticketId="9" onSent={vi.fn()} />);
    await screen.findAllByTestId("sticker-picker-item");
    fireEvent.click(screen.getAllByTestId("sticker-picker-remove")[0]);
    await waitFor(() => expect(del).toHaveBeenCalledWith("/stickers/1"));
    await waitFor(() => expect(screen.getAllByTestId("sticker-picker-item")).toHaveLength(1));
    expect(post).not.toHaveBeenCalled();
  });

  it("falha ao carregar mostra o erro e a biblioteca vazia, sem travar", async () => {
    get.mockRejectedValue(new Error("rede"));
    render(<StickerPicker ticketId="9" onSent={vi.fn()} />);
    expect(await screen.findByTestId("sticker-picker-empty")).toBeInTheDocument();
    expect(toastError).toHaveBeenCalled();
  });
});
