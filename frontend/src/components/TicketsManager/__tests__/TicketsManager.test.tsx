import React from "react";
import { expect, describe, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import TicketsManager from "../index";

// ── Mocks ────────────────────────────────────────────────────────────────────

vi.mock("../../../context/Tickets/TicketsContext", () => ({
  useTicketsContext: () => ({}),
}));

vi.mock("../../../context/Auth/AuthContext", () => ({
  AuthContext: React.createContext({ user: { queues: [] } }),
}));

vi.mock("../../TicketsList", () => ({
  default: (props: Record<string, unknown>) => (
    <div data-testid="tickets-list" data-is-group={String(props.isGroup)} />
  ),
}));

vi.mock("../../NewTicketModal/NewTicketModal", () => ({
  NewTicketModal: () => <div />,
}));

vi.mock("../../TicketsQueueSelect", () => ({
  default: () => <div />,
}));

vi.mock("../../TicketsTagFilter", () => ({
  default: () => <div />,
}));

vi.mock("../../Can", () => ({
  Can: ({ yes }: { yes: () => React.ReactNode }) => <>{yes()}</>,
}));

// ── Tests ─────────────────────────────────────────────────────────────────────

describe("TicketsManager", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renderiza as 4 pills de status", () => {
    render(<TicketsManager />);
    expect(screen.getByText("Todos")).toBeTruthy();
    expect(screen.getByText("Abertos")).toBeTruthy();
    expect(screen.getByText("Aguardando")).toBeTruthy();
    expect(screen.getByText("Fechados")).toBeTruthy();
  });

  it("renderiza os chips Grupos e Não lidas", () => {
    render(<TicketsManager />);
    expect(screen.getByText("Grupos")).toBeTruthy();
    expect(screen.getByText("Não lidas")).toBeTruthy();
  });

  it("aba Grupos começa inativa e Todos começa ativa", () => {
    render(<TicketsManager />);
    expect(screen.getByText("Grupos").closest("button")!.className).not.toContain("bg-primary");
    expect(screen.getByText("Todos").closest("button")!.className).toContain("bg-primary");
  });

  it("clicar em Grupos ativa a aba", () => {
    render(<TicketsManager />);
    const chip = screen.getByText("Grupos").closest("button")!;
    fireEvent.click(chip);
    expect(chip.className).toContain("bg-primary");
  });

  it("clicar em Grupos duas vezes mantém a aba ativa (sem toggle, #403)", () => {
    render(<TicketsManager />);
    const chip = screen.getByText("Grupos").closest("button")!;
    fireEvent.click(chip);
    fireEvent.click(chip);
    expect(chip.className).toContain("bg-primary");
    expect(screen.getByTestId("tickets-list").getAttribute("data-is-group")).toBe("true");
  });

  it("voltar para Todos após Grupos desativa Grupos", () => {
    render(<TicketsManager />);
    fireEvent.click(screen.getByText("Grupos"));
    fireEvent.click(screen.getByText("Todos"));
    expect(screen.getByText("Grupos").closest("button")!.className).not.toContain("bg-primary");
    expect(screen.getByTestId("tickets-list").getAttribute("data-is-group")).toBe("false");
  });

  it("clicar em Não lidas ativa o chip", () => {
    render(<TicketsManager />);
    const chip = screen.getByText("Não lidas").closest("button")!;
    fireEvent.click(chip);
    expect(chip.className).toContain("border-primary");
  });

  it("TicketsList recebe isGroup=false por padrão (só conversas individuais)", () => {
    render(<TicketsManager />);
    const list = screen.getByTestId("tickets-list");
    expect(list.getAttribute("data-is-group")).toBe("false");
  });

  it("TicketsList recebe isGroup=true ao selecionar Grupos", () => {
    render(<TicketsManager />);
    fireEvent.click(screen.getByText("Grupos"));
    expect(screen.getByTestId("tickets-list").getAttribute("data-is-group")).toBe("true");
  });

  it("mantém as abas Todos/Abertos/Aguardando/Fechados visíveis mesmo com Grupos selecionada", () => {
    render(<TicketsManager />);
    fireEvent.click(screen.getByText("Grupos"));

    expect(screen.getByText("Todos")).toBeInTheDocument();
    expect(screen.getByText("Abertos")).toBeInTheDocument();
    expect(screen.getByText("Aguardando")).toBeInTheDocument();
    expect(screen.getByText("Fechados")).toBeInTheDocument();
    expect(screen.getByText("Grupos")).toBeInTheDocument();
  });

  it("permite alternar de Grupos de volta para outra aba clicando nela diretamente", () => {
    render(<TicketsManager />);
    fireEvent.click(screen.getByText("Grupos"));
    fireEvent.click(screen.getByText("Abertos"));

    const list = screen.getByTestId("tickets-list");
    expect(list.getAttribute("data-is-group")).toBe("false");
  });
});
