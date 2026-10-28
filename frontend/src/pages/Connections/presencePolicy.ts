export type PresencePolicy = "never" | "on_demand" | "always";

export interface PresencePolicyOption {
  value: PresencePolicy;
  /** Chaves de idioma: connections.presencePolicy.<value>.label e .summary. */
  labelKey: string;
  /** Resumo curto, mostrado no menu e abaixo do campo. */
  summaryKey: string;
}

// "Disponível" faz o WhatsApp tratar o número como um aparelho ativo (como o WhatsApp Web aberto): o celular do operador
// deixa de receber notificações, o contato vê o número online e o WhatsApp passa a entregar a presença dos contatos.
// Os textos de cada política (nome e resumo) ficam nos arquivos de idioma.
export const PRESENCE_POLICY_OPTIONS: PresencePolicyOption[] = (["never", "on_demand", "always"] as const).map((value) => ({
  value,
  labelKey: `connections.presencePolicy.${value}.label`,
  summaryKey: `connections.presencePolicy.${value}.summary`,
}));

export const DEFAULT_PRESENCE_POLICY: PresencePolicy = "never";
export const DEFAULT_IDLE_MINUTES = 5;
export const MAX_IDLE_MINUTES = 240;

export const isPresencePolicy = (v: unknown): v is PresencePolicy => v === "never" || v === "on_demand" || v === "always";

/** Valor do banco/API → política válida; qualquer outra coisa vira o padrão (o mais seguro: não silencia o celular). */
export const toPresencePolicy = (v: unknown): PresencePolicy => (isPresencePolicy(v) ? v : DEFAULT_PRESENCE_POLICY);

export const optionFor = (policy: PresencePolicy): PresencePolicyOption =>
  PRESENCE_POLICY_OPTIONS.find((o) => o.value === policy) ?? PRESENCE_POLICY_OPTIONS[0];

export const clampIdleMinutes = (n: number): number => {
  if (!Number.isFinite(n)) return DEFAULT_IDLE_MINUTES;
  return Math.min(MAX_IDLE_MINUTES, Math.max(1, Math.round(n)));
};
