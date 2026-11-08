import { describe, expect, it } from "vitest";
import { callerPlatformKey } from "../platform";

describe("callerPlatformKey", () => {
  it.each(["android", "iphone", "web", "smba", "smbi", "ipad", "darwin", "uwp"])("%s tem tradução", (p) => {
    expect(callerPlatformKey(p)).toBe(`calls.platform.${p}`);
  });
  it("aceita maiúsculas", () => expect(callerPlatformKey("IPHONE")).toBe("calls.platform.iphone"));
  it("ausente, vazio ou desconhecido não mostra nada (nunca exibe texto que veio do aparelho)", () => {
    for (const p of [undefined, "", "windows", "ios", "<script>alert(1)</script>"]) expect(callerPlatformKey(p)).toBeNull();
  });
});
