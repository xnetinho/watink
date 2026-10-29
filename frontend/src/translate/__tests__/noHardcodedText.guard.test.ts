import { describe, expect, it } from "vitest";

// Nenhum texto visível ao usuário pode ficar fixo no código: tudo vem dos arquivos de idioma (translate/languages).
// A guarda cobre os arquivos de UI criados/alterados nesta frente (cartões do chat, presença, política de presença) e
// pode crescer: arquivo novo de UI entra na lista.
const files = import.meta.glob(
  [
    "../../components/MessagesList/components/RichCards.tsx",
    "../../components/MessagesList/components/ForwardedLabel.tsx",
    "../../components/MessagesList/utils/messageHelpers.ts",
    "../../components/TicketInfo/index.tsx",
    "../../lib/presence/format.ts",
    "../../hooks/useContactPresence.ts",
    "../../pages/Connections/components/PresencePolicyCard.tsx",
    "../../pages/Connections/presencePolicy.ts",
    "../../pages/Connections/hooks/useConnectionActions.ts",
    "../../components/TicketConnectionBadge.tsx",
    "../../pages/Tickets/components/DetailsPanel.tsx",
    "../../pages/Tickets/components/DadosTab.tsx",
    "../../pages/Tickets/components/PipelinesSection.tsx",
    "../../pages/Tickets/components/FlowsSection.tsx",
    "../../pages/Tickets/components/TicketTagsSection.tsx",
    "../../pages/Tickets/components/HelpdeskSection.tsx",
    "../../components/TicketHistory/index.tsx",
    "../../components/TicketListItem/index.tsx",
    "../../lib/tickets/lastMessage.ts",
    "../../components/ForwardMessageDialog.tsx",
    "../../components/MessageInput/DropZone.tsx",
    "../../components/MessageInput/components/MediaPreviewPanel.tsx",
    "../../components/Audio/index.tsx",
    "../../components/AudioKindPicker.tsx",
    "../../lib/audio/waveform.ts",
    "../../components/ContactModal/index.tsx",
    "../../pages/Contacts/components/ContactConnectionCell.tsx",
    "../../pages/Connections/components/ConnectionActionButtons.tsx",
    "../../lib/connections/connectionRef.ts",
    "../../hooks/useConnectionOptions.ts",
    "../../components/MessageOptionsMenu/index.tsx",
    "../../lib/messages/forward.ts",
  ],
  { query: "?raw", import: "default", eager: true },
) as Record<string, string>;

// Texto de interface em português ou inglês de uma palavra solta: acento, ou frase com artigo/preposição comum.
const PT = /[ãõáéíóúâêôçÃÕÁÉÍÓÚÂÊÔÇ]|\b(de|do|da|em|para|com|sem|que|não|um|uma|ou|ao|na|no)\b/;

const strip = (line: string) => line.replace(/\/\/.*$/, "").replace(/\/\*.*?\*\//g, "");

describe("texto fixo na interface", () => {
  it("há arquivos para checar (senão a guarda passaria vazia)", () => {
    expect(Object.keys(files).length).toBeGreaterThanOrEqual(19);
  });

  it("nenhum literal de interface em português fora dos arquivos de idioma", () => {
    const offenders: string[] = [];
    for (const [file, text] of Object.entries(files)) {
      let inBlockComment = false;
      text.split("\n").forEach((raw, i) => {
        const line = strip(raw);
        if (raw.trim().startsWith("/*") || raw.trim().startsWith("/**")) inBlockComment = !raw.includes("*/");
        else if (inBlockComment) {
          if (raw.includes("*/")) inBlockComment = false;
          return;
        }
        if (/^\s*\*/.test(raw) || /^\s*import\b/.test(line)) return;
        // literais entre aspas/crases e texto de JSX entre > <
        const literals = [...line.matchAll(/"([^"\n]+)"|'([^'\n]+)'|`([^`\n]+)`|>([^<>{}\n]+)</g)].map((m) => (m[1] ?? m[2] ?? m[3] ?? m[4] ?? "").trim());
        for (const lit of literals) {
          if (!lit || lit.length < 3) continue;
          if (/^(https?:|hsl\(|var\(|\/|\.|#|[a-z]+-[a-z0-9-]+|[a-zA-Z.]+\.[a-zA-Z.]+$)/.test(lit)) continue; // url, classe, chave i18n, caminho
          if (/^[\w.:${}/-]+$/.test(lit)) continue; // identificadores, rotas, chaves
          if (PT.test(lit)) offenders.push(`${file.replace(/^(\.\.\/)+/, "")}:${i + 1}  ${JSON.stringify(lit.slice(0, 60))}`);
        }
      });
    }
    expect(offenders, `texto fixo (use os arquivos de idioma):\n${offenders.join("\n")}`).toEqual([]);
  });
});
