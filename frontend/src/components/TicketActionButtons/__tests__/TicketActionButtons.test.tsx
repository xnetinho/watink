import React from "react";
import { expect, describe, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import TicketActionButtons from "../index";
import { i18n } from "../../../translate/i18n";
import { WhatsAppsContext } from "../../../context/WhatsApp/WhatsAppsContext";
import { makeCtx, withCalls } from "../../Calls/__tests__/helpers";

vi.mock("../../../services/api", () => ({
  default: { put: vi.fn(), delete: vi.fn() },
}));

vi.mock("../../../errors/toastError", () => ({ default: vi.fn() }));

const openTicket = {
  id: 1,
  status: "open" as const,
  contact: { name: "Ana" },
};

describe("TicketActionButtons — menu de 3 pontinhos", () => {
  it("opens TicketOptionsMenu anchored to the trigger button when clicked", async () => {
    const queryClient = new QueryClient();
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <TicketActionButtons ticket={openTicket} />
        </MemoryRouter>
      </QueryClientProvider>
    );

    const transferLabel = i18n.t("ticketOptionsMenu.transfer") as string;

    // The dropdown content must not exist before the trigger is clicked.
    expect(screen.queryByText(transferLabel)).not.toBeInTheDocument();

    const trigger = screen.getByLabelText("Mais opções");
    fireEvent.pointerDown(trigger, { pointerId: 1, button: 0 });
    fireEvent.click(trigger);

    // If TicketOptionsMenu had no DropdownMenuTrigger anchoring it, Radix's
    // Popper would have nothing to position the content against — this
    // assertion is what regresses if that wiring breaks again.
    await waitFor(() => expect(screen.getByText(transferLabel)).toBeInTheDocument());
  });
});

describe("TicketActionButtons — botão de ligar", () => {
  const renderWith = (ticket: Parameters<typeof TicketActionButtons>[0]["ticket"], canPlace: boolean) => {
    const queryClient = new QueryClient();
    const ctx = {
      ...makeCtx({ canPlace }),
    };
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          {withCalls(
            ctx,
            <WhatsAppsContext.Provider
              value={{ whatsApps: [{ id: 1, name: "Vendas", status: "CONNECTED", proxyMode: "none" }], loading: false, reloadWhatsApps: async () => undefined }}
            >
              <TicketActionButtons ticket={ticket} />
            </WhatsAppsContext.Provider>,
          )}
        </MemoryRouter>
      </QueryClientProvider>,
    );
  };

  it.each(["open", "pending", "closed"] as const)("aparece em ticket individual %s com calls:place", (status) => {
    renderWith({ id: 1, status, whatsappId: 1, contact: { name: "Ana" } }, true);
    expect(screen.getByTestId("place-call")).toBeInTheDocument();
  });

  it("não aparece sem calls:place", () => {
    renderWith({ id: 1, status: "open", whatsappId: 1, contact: { name: "Ana" } }, false);
    expect(screen.queryByTestId("place-call")).not.toBeInTheDocument();
  });

  it("não aparece em grupo", () => {
    renderWith({ id: 1, status: "open", whatsappId: 1, isGroup: true, contact: { name: "Grupo" } }, true);
    expect(screen.queryByTestId("place-call")).not.toBeInTheDocument();
  });
});
