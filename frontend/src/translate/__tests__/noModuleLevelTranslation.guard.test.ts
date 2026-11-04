import { describe, expect, it } from "vitest";
import ts from "typescript";

// tr()/i18n.t() chamado no NÍVEL DO MÓDULO é avaliado uma vez, no carregamento, e congela o texto no idioma da hora: a
// troca de idioma depois disso não muda nada. Em constantes de módulo guarde a CHAVE ("uiText...") e traduza no uso.
const files = import.meta.glob(
  ["../../**/*.{ts,tsx}", "!../../**/*.test.*", "!../../**/__tests__/**", "!../../translate/**", "!../../**/*.d.ts", "!../../setupTests.ts"],
  { query: "?raw", import: "default", eager: true },
) as Record<string, string>;

const isTranslate = (n: ts.Node): boolean =>
  ts.isCallExpression(n) &&
  ((ts.isIdentifier(n.expression) && n.expression.text === "tr") ||
    (ts.isPropertyAccessExpression(n.expression) && n.expression.expression.getText() === "i18n" && n.expression.name.text === "t"));

describe("tradução fora de função", () => {
  it("nenhum tr()/i18n.t() roda no carregamento do módulo", () => {
    const offenders: string[] = [];
    for (const [file, text] of Object.entries(files)) {
      if (!text.includes("tr(") && !text.includes("i18n.t(")) continue;
      const src = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true, file.endsWith("x") ? ts.ScriptKind.TSX : ts.ScriptKind.TS);
      const visit = (n: ts.Node, inFn: boolean): void => {
        const nowInFn = inFn || ts.isFunctionLike(n);
        if (isTranslate(n) && !nowInFn) {
          const { line } = src.getLineAndCharacterOfPosition(n.getStart());
          offenders.push(`${file.replace(/^(\.\.\/)+/, "")}:${line + 1}  ${n.getText().slice(0, 60)}`);
        }
        ts.forEachChild(n, (c) => visit(c, nowInFn));
      };
      visit(src, false);
    }
    expect(offenders, `tr() em nível de módulo (guarde a chave e traduza no uso):\n${offenders.join("\n")}`).toEqual([]);
  });
});
