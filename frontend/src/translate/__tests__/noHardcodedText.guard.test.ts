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
    "../../components/EditMessageDialog.tsx",
    "../../components/MessagesList/components/EditedBadge.tsx",
    "../../components/MessagesList/components/ViewOnceBadge.tsx",
    "../../components/MessagesList/components/MessageEditsDialog.tsx",
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

// Varredura de src/ inteiro (tsx/ts de UI). Arquivos ainda NÃO migrados ficam em DEBT: a lista só pode encolher; arquivo novo
// com texto fixo reprova. Ao migrar um arquivo, tire-o de DEBT (o teste "DEBT só tem arquivos que ainda têm texto fixo" cobra).
const all = import.meta.glob(
  ["../../**/*.{ts,tsx}", "!../../**/*.test.*", "!../../**/__tests__/**", "!../../translate/**", "!../../**/*.d.ts", "!../../setupTests.ts"],
  { query: "?raw", import: "default", eager: true },
) as Record<string, string>;
const DEBT = new Set<string>([
  "pages/Connections/components/ChannelSelectorModal.tsx",
  "pages/Connections/components/ConnectionActionGrid.tsx",
  "pages/Connections/components/ConnectionCard.tsx",
  "pages/Connections/components/ConnectionHealthBadge.tsx",
  "pages/Connections/components/ConnectionRiskBanner.tsx",
  "pages/Connections/components/ConnectionStatusBanner.tsx",
  "pages/Connections/components/IdentityCard.tsx",
  "pages/Connections/components/PairingCodePanel.tsx",
  "pages/Connections/components/PhoneInputDialog.tsx",
  "pages/Connections/components/QrCodePanel.tsx",
  "pages/Connections/components/SessionDetailsCard.tsx",
  "pages/Connections/connectionRisk.ts",
  "pages/Connections/index.tsx",
  "pages/Contacts/components/ContactActionsMenu.tsx",
  "pages/Contacts/components/ContactStatusBadge.tsx",
  "pages/FlowManager/components/FlowCard.tsx",
  "pages/FlowManager/components/FlowFormDialog.tsx",
  "pages/FlowManager/components/FlowGrid.tsx",
  "pages/FlowManager/components/FlowToolbar.tsx",
  "pages/FlowManager/hooks/useFlowManager.ts",
  "pages/FlowManager/index.tsx",
  "pages/GroupCampaigns/GroupCampaignEditor.tsx",
  "pages/GroupCampaigns/GroupCampaignReport.tsx",
  "pages/GroupCampaigns/campaignHelpers.ts",
  "pages/GroupCampaigns/components/CampaignGroupPicker.tsx",
  "pages/GroupCampaigns/components/CampaignPacingForm.tsx",
  "pages/GroupCampaigns/components/CampaignRiskWarning.tsx",
  "pages/GroupCampaigns/components/CampaignScheduleForm.tsx",
  "pages/GroupCampaigns/components/CampaignVariantsEditor.tsx",
  "pages/Groups/CreateGroupDialog.tsx",
  "pages/Groups/GroupDetail.tsx",
  "pages/Groups/GroupInviteCard.tsx",
  "pages/Groups/GroupParticipantsPanel.tsx",
  "pages/Groups/GroupSettingsPanel.tsx",
  "pages/Groups/GroupsErrorState.tsx",
  "pages/Groups/JoinRequestsPanel.tsx",
  "pages/Groups/groupTypes.ts",
  "pages/GroupsWhatsapp/components/CampanhasTab.tsx",
  "pages/GroupsWhatsapp/components/ComunidadesTab.tsx",
  "pages/GroupsWhatsapp/components/GruposTab.tsx",
  "pages/GroupsWhatsapp/components/MonitoramentoTab.tsx",
  "pages/GroupsWhatsapp/index.tsx",
  "pages/KnowledgeBase/KnowledgeBaseConfig.tsx",
  "pages/KnowledgeBase/RetrievalPlayground.tsx",
  "pages/KnowledgeBase/components/AddSourceDialog.tsx",
  "pages/KnowledgeBase/components/KBSourceBadges.tsx",
  "pages/KnowledgeBase/components/KBSourceListItem.tsx",
  "pages/KnowledgeBase/index.tsx",
  "pages/MonitorQueues/index.tsx",
  "pages/Pipelines/PipelineBoard.tsx",
  "pages/Pipelines/PipelineCreator.tsx",
  "pages/Pipelines/PipelineFunnelChart.tsx",
  "pages/Pipelines/PipelineGantt.tsx",
  "pages/Pipelines/PipelineKanban.tsx",
  "pages/Pipelines/components/PipelineAIChat.tsx",
  "pages/Pipelines/components/PipelineFormFields.tsx",
  "pages/Pipelines/components/PipelinePreview.tsx",
  "pages/Pipelines/components/PipelineStageList.tsx",
  "pages/Pipelines/hooks/usePipelineCreator.ts",
  "pages/Pipelines/index.tsx",
  "pages/Pipelines/pipelineCreatorTypes.ts",
  "pages/Proxies/ConnectionGroupsDialog.tsx",
  "pages/Proxies/ProxyFormDialog.tsx",
  "pages/Proxies/ProxyGroupsPanel.tsx",
  "pages/Proxies/ProxyImportDialog.tsx",
  "pages/QuickAnswers/QuickAnswerEditor.tsx",
  "pages/QuickAnswers/components/QuickAnswersTable.tsx",
  "pages/QuickAnswers/components/QuickAnswersToolbar.tsx",
  "pages/QuickAnswers/editors/ButtonsEditor.tsx",
  "pages/QuickAnswers/editors/CarouselEditor.tsx",
  "pages/QuickAnswers/editors/FormatToolbar.tsx",
  "pages/QuickAnswers/editors/ListEditor.tsx",
  "pages/QuickAnswers/editors/MediaEditor.tsx",
  "pages/QuickAnswers/editors/PhoneMockup.tsx",
  "pages/QuickAnswers/editors/PixEditor.tsx",
  "pages/QuickAnswers/editors/PollEditor.tsx",
  "pages/QuickAnswers/editors/TextEditor.tsx",
  "pages/QuickAnswers/index.tsx",
  "pages/QuickAnswers/quickAnswersHelpers.ts",
  "pages/QuickAnswers/quickAnswersTypes.ts",
  "pages/Register/index.tsx",
  "pages/SaaS/components/PlanModal.tsx",
  "pages/SaaS/components/PlansTab.tsx",
  "pages/SaaS/components/TenantModal.tsx",
  "pages/SaaS/components/TenantsTab.tsx",
  "pages/SaaS/hooks/useSaaS.ts",
  "pages/SaaS/index.tsx",
  "pages/UserProfile/components/ProfileAvatarCard.tsx",
  "pages/VersionDashboard/components/DashboardHeader.tsx",
  "pages/VersionDashboard/components/ProcessCard.tsx",
  "pages/VersionDashboard/components/RabbitMQCard.tsx",
  "pages/VersionDashboard/components/SystemStatsGrid.tsx",
  "pages/VersionDashboard/components/TenantConsumptionTable.tsx",
  "pages/VersionDashboard/components/UpdateDialog.tsx",
  "pages/VersionDashboard/hooks/useVersionDashboard.ts",
  "pages/VersionDashboard/index.tsx",
  "pages/VersionDashboard/utils.ts",
].map((d) => `../../${d}`));

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
    expect(Object.keys(files).length).toBeGreaterThanOrEqual(19);
    expect(Object.keys(all).length).toBeGreaterThan(200);
  });

  it("os arquivos já migrados (lista explícita) não têm texto fixo", () => {
    const offenders = Object.entries(files).flatMap(([file, text]) => hardcodedLines(text).map((l) => `${file.replace(/^(\.\.\/)+/, "")}:${l}`));
    expect(offenders, `texto fixo (use os arquivos de idioma):\n${offenders.join("\n")}`).toEqual([]);
  });

  it("nenhum arquivo de UI fora de DEBT tem texto fixo em português", () => {
    const offenders = Object.entries(all)
      .filter(([file]) => !DEBT.has(file))
      .flatMap(([file, text]) => hardcodedLines(text).map((l) => `${file.replace(/^(\.\.\/)+/, "")}:${l}`));
    expect(offenders, `texto fixo (use os arquivos de idioma ou, se for dívida antiga, não adicione arquivo novo a DEBT):\n${offenders.join("\n")}`).toEqual([]);
  });

  it("DEBT só tem arquivos que ainda têm texto fixo (migrou? tire da lista)", () => {
    const limpos = [...DEBT].filter((f) => all[f] !== undefined && hardcodedLines(all[f]).length === 0).map((f) => f.replace(/^(\.\.\/)+/, ""));
    const sumidos = [...DEBT].filter((f) => all[f] === undefined).map((f) => f.replace(/^(\.\.\/)+/, ""));
    expect(limpos, `já sem texto fixo, tire de DEBT:\n${limpos.join("\n")}`).toEqual([]);
    expect(sumidos, `não existe mais, tire de DEBT:\n${sumidos.join("\n")}`).toEqual([]);
  });
});
