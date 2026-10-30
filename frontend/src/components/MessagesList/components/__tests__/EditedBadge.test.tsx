import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";

const get = vi.fn();
vi.mock("../../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a) } }));

import EditedBadge from "../EditedBadge";
import { messagesReducer } from "../../hooks/messagesReducer";
import type { Message } from "../../types";

const msg = (over: Partial<Message> = {}): Message =>
  ({ id: "M1", body: "Teste 6 c", fromMe: false, createdAt: "2026-10-29T19:51:27Z", editCount: 3, ...over }) as Message;

const chain = [
  { id: 1, messageId: "M1", version: 1, body: "Teste 6", editedAt: "2026-10-29T19:51:27Z", editedBy: "contact" },
  { id: 2, messageId: "M1", version: 2, body: "Teste 6 a", editedAt: "2026-10-29T19:51:33Z", editedBy: "contact" },
  { id: 3, messageId: "M1", version: 3, body: "Teste 6 b", editedAt: "2026-10-29T19:51:38Z", editedBy: "me" },
  { id: 4, messageId: "M1", version: 4, body: "Teste 6 c", editedAt: "2026-10-29T19:51:43Z", editedBy: "contact" },
];

describe("selo Editada", () => {
  beforeEach(() => get.mockReset());

  it("não aparece em mensagem nunca editada, sem contador ou apagada", () => {
    for (const m of [msg({ editCount: 0 }), msg({ editCount: undefined }), msg({ isDeleted: true })]) {
      const { container, unmount } = render(<EditedBadge message={m} />);
      expect(container.querySelector('[data-testid="edited-badge"]')).toBeNull();
      unmount();
    }
  });

  it("uma edição mostra 'Editada'; várias mostram quantas", () => {
    const { rerender } = render(<EditedBadge message={msg({ editCount: 1 })} />);
    expect(screen.getByTestId("edited-badge").textContent).toBe("Editada");
    rerender(<EditedBadge message={msg({ editCount: 3 })} />);
    expect(screen.getByTestId("edited-badge").textContent).toBe("Editada 3 vezes");
  });

  it("clicar abre o histórico completo, da original à atual, com autor e horário", async () => {
    get.mockResolvedValue({ data: { edits: chain } });
    render(<EditedBadge message={msg()} />);
    fireEvent.click(screen.getByTestId("edited-badge"));

    await waitFor(() => expect(screen.getByTestId("edits-list")).toBeInTheDocument());
    expect(get).toHaveBeenCalledWith("/message/M1/edits");

    const v1 = screen.getByTestId("edit-version-1");
    expect(within(v1).getByText("Original")).toBeInTheDocument();
    expect(within(v1).getByText("Teste 6")).toBeInTheDocument();
    expect(within(v1).queryByText(/Editada por/)).toBeNull();

    expect(within(screen.getByTestId("edit-version-2")).getByText("Edição 1")).toBeInTheDocument();
    expect(within(screen.getByTestId("edit-version-2")).getByText("Editada pelo contato")).toBeInTheDocument();
    expect(within(screen.getByTestId("edit-version-3")).getByText("Editada por você")).toBeInTheDocument();

    const v4 = screen.getByTestId("edit-version-4");
    expect(within(v4).getByText("Atual")).toBeInTheDocument();
    expect(within(v4).getByText("Teste 6 c")).toBeInTheDocument();
    expect(screen.getAllByText("Atual")).toHaveLength(1);
    expect(screen.getAllByTestId(/^edit-version-/)).toHaveLength(4);
  });

  it("dez edições aparecem todas, em ordem", async () => {
    const dez = Array.from({ length: 11 }, (_, i) => ({ id: i + 1, messageId: "M1", version: i + 1, body: `v${i + 1}`, editedAt: `2026-10-29T19:51:${String(i).padStart(2, "0")}Z`, editedBy: "contact" }));
    get.mockResolvedValue({ data: { edits: dez } });
    render(<EditedBadge message={msg({ editCount: 10 })} />);
    fireEvent.click(screen.getByTestId("edited-badge"));
    await waitFor(() => expect(screen.getAllByTestId(/^edit-version-/)).toHaveLength(11));
    expect(screen.getByTestId("edited-badge").textContent).toBe("Editada 10 vezes");
  });

  it("fechar o painel o remove", async () => {
    get.mockResolvedValue({ data: { edits: chain } });
    render(<EditedBadge message={msg()} />);
    fireEvent.click(screen.getByTestId("edited-badge"));
    await waitFor(() => expect(screen.getByTestId("edits-list")).toBeInTheDocument());
    fireEvent.click(screen.getAllByRole("button", { name: "Fechar" })[0]);
    await waitFor(() => expect(screen.queryByTestId("edits-list")).toBeNull());
  });
});

describe("edição em tempo real", () => {
  it("o update que traz editCount e o texto novo faz o selo aparecer sem recarregar o chat", () => {
    const before = [msg({ editCount: 0, body: "Teste 6" })];
    const after = messagesReducer(before, { type: "UPDATE_MESSAGE", payload: msg({ editCount: 1, body: "Teste 6 a" }) });
    expect(after[0].body).toBe("Teste 6 a");
    expect(after[0].editCount).toBe(1);
  });
});

describe("posição do selo", () => {
  it("fica no rodapé à esquerda (o horário fica à direita), não no topo", () => {
    render(<EditedBadge message={msg()} />);
    const cls = screen.getByTestId("edited-badge").className;
    expect(cls).toContain("absolute");
    expect(cls).toContain("bottom-0");
    expect(cls).toContain("left-1.5");
    expect(cls).not.toContain("right-");
  });
});

describe("quem editou aparece no selo", () => {
  it("na mensagem NOSSA o selo diz 'Você editou'", () => {
    const { rerender } = render(<EditedBadge message={msg({ fromMe: true, editCount: 1 })} />);
    expect(screen.getByTestId("edited-badge").textContent).toBe("Você editou");
    rerender(<EditedBadge message={msg({ fromMe: true, editCount: 4 })} />);
    expect(screen.getByTestId("edited-badge").textContent).toBe("Você editou 4 vezes");
  });

  it("na mensagem do contato continua 'Editada'", () => {
    render(<EditedBadge message={msg({ fromMe: false, editCount: 1 })} />);
    expect(screen.getByTestId("edited-badge").textContent).toBe("Editada");
  });
});
