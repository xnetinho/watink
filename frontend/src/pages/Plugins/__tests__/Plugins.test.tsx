import React from "react";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi, beforeEach } from "vitest";

import { AuthContext } from "../../../context/Auth/AuthContext";
import Plugins from "..";
import { EMBEDDED_PLUGINS } from "../embeddedPlugins";

const get = vi.fn();
const post = vi.fn();
vi.mock("../../../services/api", () => ({ default: { get: (...a: unknown[]) => get(...a), post: (...a: unknown[]) => post(...a) } }));
vi.mock("../../../errors/toastError", () => ({ default: vi.fn() }));
vi.mock("react-toastify", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));
vi.mock("../../../components/Can", () => ({
  default: ({ yes }: { yes: () => React.ReactNode }) => <>{yes()}</>,
}));

const renderPage = () =>
  render(
    <AuthContext.Provider value={{ user: { id: 1, name: "Admin" } } as never}>
      <Plugins />
    </AuthContext.Provider>,
  );

describe("Plugins", () => {
  beforeEach(() => {
    get.mockReset();
    post.mockReset();
  });

  it("lista os 5 plugins embarcados e marca os ativos da empresa", async () => {
    get.mockResolvedValue({ data: { active: ["helpdesk"] } });
    renderPage();

    for (const p of EMBEDDED_PLUGINS) {
      expect(await screen.findByTestId(`plugin-${p.slug}`)).toBeInTheDocument();
    }
    expect(EMBEDDED_PLUGINS).toHaveLength(5);
    await waitFor(() => expect(screen.getByRole("switch", { name: /Desativar Helpdesk/ })).toBeChecked());
    expect(screen.getByRole("switch", { name: /Ativar Webchat/ })).not.toBeChecked();
  });

  it("ativar chama POST /plugins/:slug/activate e atualiza o estado", async () => {
    get.mockResolvedValue({ data: { active: [] } });
    post.mockResolvedValue({ data: {} });
    renderPage();

    const toggle = await screen.findByRole("switch", { name: /Ativar Grupos e Comunidades/ });
    await waitFor(() => expect(toggle).not.toBeDisabled());
    await userEvent.click(toggle);

    await waitFor(() => expect(post).toHaveBeenCalledWith("/plugins/groups/activate"));
    await waitFor(() => expect(screen.getByRole("switch", { name: /Desativar Grupos e Comunidades/ })).toBeChecked());
  });

  it("desativar chama POST /plugins/:slug/deactivate", async () => {
    get.mockResolvedValue({ data: { active: ["webchat"] } });
    post.mockResolvedValue({ data: {} });
    renderPage();

    const toggle = await screen.findByRole("switch", { name: /Desativar Webchat/ });
    await waitFor(() => expect(toggle).not.toBeDisabled());
    await userEvent.click(toggle);

    await waitFor(() => expect(post).toHaveBeenCalledWith("/plugins/webchat/deactivate"));
  });

  it("não oferece compra, checkout nem licença", async () => {
    get.mockResolvedValue({ data: { active: [] } });
    renderPage();
    await screen.findByTestId("plugin-helpdesk");
    expect(screen.queryByText(/R\$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/checkout|licen[cç]a|carrinho|marketplace/i)).not.toBeInTheDocument();
  });
});
