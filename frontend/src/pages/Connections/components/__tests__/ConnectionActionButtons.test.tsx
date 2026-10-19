import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ConnectionActionButtons } from "../ConnectionActionButtons";
import type { ConnectionSession } from "../../connectionsTypes";

const baseWhatsApp = (over: Partial<ConnectionSession> = {}): ConnectionSession => ({
  id: 1,
  name: "Vendas",
  status: "DISCONNECTED",
  ...over,
});

const noop = () => undefined;

// Radix DropdownMenu abre no pointerdown do trigger; fireEvent.click (que o jsdom não compõe em
// pointerdown+pointerup) não é suficiente.
const openMenu = () => {
  const trigger = screen.getByRole("button", { name: "" });
  fireEvent.pointerDown(trigger, { button: 0, ctrlKey: false, pointerType: "mouse" });
  fireEvent.pointerUp(trigger, { button: 0, ctrlKey: false, pointerType: "mouse" });
};

describe("ConnectionActionButtons", () => {
  it("desconectado: mostra Conectar e NÃO mostra Reiniciar Sessão no menu", async () => {
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ status: "DISCONNECTED" })}
        onConnect={noop}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    expect(screen.getByRole("button", { name: /conectar/i })).toBeInTheDocument();
    openMenu();
    expect(await screen.findByText(/editar/i)).toBeInTheDocument();
    expect(screen.queryByText(/reiniciar sessão/i)).not.toBeInTheDocument();
  });

  it("conectado: mostra Reiniciar Sessão no menu (não é o estado desconectado)", async () => {
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ status: "CONNECTED" })}
        onConnect={noop}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    openMenu();
    expect(await screen.findByText(/reiniciar sessão/i)).toBeInTheDocument();
  });

  it("QRCODE: mostra Reiniciar Sessão no menu (aguardando leitura, não desconectado)", async () => {
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ status: "QRCODE" })}
        onConnect={noop}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    openMenu();
    expect(await screen.findByText(/reiniciar sessão/i)).toBeInTheDocument();
  });

  it("pending: Reiniciar Sessão (quando visível) fica marcado como desabilitado (Radix bloqueia o clique via CSS pointer-events; jsdom não aplica CSS, então o teste verifica o atributo em vez do clique)", async () => {
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ status: "CONNECTED" })}
        pending
        onConnect={noop}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    openMenu();
    const item = await screen.findByText(/reiniciar sessão/i);
    expect(item.closest('[role="menuitem"]')).toHaveAttribute("data-disabled");
  });

  it("sem pending: Reiniciar Sessão não tem o atributo de desabilitado", async () => {
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ status: "CONNECTED" })}
        onConnect={noop}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    openMenu();
    const item = await screen.findByText(/reiniciar sessão/i);
    expect(item.closest('[role="menuitem"]')).not.toHaveAttribute("data-disabled");
  });

  it("timeout: também mostra Conectar (mesmo tratamento que desconectado)", () => {
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ status: "TIMEOUT" })}
        onConnect={noop}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    expect(screen.getByRole("button", { name: /conectar/i })).toBeInTheDocument();
  });

  it("conectado: NÃO mostra Conectar, mostra Desconectar", () => {
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ status: "CONNECTED" })}
        onConnect={noop}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    expect(screen.queryByRole("button", { name: /^conectar$/i })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /desconectar/i })).toBeInTheDocument();
  });

  it("pending: o botão Conectar fica desabilitado (evita o segundo POST no lock de 10s)", () => {
    const onConnect = vi.fn();
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ status: "DISCONNECTED" })}
        pending
        onConnect={onConnect}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    const btn = screen.getByRole("button", { name: /conectar/i });
    expect(btn).toBeDisabled();
    fireEvent.click(btn);
    expect(onConnect).not.toHaveBeenCalled();
  });

  it("não pending: clicar Conectar chama onConnect com o id", () => {
    const onConnect = vi.fn();
    render(
      <ConnectionActionButtons
        whatsApp={baseWhatsApp({ id: 42, status: "DISCONNECTED" })}
        onConnect={onConnect}
        onDisconnect={noop}
        onRequestQrCode={noop}
        onEdit={noop}
        onDelete={noop}
        onConfigure={noop}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /conectar/i }));
    expect(onConnect).toHaveBeenCalledWith(42);
  });
});
