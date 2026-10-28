export type PresencePolicy = "never" | "on_demand" | "always";

export interface PresencePolicyOption {
  value: PresencePolicy;
  label: string;
  /** Resumo curto, mostrado no menu e abaixo do campo. */
  summary: string;
}

// "Disponível" faz o WhatsApp tratar o número como um aparelho ativo (como o WhatsApp Web aberto): o celular do operador
// deixa de receber notificações, o contato vê o número online e o WhatsApp passa a entregar a presença dos contatos.
export const PRESENCE_POLICY_OPTIONS: PresencePolicyOption[] = [
  {
    value: "never",
    label: "Nunca disponível",
    summary:
      "O celular continua recebendo notificações. O contato não vê este número online. O online e a última vez visto dos contatos praticamente não aparecem no Watink.",
  },
  {
    value: "on_demand",
    label: "Disponível durante o atendimento",
    summary:
      "O número fica disponível só enquanto um operador está com uma conversa desta conexão aberta, e volta ao normal depois do tempo ocioso. O celular só fica sem notificações nesse período.",
  },
  {
    value: "always",
    label: "Sempre disponível",
    summary:
      "O número fica sempre disponível, mesmo após reconectar. O online e a última vez visto dos contatos aparecem sempre, mas o celular não recebe notificações enquanto a conexão estiver ativa.",
  },
];

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
