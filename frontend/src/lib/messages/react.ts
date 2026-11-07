interface Reactable {
  id?: number | string;
  mediaType?: unknown;
  isDeleted?: unknown;
}

/**
 * A mensagem aceita reação? O registro de chamada (mediaType "call") é mensagem de sistema do Watink, não existe no WhatsApp e não tem como
 * receber reação; mensagem apagada também não. O servidor repete a regra (ReactToMessage).
 */
export function canReact(m: Reactable): boolean {
  if (m.id === undefined || m.isDeleted === true) return false;
  return m.mediaType !== "call";
}
