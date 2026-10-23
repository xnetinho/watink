import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import MessageMedia from "../MessageMedia";
import { hasMediaBubble } from "../MessageBubble";
import type { Message } from "../../types";
import "../../../Calls/__tests__/helpers";

vi.mock("@/services/api", () => ({ default: { get: vi.fn(), post: vi.fn() } }));
vi.mock("@/errors/toastError", () => ({ default: vi.fn() }));

const msg = (over: Partial<Message> = {}): Message =>
  ({ id: "VO1", body: "", fromMe: false, mediaType: "view_once", createdAt: new Date().toISOString(), ...over }) as Message;

// O WhatsApp não entrega a visualização única a aparelhos vinculados: chega só um aviso, sem conteúdo.
// O chat precisa mostrar que o cliente mandou algo e orientar a ver no celular, SEM oferecer um download
// que nunca funcionaria.
describe("MessageMedia — visualização única", () => {
  it("tem bolha própria, mesmo sem mediaUrl", () => {
    expect(hasMediaBubble(msg())).toBe(true);
  });

  it("mostra o aviso e orienta a ver no celular", () => {
    render(<MessageMedia message={msg()} />);
    const box = screen.getByTestId("view-once-notice");
    expect(box).toHaveTextContent("Visualização única");
    expect(box).toHaveTextContent(/celular/i);
  });

  it("NÃO oferece download nem miniatura desfocada", () => {
    render(<MessageMedia message={msg()} />);
    expect(screen.queryByRole("button", { name: /baixar|download/i })).not.toBeInTheDocument();
    expect(screen.queryByRole("img")).not.toBeInTheDocument();
  });

  it("uma imagem comum continua sem o aviso", () => {
    render(<MessageMedia message={msg({ mediaType: "image", mediaUrl: "/x.jpg" })} />);
    expect(screen.queryByTestId("view-once-notice")).not.toBeInTheDocument();
  });
});
