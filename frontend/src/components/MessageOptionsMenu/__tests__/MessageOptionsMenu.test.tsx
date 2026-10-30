import React from "react";
import { expect, describe, it, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import MessageOptionsMenu from "../index";

const mockApiPost = vi.fn();
const mockApiGet = vi.fn();

vi.mock("../../../services/api", () => ({
  default: { post: (...args: unknown[]) => mockApiPost(...args), get: (...args: unknown[]) => mockApiGet(...args), delete: vi.fn() },
}));

vi.mock("../../../errors/toastError", () => ({ default: vi.fn() }));

describe("MessageOptionsMenu — reações rápidas", () => {
  beforeEach(() => {
    mockApiPost.mockReset();
    mockApiPost.mockResolvedValue({ data: { message: "ok" } });
  });

  it("sends POST /message/:id/react with the picked emoji", async () => {
    const handleClose = vi.fn();
    render(
      <MessageOptionsMenu
        message={{ id: 42, fromMe: false }}
        menuOpen
        handleClose={handleClose}
        anchorEl={null}
      />
    );

    fireEvent.click(screen.getByLabelText("Reagir com 👍"));

    await waitFor(() =>
      expect(mockApiPost).toHaveBeenCalledWith("/message/42/react", { reaction: "👍" })
    );
    expect(handleClose).toHaveBeenCalled();
  });
});

describe("MessageOptionsMenu — alinhamento do popup", () => {
  it("mirrors the full bounding rect of anchorEl onto the invisible trigger, not just a zero-size corner point", () => {
    const anchorEl = document.createElement("button");
    document.body.appendChild(anchorEl);
    vi.spyOn(anchorEl, "getBoundingClientRect").mockReturnValue({
      top: 120,
      left: 340,
      right: 356,
      bottom: 136,
      width: 16,
      height: 16,
      x: 340,
      y: 120,
      toJSON: () => ({}),
    } as DOMRect);

    const { container } = render(
      <MessageOptionsMenu
        message={{ id: 1, fromMe: false }}
        menuOpen={false}
        handleClose={() => {}}
        anchorEl={anchorEl}
      />
    );

    const invisibleTrigger = container.querySelector('[style*="position: fixed"]') as HTMLElement;
    expect(invisibleTrigger).toBeTruthy();
    expect(invisibleTrigger.style.top).toBe("120px");
    expect(invisibleTrigger.style.left).toBe("340px");
    expect(invisibleTrigger.style.width).toBe("16px");
    expect(invisibleTrigger.style.height).toBe("16px");

    document.body.removeChild(anchorEl);
  });
});

describe("MessageOptionsMenu — lado em que abre", () => {
  const open = (fromMe: boolean) =>
    render(<MessageOptionsMenu message={{ id: 7, fromMe }} menuOpen handleClose={() => undefined} anchorEl={null} />);

  it("na mensagem NOSSA (colada na borda direita da tela) o menu abre para a ESQUERDA, para não ser cortado", () => {
    open(true);
    const menu = screen.getByTestId("message-options-content");
    expect(menu.getAttribute("data-side")).toBe("left");
    expect(menu.getAttribute("data-side-pref")).toBe("left");
  });

  it("na mensagem do contato (à esquerda da tela) o menu abre para baixo/direita, como sempre", () => {
    open(false);
    const menu = screen.getByTestId("message-options-content");
    expect(menu.getAttribute("data-side")).toBe("bottom");
    expect(menu.getAttribute("data-side-pref")).toBe("right");
  });
});

describe("MessageOptionsMenu — Dados da mensagem", () => {
  // No app, handleClose fecha o menu ao escolher um item; com menuOpen fixo o Radix disputa o foco entre menu e diálogo.
  const Harness: React.FC<{ message: Record<string, unknown>; channel: string }> = ({ message, channel }) => {
    const [openMenu, setOpenMenu] = React.useState(true);
    return <MessageOptionsMenu message={message} menuOpen={openMenu} handleClose={() => setOpenMenu(false)} anchorEl={null} channel={channel} />;
  };
  const open = (message: Record<string, unknown>, channel = "whatsapp") => render(<Harness message={message} channel={channel} />);

  it("aparece em mensagem nossa já entregue e abre o painel com os horários", async () => {
    mockApiGet.mockResolvedValue({ data: { receipts: [{ senderJid: "", name: "", delivered: "2026-10-30T14:00:00Z", read: null, played: null }] } });
    open({ id: "M9", fromMe: true, ack: 2, mediaType: "chat", createdAt: "2026-10-30T13:59:00Z" });
    fireEvent.click(screen.getByTestId("message-receipts-item"));
    await waitFor(() => expect(screen.getByTestId("receipt-delivered")).toHaveTextContent(/30\/10\/2026/));
    expect(mockApiGet).toHaveBeenCalledWith("/message/M9/receipts");
  });

  it("não aparece em mensagem do contato, com erro ou em canal sem recibo", () => {
    open({ id: 1, fromMe: false, ack: 3 });
    expect(screen.queryByTestId("message-receipts-item")).toBeNull();
  });
  it("não aparece no Telegram", () => {
    open({ id: 2, fromMe: true, ack: 3 }, "telegram");
    expect(screen.queryByTestId("message-receipts-item")).toBeNull();
  });
});
