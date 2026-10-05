import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useStartChat } from "../useStartChat";

const post = vi.fn();
const navigate = vi.fn();
const toastError = vi.fn();

vi.mock("../../services/api", () => ({ default: { post: (...a: unknown[]) => post(...a) } }));
vi.mock("react-router", () => ({ useNavigate: () => navigate }));
vi.mock("../../errors/toastError", () => ({ default: (e: unknown) => toastError(e) }));

function Harness({ onStarted }: { onStarted?: () => void }) {
  const { startChat, connectionDialog } = useStartChat(onStarted);
  return (
    <>
      <button onClick={() => startChat(7)}>start</button>
      {connectionDialog}
    </>
  );
}

const conflict = {
  response: {
    status: 409,
    data: {
      code: "CONNECTION_REQUIRED",
      connections: [
        { id: 1, name: "Vendas", number: "5511111" },
        { id: 2, name: "Suporte", number: "5522222" },
      ],
    },
  },
};

describe("useStartChat", () => {
  beforeEach(() => {
    post.mockReset();
    navigate.mockReset();
    toastError.mockReset();
  });

  it("navega para o ticket quando o backend cria direto", async () => {
    post.mockResolvedValue({ data: { id: 42 } });
    const onStarted = vi.fn();
    render(<Harness onStarted={onStarted} />);
    fireEvent.click(screen.getByText("start"));

    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/tickets/42"));
    expect(post).toHaveBeenCalledWith("/tickets", { contactId: 7 });
    expect(onStarted).toHaveBeenCalled();
  });

  it("com 2+ conexões pergunta qual usar e reenvia com whatsappId", async () => {
    post.mockRejectedValueOnce(conflict).mockResolvedValueOnce({ data: { id: 43 } });
    render(<Harness />);
    fireEvent.click(screen.getByText("start"));

    expect(await screen.findByText(/Suporte/)).toBeTruthy();
    expect(toastError).not.toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText(/Suporte/));
    fireEvent.click(screen.getByText("Iniciar conversa"));

    await waitFor(() => expect(navigate).toHaveBeenCalledWith("/tickets/43"));
    expect(post).toHaveBeenLastCalledWith("/tickets", { contactId: 7, whatsappId: 2 });
  });

  it("outros erros viram toast e não abrem o diálogo", async () => {
    const err = { response: { status: 409, data: { code: "NO_CONNECTED_CONNECTION", error: "sem conexão" } } };
    post.mockRejectedValue(err);
    render(<Harness />);
    fireEvent.click(screen.getByText("start"));

    await waitFor(() => expect(toastError).toHaveBeenCalledWith(err));
    expect(screen.queryByText(/por qual conexão/)).toBeNull();
    expect(navigate).not.toHaveBeenCalled();
  });
});
