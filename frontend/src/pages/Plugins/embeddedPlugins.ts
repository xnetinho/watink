export interface EmbeddedPlugin {
  slug: string;
  name: string;
  description: string;
}

// Mesma lista do backend (plugins.EmbeddedSlugs). Os slugs são o contrato com a API.
export const EMBEDDED_PLUGINS: EmbeddedPlugin[] = [
  { slug: "helpdesk", name: "uiText.plugins.items.helpdesk.name", description: "uiText.plugins.items.helpdesk.description" },
  { slug: "webchat", name: "uiText.plugins.items.webchat.name", description: "uiText.plugins.items.webchat.description" },
  { slug: "assistant", name: "uiText.plugins.items.assistant.name", description: "uiText.plugins.items.assistant.description" },
  { slug: "groups", name: "uiText.plugins.items.groups.name", description: "uiText.plugins.items.groups.description" },
  { slug: "inventory-advanced", name: "uiText.plugins.items.inventory_advanced.name", description: "uiText.plugins.items.inventory_advanced.description" },
];
