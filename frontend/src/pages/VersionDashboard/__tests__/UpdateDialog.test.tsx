import React from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import UpdateDialog from "../components/UpdateDialog";

const base = {
  open: true,
  onOpenChange: () => {},
  availableVersion: "1.2.0",
  frontendVersion: "1.1.0",
  releaseMeta: { breaking: false, minCompatibleFrom: "", migrationNotes: "" },
  changelog: ["RLS real"],
  blockedByCompatibility: false,
};

describe("UpdateDialog", () => {
  it("informa o comando de atualização e não oferece atualizar pelo painel", () => {
    render(<UpdateDialog {...base} />);
    expect(screen.getByText(/docker compose pull && docker compose up -d/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Atualizar Agora/i })).not.toBeInTheDocument();
    expect(screen.queryByText(/entrar em manutenção/i)).not.toBeInTheDocument();
  });
});
