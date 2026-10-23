import { i18n } from "../../translate/i18n";

/** i18n.t() deste projeto pode devolver null; os componentes de chamadas só querem string. */
export function t(key: string): string {
  return String(i18n.t(key) ?? "");
}
