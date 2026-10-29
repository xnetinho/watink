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
  ({ id: "U1", body: "", fromMe: false, createdAt: new Date().toISOString(), ...over }) as Message;

// O WhatsApp cria tipos novos de mensagem com o tempo. O que o chat não conhece não pode sumir nem aparecer vazio.
describe("MessageMedia — mensagem não suportada", () => {
  it("o engine marca 'unsupported' e o chat avisa, com o tipo", () => {
    const m = msg({ mediaType: "unsupported", body: "listMessage" });
    expect(hasMediaBubble(m)).toBe(true);
    render(<MessageMedia message={m} />);
    const box = screen.getByTestId("unsupported-notice");
    expect(box).toHaveTextContent(/não suportada/i);
    expect(box).toHaveTextContent("listMessage");
  });

  it("um mediaType que o frontend nunca viu também vira aviso (futuro-prova)", () => {
    const m = msg({ mediaType: "tipo_novo_2030" });
    expect(hasMediaBubble(m)).toBe(true);
    render(<MessageMedia message={m} />);
    expect(screen.getByTestId("unsupported-notice")).toHaveTextContent("tipo_novo_2030");
  });

  it("tipos conhecidos não disparam o aviso", () => {
    for (const t of ["chat", "poll", "event", "view_once"]) {
      const { container } = render(<MessageMedia message={msg({ mediaType: t, body: "x" })} />);
      expect(container.querySelector('[data-testid="unsupported-notice"]'), t).toBeNull();
    }
  });
});
