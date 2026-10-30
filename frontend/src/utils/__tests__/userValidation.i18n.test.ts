import { describe, expect, it } from "vitest";
import { i18n } from "../../translate/i18n";
import { UserSchema } from "../userValidation";

// A mensagem do Yup é avaliada na hora da validação: troca de idioma muda o texto sem recarregar a tela.
describe("UserSchema — mensagens seguem o idioma ativo", () => {
  const erros = async () => {
    try {
      await UserSchema.validate({ name: "a", email: "x" }, { abortEarly: false });
      return [] as string[];
    } catch (e) {
      return (e as { errors: string[] }).errors;
    }
  };

  it("pt, en e es", async () => {
    await i18n.changeLanguage("pt");
    expect(await erros()).toContain("Muito curto!");
    await i18n.changeLanguage("en");
    expect(await erros()).toContain("Too short!");
    await i18n.changeLanguage("es");
    expect(await erros()).toContain("¡Muy corto!");
    await i18n.changeLanguage("pt");
  });
});
