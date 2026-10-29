import React from "react";
import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { TicketConnectionBadge } from "../TicketConnectionBadge";
import { channelDefinition, CHANNELS_CATALOG } from "../../config/channels";
import type { TicketConnection } from "../../types/Ticket";

const conn = (kind: string, name: string, id = 1): TicketConnection => ({ kind, id, name, source: kind === "whatsapp" ? "whatsapp" : "channel" });

describe("selo de identificação do ticket (canal + conexão)", () => {
  it("WhatsApp mostra o NOME da conexão (antes não mostrava nada)", () => {
    render(<TicketConnectionBadge channel="whatsapp" connection={conn("whatsapp", "Comercial")} />);
    const b = screen.getByTestId("ticket-connection-badge");
    expect(b).toHaveTextContent("Comercial");
    expect(b).toHaveAttribute("data-channel", "whatsapp");
  });

  it("Telegram mostra o nome da conexão, com a cor do canal", () => {
    render(<TicketConnectionBadge connection={conn("telegram", "Jarvis")} />);
    const b = screen.getByTestId("ticket-connection-badge");
    expect(b).toHaveTextContent("Jarvis");
    expect(b.className).toContain(channelDefinition("telegram").colorClass.split(" ")[0]);
  });

  it("duas conexões do MESMO canal se distinguem pelo nome e têm a mesma cor", () => {
    const { rerender } = render(<TicketConnectionBadge connection={conn("whatsapp", "Comercial", 1)} />);
    const a = screen.getByTestId("ticket-connection-badge");
    const aText = a.textContent;
    const aClass = a.className;
    rerender(<TicketConnectionBadge connection={conn("whatsapp", "Suporte", 2)} />);
    const b = screen.getByTestId("ticket-connection-badge");
    expect(b.textContent).not.toBe(aText);
    expect(b.className).toBe(aClass);
  });

  it("todo canal do catálogo é desenhado com o mesmo componente, sem código novo", () => {
    for (const def of CHANNELS_CATALOG) {
      const { unmount } = render(<TicketConnectionBadge connection={conn(def.kind, `Conexão ${def.kind}`)} />);
      expect(screen.getByTestId("ticket-connection-badge")).toHaveTextContent(`Conexão ${def.kind}`);
      unmount();
    }
  });

  it("canal que ainda não está no catálogo não quebra a tela (cai no genérico, com o nome da conexão)", () => {
    render(<TicketConnectionBadge connection={conn("canal_do_futuro", "Minha loja")} />);
    const b = screen.getByTestId("ticket-connection-badge");
    expect(b).toHaveTextContent("Minha loja");
    expect(b).toHaveAttribute("data-channel", "canal_do_futuro");
  });

  it("conexão apagada: mostra o nome do canal, nunca um nome inventado", () => {
    render(<TicketConnectionBadge channel="telegram" connection={null} />);
    expect(screen.getByTestId("ticket-connection-badge")).toHaveTextContent("Telegram");
  });

  it("sem nada, assume o canal padrão em vez de sumir", () => {
    render(<TicketConnectionBadge />);
    expect(screen.getByTestId("ticket-connection-badge")).toHaveTextContent("WhatsApp");
  });

  it("nome longo é cortado e o texto completo fica no title", () => {
    render(<TicketConnectionBadge connection={conn("whatsapp", "Atendimento Comercial Região Nordeste Filial 3")} />);
    const b = screen.getByTestId("ticket-connection-badge");
    expect(b.getAttribute("title")).toContain("Atendimento Comercial Região Nordeste Filial 3");
    expect(b.querySelector(".truncate")).not.toBeNull();
  });
});
