import { describe, expect, it } from "vitest";
import { canCallBack } from "../canPlace";

describe("canCallBack", () => {
  it.each(["missed", "rejected", "failed", "interrupted"])("chamada recebida %s", (status) => expect(canCallBack({ direction: "incoming", status })).toBe(true));
  it.each(["ended", "active", "ringing"])("chamada recebida %s não", (status) => expect(canCallBack({ direction: "incoming", status })).toBe(false));
  it("chamada nossa nunca (o chamador somos nós)", () => {
    for (const status of ["missed", "rejected", "failed", "interrupted", "ended"]) expect(canCallBack({ direction: "outgoing", status })).toBe(false);
  });
});
