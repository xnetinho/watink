export interface ConnectionOption {
  id: number;
  kind: string; // chave do catálogo de canais ("whatsapp", "telegram"...)
  name: string;
  status: string;
}

/** Chave estável para select/lista: o id da conexão como texto. */
export const connectionKey = (c: { id: number }): string => String(c.id);

export const parseConnectionKey = (key: string): number | null => {
  const n = Number(key);
  return Number.isInteger(n) && n > 0 ? n : null;
};

/** Reduz a lista única de /channels/connections ao que os seletores precisam. */
export const toConnectionOptions = (list: ConnectionOption[]): ConnectionOption[] =>
  list.map((c) => ({ id: c.id, kind: c.kind, name: c.name, status: c.status }));
