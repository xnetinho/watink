import React from "react";
import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import MoveBeforeDeleteModal from "../index";
import { i18n } from "@/translate/i18n";

void i18n.changeLanguage("pt");

const props = {
  open: true,
  title: "Excluir a fila X",
  description: "tem coisas",
  targetLabel: "Mover para a fila",
  targets: [{ id: 2, name: "Destino" }],
  onClose: vi.fn(),
  onConfirm: vi.fn(),
};

describe("MoveBeforeDeleteModal", () => {
  it("não deixa confirmar sem escolher o destino", () => {
    render(<MoveBeforeDeleteModal {...props} />);
    const confirm = screen.getByRole("button", { name: String(i18n.t("moveBeforeDelete.confirm")) });
    expect(confirm).toBeDisabled();
    fireEvent.click(confirm);
    expect(props.onConfirm).not.toHaveBeenCalled();
  });

  it("mostra título e descrição recebidos, sem texto fixo", () => {
    render(<MoveBeforeDeleteModal {...props} />);
    expect(screen.getByText("Excluir a fila X")).toBeInTheDocument();
    expect(screen.getByText("tem coisas")).toBeInTheDocument();
  });
});
