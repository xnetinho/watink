import api from "../../services/api";

export interface MessageEditVersion {
  id: number;
  messageId: string;
  /** 1 = o texto original; cada edição acrescenta a seguinte. */
  version: number;
  body: string;
  editedAt: string;
  /** "me" (o operador editou pelo Watink) ou "contact" (o contato editou). */
  editedBy: "me" | "contact";
}

/** Busca a cadeia completa de uma mensagem editada, da versão 1 (original) à mais recente. */
export function fetchMessageEdits(messageId: string | number): Promise<MessageEditVersion[]> {
  // Encadeia direto na promessa do axios (sem `async`/`await` no meio): quem chama anexa o catch na mesma volta do loop.
  return api.get<{ edits: MessageEditVersion[] }>(`/message/${messageId}/edits`).then(({ data }) => data.edits ?? []);
}
