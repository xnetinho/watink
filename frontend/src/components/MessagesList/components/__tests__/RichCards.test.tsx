import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import MessageMedia from "../MessageMedia";
import ForwardedLabel from "../ForwardedLabel";
import type { Message } from "../../types";
import "../../../Calls/__tests__/helpers";

vi.mock("@/services/api", () => ({ default: { get: vi.fn(), post: vi.fn() } }));
vi.mock("@/errors/toastError", () => ({ default: vi.fn() }));

const msg = (over: Partial<Message> = {}): Message =>
  ({ id: "R1", body: "x", fromMe: false, createdAt: new Date().toISOString(), ...over }) as Message;
const withData = (mediaType: string, data: Record<string, unknown>) => msg({ mediaType, dataJson: JSON.stringify(data) });

describe("cartões de mensagem", () => {
  it("PIX mostra nome, chave, tipo e botão de copiar", () => {
    render(<MessageMedia message={withData("pix", { rich: { name: "Meu PIX", key: "+5583988095596", keyType: "PHONE" } })} />);
    const c = screen.getByTestId("pix-card");
    expect(c).toHaveTextContent("Meu PIX");
    expect(c).toHaveTextContent("+5583988095596");
    expect(c).toHaveTextContent("Copiar chave PIX");
  });

  it("evento mostra local e só oferece 'Entrar' com link https", () => {
    const { rerender } = render(<MessageMedia message={withData("event", { rich: { name: "Reunião", location: "Sala 1", start: 1793116800 } })} />);
    expect(screen.getByTestId("event-card")).toHaveTextContent("Sala 1");
    expect(screen.queryByText("Entrar na reunião")).toBeNull();
    rerender(<MessageMedia message={withData("event", { rich: { name: "Reunião", joinLink: "https://call.whatsapp.com/video/x" } })} />);
    expect(screen.getByText("Entrar na reunião")).toHaveAttribute("href", "https://call.whatsapp.com/video/x");
    rerender(<MessageMedia message={withData("event", { rich: { name: "Reunião", joinLink: "javascript:alert(1)" } })} />);
    expect(screen.queryByText("Entrar na reunião")).toBeNull();
  });

  it("evento cancelado não oferece entrar", () => {
    render(<MessageMedia message={withData("event", { rich: { name: "R", canceled: true, joinLink: "https://x.y/z" } })} />);
    expect(screen.getByText("Evento cancelado")).toBeInTheDocument();
    expect(screen.queryByText("Entrar na reunião")).toBeNull();
  });

  it("enquete mostra opções e se é única ou múltipla", () => {
    const { rerender } = render(<MessageMedia message={withData("poll", { rich: { question: "Q?", options: ["A", "B"], selectable: 1 } })} />);
    expect(screen.getByTestId("poll-card")).toHaveTextContent("Escolha 1 opção");
    rerender(<MessageMedia message={withData("poll", { rich: { question: "Q?", options: ["A", "B", "C"], selectable: 3 } })} />);
    expect(screen.getByTestId("poll-card")).toHaveTextContent("Escolha até 3 opções");
  });

  it("sticker aparece direto, sem botão de baixar nem modal", () => {
    render(<MessageMedia message={msg({ mediaType: "sticker", mediaUrl: "/public/media/a.webp" })} />);
    expect(screen.getByTestId("sticker")).toBeInTheDocument();
    expect(screen.queryByText(/baixar/i)).toBeNull();
  });
});

describe("selo de encaminhada", () => {
  it("encaminhada", () => {
    render(<ForwardedLabel message={withData("chat", { forwarded: true, forwardScore: 1 })} />);
    expect(screen.getByTestId("forwarded-label")).toHaveTextContent("Encaminhada");
    expect(screen.getByTestId("forwarded-label")).not.toHaveTextContent("com frequência");
  });
  it("encaminhada com frequência (score > 4)", () => {
    render(<ForwardedLabel message={withData("chat", { forwarded: true, forwardScore: 127 })} />);
    expect(screen.getByTestId("forwarded-label")).toHaveTextContent("Encaminhada com frequência");
  });
  it("mensagem normal não tem selo", () => {
    const { container } = render(<ForwardedLabel message={withData("chat", {})} />);
    expect(container).toBeEmptyDOMElement();
  });
});

describe("mensagens anteriores ao recurso (sem dado estruturado)", () => {
  it("enquete e evento antigos mostram o texto salvo, não um cartão vazio", () => {
    const { rerender } = render(<MessageMedia message={msg({ mediaType: "poll", body: "📊 Q?\n• A\n• B", dataJson: "{}" })} />);
    expect(screen.getByTestId("poll-card")).toHaveTextContent("Q?");
    rerender(<MessageMedia message={msg({ mediaType: "event", body: "📅 Reunião", dataJson: "{}" })} />);
    expect(screen.getByTestId("event-card")).toHaveTextContent("Reunião");
  });
});
