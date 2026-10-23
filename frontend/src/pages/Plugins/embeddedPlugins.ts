export interface EmbeddedPlugin {
  slug: string;
  name: string;
  description: string;
}

// Mesma lista do backend (plugins.EmbeddedSlugs). Os slugs são o contrato com a API.
export const EMBEDDED_PLUGINS: EmbeddedPlugin[] = [
  { slug: "helpdesk", name: "Helpdesk", description: "Protocolos de atendimento, kanban e acompanhamento público." },
  { slug: "webchat", name: "Webchat", description: "Widget de chat para o seu site, integrado aos tickets." },
  { slug: "assistant", name: "Assistentes de IA", description: "Assistentes com base de conhecimento e automações." },
  { slug: "groups", name: "Grupos e Comunidades", description: "Gestão de grupos, comunidades e campanhas de grupo." },
  { slug: "inventory-advanced", name: "Estoque Avançado", description: "Produtos, SKUs, depósitos e movimentações de estoque." },
];
