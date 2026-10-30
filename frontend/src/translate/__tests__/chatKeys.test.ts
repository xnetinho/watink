import { describe, expect, it } from "vitest";
import { messages } from "../languages";

type Tree = { [k: string]: string | Tree };
const leaves = (n: Tree, p = ""): string[] => Object.entries(n).flatMap(([k, v]) => (typeof v === "string" ? [`${p}${k}`] : leaves(v, `${p}${k}.`)));
const values = (n: Tree): string[] => Object.values(n).flatMap((v) => (typeof v === "string" ? [v] : values(v)));
const block = (lang: "pt" | "en" | "es", name: string) => (messages[lang].translations as unknown as Record<string, Tree>)[name];

// Blocos que os cartões do chat, o selo de encaminhada, a presença e a política de presença usam.
const SUBTREES: Array<[string, string[]]> = [
  ["chat", ["pix", "event", "poll", "liveLocation", "forwarded", "forwardedMany", "previews", "presence", "unsupported", "viewOnce"]],
  ["connections", ["presencePolicy"]],
  ["uiText", ["connections", "validation", "whatsAppModal", "common", "assistants", "dashboard", "flow", "settings", "clients", "activities", "helpdesk", "acessos", "swagger", "billing", "plans", "plugins", "webchatModal", "themeToggle", "priority", "imageCompression", "channels", "layout", "calls", "fileUploader", "messageInput", "messagesList", "notificationPermissionBanner", "notificationsPopOver", "queueModal", "tagModal", "ticketHeader", "ticketQuickShortcuts", "ticketsManager", "whatsAppBubblePreview", "ui", "hooks", "errors", "utils", "mainTopBar", "login", "queues", "tagManager", "communities", "monitorQueues", "initialSetup", "userProfile"]],
];
const pick = (t: Tree, keys: string[]): Tree => Object.fromEntries(keys.filter((k) => k in t).map((k) => [k, t[k]]));

describe("traduções de chat e presença", () => {
  for (const [name, keys] of SUBTREES) {
    const pt = leaves(pick(block("pt", name), keys)).sort();
    it(`${name}: as 3 línguas têm exatamente as mesmas chaves`, () => {
      expect(pt.length).toBeGreaterThan(0);
      expect(leaves(pick(block("en", name), keys)).sort()).toEqual(pt);
      expect(leaves(pick(block("es", name), keys)).sort()).toEqual(pt);
    });
    it(`${name}: nenhum texto é vazio`, () => {
      for (const l of ["pt", "en", "es"] as const) values(pick(block(l, name), keys)).forEach((v) => expect(v.trim().length).toBeGreaterThan(0));
    });
  }

  it("en e es não são cópia do pt (tradução de verdade)", () => {
    const flat = (l: "pt" | "en" | "es") => JSON.stringify(pick(block(l, "connections"), ["presencePolicy"]));
    expect(flat("en")).not.toBe(flat("pt"));
    expect(flat("es")).not.toBe(flat("pt"));
  });

  it("variáveis {{...}} são as mesmas em todas as línguas", () => {
    const vars = (s: string) => [...s.matchAll(/\{\{(\w+)\}\}/g)].map((m) => m[1]).sort().join(",");
    const flat = (l: "pt" | "en" | "es", name: string, keys: string[]): Record<string, string> => {
      const out: Record<string, string> = {};
      const walk = (n: Tree, p: string) => Object.entries(n).forEach(([k, v]) => (typeof v === "string" ? (out[`${p}${k}`] = v) : walk(v, `${p}${k}.`)));
      walk(pick(block(l, name), keys), "");
      return out;
    };
    for (const [name, keys] of SUBTREES) {
      const pt = flat("pt", name, keys);
      for (const l of ["en", "es"] as const) {
        const o = flat(l, name, keys);
        for (const k of Object.keys(pt)) expect(vars(o[k] ?? ""), `${l}.${name}.${k}`).toBe(vars(pt[k]));
      }
    }
  });
});
