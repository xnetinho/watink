import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import DetailsPanel from "../DetailsPanel";
import { i18n } from "../../../../translate/i18n";

vi.mock("../../../../services/api", () => ({ default: { get: vi.fn().mockResolvedValue({ data: [] }), post: vi.fn(), put: vi.fn(), delete: vi.fn() } }));
vi.mock("../../../../hooks/useDeals", () => ({ useDeals: () => ({ deals: [], pipelines: [], loading: false, refetch: vi.fn() }) }));
vi.mock("../PipelinesSection", () => ({ default: () => null }));
vi.mock("../FlowsSection", () => ({ default: () => null }));
vi.mock("../HelpdeskSection", () => ({ default: () => null }));
vi.mock("../TicketTagsSection", () => ({ default: () => null }));
vi.mock("../../../Clients/ClientModal", () => ({ default: () => null }));
vi.mock("@/components/TicketHistory", () => ({ default: () => null }));

const ticket = {
  id: 7,
  channel: "telegram",
  connection: { kind: "telegram", id: 2, name: "Jarvis", source: "channel" as const },
  user: { name: "Admin" },
  contact: { id: 4, name: "Ana", number: "5511999990001" },
};

describe("painel de detalhes: tudo vem dos arquivos de idioma", () => {
  it("em português", () => {
    render(<DetailsPanel ticket={ticket} loading={false} onClose={() => undefined} />);
    expect(screen.getByText("Detalhes do Contato")).toBeInTheDocument();
    expect(screen.getByText("Dados")).toBeInTheDocument();
    expect(screen.getByText("Histórico")).toBeInTheDocument();
    expect(screen.getByLabelText("Fechar painel")).toBeInTheDocument();
    expect(screen.getByText("Atualizar contato")).toBeInTheDocument();
    expect(screen.getByTestId("details-connection")).toHaveTextContent("Conexão");
  });

  it("em inglês e espanhol, nada fica em português", () => {
    for (const [lng, title, sync] of [["en", "Contact details", "Refresh contact"], ["es", "Detalles del contacto", "Actualizar contacto"]] as const) {
      void i18n.changeLanguage(lng);
      const { unmount } = render(<DetailsPanel ticket={ticket} loading={false} onClose={() => undefined} />);
      expect(screen.getByText(title)).toBeInTheDocument();
      expect(screen.getByText(sync)).toBeInTheDocument();
      expect(screen.queryByText("Atualizar contato")).toBeNull();
      expect(screen.queryByText("Detalhes do Contato")).toBeNull();
      unmount();
    }
    void i18n.changeLanguage("pt");
  });

  it("grupo usa o título de grupo", () => {
    render(<DetailsPanel ticket={{ ...ticket, isGroup: true, contact: { ...ticket.contact, isGroup: true } }} loading={false} onClose={() => undefined} />);
    expect(screen.getByText("Detalhes do Grupo")).toBeInTheDocument();
  });
});
