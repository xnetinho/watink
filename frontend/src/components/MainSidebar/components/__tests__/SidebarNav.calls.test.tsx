import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import SidebarNav from "../SidebarNav";
import { AuthContext } from "@/context/Auth/AuthContext";
import { TooltipProvider } from "@/components/ui/tooltip";
import { i18n } from "@/translate/i18n";

vi.mock("@/services/api", () => ({ default: { get: vi.fn(), put: vi.fn() } }));
void i18n.changeLanguage("pt");

function renderNav(user: Record<string, unknown>) {
  render(
    <MemoryRouter>
      <TooltipProvider>
        <AuthContext.Provider value={{ user, loading: false, isAuth: true, handleLogout: vi.fn() } as never}>
          <SidebarNav collapsed={false} activePlugins={[]} />
        </AuthContext.Provider>
      </TooltipProvider>
    </MemoryRouter>,
  );
}

describe("menu Chamadas", () => {
  it("aparece para quem tem calls:read", () => {
    renderNav({ id: 1, alcance: "proprio", permissions: ["calls:read"] });
    expect(screen.getByText("Chamadas")).toBeInTheDocument();
    expect(screen.getByText("Chamadas").closest("a")).toHaveAttribute("href", "/calls");
  });

  it("não aparece sem calls:read, mesmo com as outras permissões de chamada", () => {
    renderNav({ id: 1, alcance: "proprio", permissions: ["calls:receive", "calls:place", "calls:delete", "calls:manage"] });
    expect(screen.queryByText("Chamadas")).not.toBeInTheDocument();
  });

  it("não aparece para quem não tem nenhuma permissão", () => {
    renderNav({ id: 1, alcance: "proprio", permissions: [] });
    expect(screen.queryByText("Chamadas")).not.toBeInTheDocument();
  });

  it("alcance de empresa vê o menu (mesma regra do backend)", () => {
    renderNav({ id: 1, alcance: "tenant", permissions: [] });
    expect(screen.getByText("Chamadas")).toBeInTheDocument();
  });
});
