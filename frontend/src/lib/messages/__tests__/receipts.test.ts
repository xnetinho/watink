import { describe, expect, it } from "vitest";
import { canShowReceipts } from "../receipts";

describe("canShowReceipts — quando o item 'Dados da mensagem' aparece", () => {
  it("só em mensagem NOSSA já confirmada pelo canal (ack 1 a 4)", () => {
    for (const ack of [1, 2, 3, 4]) expect(canShowReceipts({ fromMe: true, ack }, "whatsapp")).toBe(true);
  });
  it("não aparece na mensagem do contato, apagada, pendente (0) ou com erro (5)", () => {
    expect(canShowReceipts({ fromMe: false, ack: 3 }, "whatsapp")).toBe(false);
    expect(canShowReceipts({ fromMe: true, ack: 3, isDeleted: true }, "whatsapp")).toBe(false);
    expect(canShowReceipts({ fromMe: true, ack: 0 }, "whatsapp")).toBe(false);
    expect(canShowReceipts({ fromMe: true, ack: 5 }, "whatsapp")).toBe(false);
    expect(canShowReceipts({ fromMe: true }, "whatsapp")).toBe(false);
  });
  it("não aparece em canal sem recibo por mensagem (Telegram); sem canal informado, vale", () => {
    expect(canShowReceipts({ fromMe: true, ack: 3 }, "telegram")).toBe(false);
    expect(canShowReceipts({ fromMe: true, ack: 3 })).toBe(true);
  });
});
