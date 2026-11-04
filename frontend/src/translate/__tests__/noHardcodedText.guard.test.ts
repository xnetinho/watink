import { describe, expect, it } from "vitest";

// Nenhum texto visível ao usuário pode ficar fixo no código: tudo vem dos arquivos de idioma (translate/languages).
// A varredura cobre src/ inteiro (tsx/ts de UI). Valor que NÃO é texto de tela (gravado no banco, enviado à API, nome fictício)
// leva o marcador "i18n-ignore" na própria linha, com a justificativa.
const all = import.meta.glob(
  ["../../**/*.{ts,tsx}", "!../../**/*.test.*", "!../../**/__tests__/**", "!../../translate/**", "!../../**/*.d.ts", "!../../setupTests.ts"],
  { query: "?raw", import: "default", eager: true },
) as Record<string, string>;

const ACC = /[ãõáéíóúâêôçÃÕÁÉÍÓÚÂÊÔÇ]/;
const STOP = /\b(de|do|da|em|para|com|sem|que|não|um|uma|ou|ao|na|no|seu|sua|nova|novo|ainda|este|esta|erro|salvar|cancelar|excluir)\b/i;
const SINGLE = new Set("salvar cancelar excluir editar nome buscar fechar voltar adicionar remover criar ativo inativo baixa alta média urgente normal sim não tipo ações todos todas nenhum nenhuma selecione digite carregando enviar aplicar limpar filtrar mais menos título descrição usuário usuários status prioridade ativar desativar confirmar continuar próximo anterior pesquisar atualizar copiar abrir upload sucesso aviso opcional obrigatório padrão geral avançado conexão conexões fila filas etiqueta etiquetas contato contatos mensagem mensagens ticket tickets atendente atendentes subtítulo prévia personalização integração botão contato descrição ocorrências início decisão automação automático você configurações vídeo áudio média botões mídia itálico monoespaçado obrigatório tenant usuários contatos abertos whatsapps rotação fixo cidade bairro endereço endereços número complemento documento empresa cargo pendente aberto fechado cancelado rascunho atualizado ativa inativa hoje ontem categoria telefone email".split(" "));
const isPT = (x: string): boolean => ACC.test(x) || (x.includes(" ") && STOP.test(x)) || SINGLE.has(x.toLowerCase());
const SKIP_LIT = /^(https?:|hsl\(|var\(|\/|\.|#|[a-z]+-[a-z0-9-]+|[a-zA-Z.]+\.[a-zA-Z.]+$)/;
const SKIP_ID = /^[\w.:${}/-]+$/;
const JSX_STOP_LINE = /^(export|const|let|return|if|for|case|type|interface|function)\b/;

/** Linhas com texto de interface em português (literal entre aspas/crases, texto de JSX ou texto solto em JSX). */
export function hardcodedLines(text: string): string[] {
  const out: string[] = [];
  let blk = false;
  text.split("\n").forEach((raw, i) => {
    const t = raw.trim();
    if (t.startsWith("/*")) { blk = !raw.includes("*/"); return; }
    if (blk) { if (raw.includes("*/")) blk = false; return; }
    if (t.startsWith("*") || t.startsWith("import") || t.startsWith("//") || raw.includes("console.") || raw.includes("i18n-ignore")) return;
    if (raw.includes("{/*") && !raw.includes("*/}")) { blk = true; return; }
    const line = raw.replace(/\/\/.*$/, "").replace(/\/\*.*?\*\//g, "");
    let hit: string | null = null;
    for (const m of line.matchAll(/"([^"\n]+)"|'([^'\n]+)'|`([^`\n]+)`|>([^<>{}\n]+)</g)) {
      const lit = (m[1] ?? m[2] ?? m[3] ?? m[4] ?? "").trim();
      if (lit.length < 3 || SKIP_LIT.test(lit) || (SKIP_ID.test(lit) && !(/^[A-ZÀ-Ú]/.test(lit) && SINGLE.has(lit.toLowerCase())))) continue;
      if (isPT(lit)) { hit = lit; break; }
    }
    const t2 = line.trim();
    if (!hit && t2 && !"<{}()/*@.&|?:+-=;,'\"`".includes(t2[0]) && !JSX_STOP_LINE.test(t2) && !/[=;{}()<>:]/.test(t2) && /[A-Za-zÀ-ú]{3,}/.test(t2) && isPT(t2)) hit = t2;
    if (hit) out.push(`${i + 1}  ${JSON.stringify(hit.slice(0, 60))}`);
  });
  return out;
}

describe("texto fixo na interface", () => {
  it("há arquivos para checar (senão a guarda passaria vazia)", () => {
    expect(Object.keys(all).length).toBeGreaterThan(200);
  });

  it("nenhum arquivo de UI tem texto fixo em português", () => {
    const offenders = Object.entries(all).flatMap(([file, text]) => hardcodedLines(text).map((l) => `${file.replace(/^(\.\.\/)+/, "")}:${l}`));
    expect(offenders, `texto fixo (use os arquivos de idioma; valor que não é texto de tela leva // i18n-ignore na linha):\n${offenders.join("\n")}`).toEqual([]);
  });
});
