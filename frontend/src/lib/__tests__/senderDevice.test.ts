import { describe, expect, it } from "vitest";
import { senderDeviceFromId } from "../senderDevice";

describe("senderDeviceFromId", () => {
  // ids reais capturados em 7/nov/2026 (Android, iPhone e WhatsApp Web do mesmo teste)
  it.each([
    ["ACF893AD9DC7F9C71FA2C47EA45818AF", "android"],
    ["AC35D39B5CA330A5CD339807F03EEC7E", "android"],
    ["3AF2FA95D11B72616BF0", "ios"],
    ["3A5D81A054B0705B761F", "ios"],
    ["3EB0908DBE993FFAB74472", "web"],
  ])("%s é %s", (id, device) => expect(senderDeviceFromId(id)).toBe(device));

  it("21 caracteres (Android antigo), desktop (3F… ou 18 caracteres)", () => {
    expect(senderDeviceFromId("A".repeat(21))).toBe("android");
    expect(senderDeviceFromId("3FAABBCCDD")).toBe("desktop");
    expect(senderDeviceFromId("B".repeat(18))).toBe("desktop");
  });

  it("a ordem das regras vale: 32 caracteres nunca vira desktop nem web", () => {
    expect(senderDeviceFromId("3E" + "A".repeat(30))).toBe("android");
  });

  it("id fora de qualquer padrão, vazio ou que não é texto não chuta", () => {
    expect(senderDeviceFromId("3EB0F51B5C4C8A0EA663")).toBeNull(); // enviado pelo próprio painel (20 caracteres)
    expect(senderDeviceFromId("")).toBeNull();
    expect(senderDeviceFromId(42)).toBeNull();
    expect(senderDeviceFromId(undefined)).toBeNull();
  });
});
