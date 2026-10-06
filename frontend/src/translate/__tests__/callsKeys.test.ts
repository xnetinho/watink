import { describe, expect, it } from "vitest";
import { messages } from "../languages";

type Tree = { [k: string]: string | Tree };

function leaves(node: Tree, prefix = ""): string[] {
  return Object.entries(node).flatMap(([k, v]) =>
    typeof v === "string" ? [`${prefix}${k}`] : leaves(v, `${prefix}${k}.`),
  );
}

function values(node: Tree): string[] {
  return Object.values(node).flatMap((v) => (typeof v === "string" ? [v] : values(v)));
}

const calls = (lang: "pt" | "en" | "es") =>
  (messages[lang].translations as unknown as { calls: Tree }).calls;

describe("traduções de chamadas", () => {
  const pt = leaves(calls("pt")).sort();

  it("existe um bloco `calls` nas três línguas", () => {
    for (const l of ["pt", "en", "es"] as const) expect(calls(l)).toBeDefined();
  });

  it("as três línguas têm exatamente as mesmas chaves", () => {
    expect(leaves(calls("en")).sort()).toEqual(pt);
    expect(leaves(calls("es")).sort()).toEqual(pt);
  });

  it("nenhum texto é vazio", () => {
    for (const l of ["pt", "en", "es"] as const) {
      values(calls(l)).forEach((v) => expect(v.trim().length).toBeGreaterThan(0));
    }
  });

  it("os motivos de fim e as situações cobrem todos os valores que o backend envia", () => {
    const reasons = ["user_ended", "declined", "timeout", "busy", "cancelled", "failed", "no_operator", "proxy_blocked", "unsupported_type", "accepted_elsewhere", "interrupted"];
    const statuses = ["ringing", "active", "ended", "missed", "rejected", "failed", "interrupted"];
    for (const l of ["pt", "en", "es"] as const) {
      const c = calls(l) as unknown as { endReason: Tree; status: Tree };
      reasons.forEach((r) => expect(c.endReason[r], `${l}.endReason.${r}`).toBeTruthy());
      statuses.forEach((s) => expect(c.status[s], `${l}.status.${s}`).toBeTruthy());
    }
  });

  it("o texto de responsabilidade existe e é diferente em cada língua", () => {
    const t = (l: "pt" | "en" | "es") => ((calls(l) as unknown as { settings: Tree }).settings.ackText as string);
    expect(new Set([t("pt"), t("en"), t("es")]).size).toBe(3);
  });

  it("os textos de limite de medição e de 'estimado' estão presentes", () => {
    for (const l of ["pt", "en", "es"] as const) {
      const q = (calls(l) as unknown as { quality: Tree }).quality;
      expect(q.lossNote).toBeTruthy();
      expect(q.indexNote).toBeTruthy();
      expect(q.estimated).toBeTruthy();
    }
  });
});
