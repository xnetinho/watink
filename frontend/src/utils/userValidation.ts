import * as Yup from "yup";

import { i18n } from "../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
export const UserSchema = Yup.object().shape({
  name: Yup.string()
    .min(2, () => tr("uiText.validation.tooShort"))
    .max(50, () => tr("uiText.validation.tooLong"))
    .required(() => tr("uiText.validation.required")),
  password: Yup.string().min(5, () => tr("uiText.validation.tooShort")).max(50, () => tr("uiText.validation.tooLong")),
  email: Yup.string().email(() => tr("uiText.utils.invalidEmail")).required(() => tr("uiText.validation.required")),
});

export const UserProfileSchema = Yup.object().shape({
  name: Yup.string()
    .min(2, "Too Short!")
    .max(50, "Too Long!")
    .required("Required"),
  email: Yup.string().email("Invalid email").required("Required"),
  password: Yup.string().min(5, "Too Short!").max(50, "Too Long!"),
  confirmPassword: Yup.string().oneOf(
    [Yup.ref("password")],
    "Passwords must match"
  ),
});
