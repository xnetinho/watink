// Valor que o backend devolve no lugar de uma setting secreta (chave de API,
// token, senha) para quem não tem settings:update. Precisa ser idêntico ao
// maskedSecretPlaceholder de business/internal/controllers/setting_secrets.go.
export const MASKED_SECRET = "••••••••";

export const isMaskedSecret = (value: string): boolean => value === MASKED_SECRET;

// Decide se um campo secreto deve ser gravado ao sair do foco. Nunca grava o
// placeholder: sem esta guarda, abrir o campo e sair gravaria "••••••••" por
// cima da chave real de quem tem o valor mascarado na tela.
export const shouldSaveSecret = (typed: string, current: string): boolean =>
  !isMaskedSecret(typed) && typed !== current;
