import React from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

const post = vi.fn();
const toastError = vi.fn();
vi.mock("../../services/api", () => ({ default: { post: (...a: unknown[]) => post(...a) } }));
vi.mock("../../errors/toastError", () => ({ default: (...a: unknown[]) => toastError(...a) }));

import EditMessageDialog from "../EditMessageDialog";

describe("EditMessageDialog", () => {
  beforeEach(() => {
    post.mockReset();
    toastError.mockReset();
  });

  it("abre com o texto atual e só habilita Salvar quando muda e não está vazio", () => {
    render(<EditMessageDialog messageId="M1" initialBody="texto antigo" onClose={() => undefined} />);
    const field = screen.getByTestId("edit-message-text") as HTMLTextAreaElement;
    const save = screen.getByTestId("edit-message-save") as HTMLButtonElement;
    expect(field.value).toBe("texto antigo");
    expect(save.disabled).toBe(true);

    fireEvent.change(field, { target: { value: "texto novo" } });
    expect(save.disabled).toBe(false);
    fireEvent.change(field, { target: { value: "   " } });
    expect(save.disabled).toBe(true);
    fireEvent.change(field, { target: { value: "texto antigo  " } });
    expect(save.disabled).toBe(true);
  });

  it("salvar chama a rota de edição com o texto aparado e fecha", async () => {
    post.mockResolvedValue({});
    const onClose = vi.fn();
    render(<EditMessageDialog messageId="M1" initialBody="a" onClose={onClose} />);
    fireEvent.change(screen.getByTestId("edit-message-text"), { target: { value: "  b  " } });
    fireEvent.click(screen.getByTestId("edit-message-save"));
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    expect(post).toHaveBeenCalledWith("/message/M1/edit", { body: "b" });
  });

  it("Ctrl+Enter salva", async () => {
    post.mockResolvedValue({});
    const onClose = vi.fn();
    render(<EditMessageDialog messageId="M1" initialBody="a" onClose={onClose} />);
    const field = screen.getByTestId("edit-message-text");
    fireEvent.change(field, { target: { value: "b" } });
    fireEvent.keyDown(field, { key: "Enter", ctrlKey: true });
    await waitFor(() => expect(post).toHaveBeenCalledTimes(1));
  });

  it("o servidor recusou (prazo vencido etc.): avisa, mantém aberto e libera Salvar de novo", async () => {
    post.mockRejectedValue(new Error("422"));
    const onClose = vi.fn();
    render(<EditMessageDialog messageId="M1" initialBody="a" onClose={onClose} />);
    fireEvent.change(screen.getByTestId("edit-message-text"), { target: { value: "b" } });
    fireEvent.click(screen.getByTestId("edit-message-save"));
    await waitFor(() => expect(toastError).toHaveBeenCalled());
    expect(onClose).not.toHaveBeenCalled();
    expect((screen.getByTestId("edit-message-save") as HTMLButtonElement).disabled).toBe(false);
  });

  it("legenda de mídia tem título próprio", () => {
    render(<EditMessageDialog messageId="M1" initialBody="a" isCaption onClose={() => undefined} />);
    expect(screen.getByText("Editar legenda")).toBeInTheDocument();
  });
});
