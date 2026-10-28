import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import TicketInfo from "../index";

vi.mock("../../../services/api", () => ({ default: { get: vi.fn(), post: vi.fn().mockResolvedValue({ data: {} }) } }));
vi.mock("../../../services/sse-client", () => ({ subscribeToSocket: () => () => undefined }));

const contact = { name: "Diomedes", number: "558388095596", lid: null };

describe("cabeçalho da conversa: identificação do canal e da conexão", () => {
  it("mostra o selo com a conexão do ticket", () => {
    render(<TicketInfo contact={contact} ticket={{ id: 5, channel: "whatsapp", connection: { kind: "whatsapp", id: 2, name: "Comercial", source: "whatsapp" } }} />);
    expect(screen.getByTestId("ticket-connection-badge")).toHaveTextContent("Comercial");
  });
  it("sem conexão mas com canal (conexão apagada): mostra o canal", () => {
    render(<TicketInfo contact={contact} ticket={{ id: 5, channel: "telegram" }} />);
    expect(screen.getByTestId("ticket-connection-badge")).toHaveTextContent("Telegram");
  });
});
