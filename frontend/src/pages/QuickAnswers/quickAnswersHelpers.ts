import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");

import type {
  QuickAnswerType,
  QuickAnswerContentButtons,
  QuickAnswerContentList,
  QuickAnswerContentMedia,
  QuickAnswerContentPoll,
  QuickAnswerContentCarousel,
  QuickAnswerCarouselButtonType,
  QuickAnswerContentPix,
  PixKeyType,
} from "./quickAnswersTypes";

// ─── helpers ────────────────────────────────────────────────────────────────

export function genId(prefix: string, index: number): string {
  return `${prefix}_${Date.now()}_${index}`;
}

export const PIX_KEY_TYPES: { value: PixKeyType; label: string }[] = [
  { value: "EVP", label: "uiText.quick.pix.evp" },
  { value: "CPF", label: "CPF" },
  { value: "CNPJ", label: "CNPJ" },
  { value: "PHONE", label: "uiText.quick.pix.phone" },
  { value: "EMAIL", label: "uiText.quick.pix.email" }, // i18n-ignore: valor técnico enviado à API
];

export const defaultPix = (): QuickAnswerContentPix => ({
  body: "",
  pixKey: "",
  pixType: "EVP",
  pixName: "",
});

export const defaultButtons = (): QuickAnswerContentButtons => ({
  body: "",
  footer: "",
  buttons: [{ id: genId("btn", 0), label: "" }],
});

export const defaultList = (): QuickAnswerContentList => ({
  body: "",
  button_text: "",
  footer: "",
  sections: [
    { title: "", rows: [{ id: genId("row", 0), title: "", description: "" }] },
  ],
});

export const defaultMedia = (): QuickAnswerContentMedia => ({
  media_type: "image",
  url: "",
  caption: "",
});

export const defaultPoll = (): QuickAnswerContentPoll => ({
  question: "",
  options: ["", ""],
  max_selections: 1,
  capture_results: false,
  on_answer: null,
});

export const defaultCarousel = (): QuickAnswerContentCarousel => ({
  body: "",
  cards: [
    {
      image: "",
      title: "",
      footer: "",
      buttons: [{ id: genId("cbtn", 0), label: "", type: "quickreply" }],
    },
  ],
});

// ─── type config ─────────────────────────────────────────────────────────────

export const TYPE_OPTIONS: { value: QuickAnswerType; label: string; description: string }[] = [
  { value: "text", label: "uiText.quick.type.text.label", description: "uiText.quick.type.text.description" },
  { value: "interactive_buttons", label: "uiText.quick.type.interactive_buttons.label", description: "uiText.quick.type.interactive_buttons.description" },
  { value: "list", label: "uiText.quick.type.list.label", description: "uiText.quick.type.list.description" },
  { value: "media", label: "uiText.quick.type.media.label", description: "uiText.quick.type.media.description" },
  { value: "poll", label: "uiText.quick.type.poll.label", description: "uiText.quick.type.poll.description" },
  { value: "carousel", label: "uiText.quick.type.carousel.label", description: "uiText.quick.type.carousel.description" },
  { value: "pix", label: "uiText.quick.type.pix.label", description: "uiText.quick.type.pix.description" },
];

export const CAROUSEL_BUTTON_TYPES: { value: QuickAnswerCarouselButtonType; label: string }[] = [
  { value: "quickreply", label: "uiText.quick.carouselBtn.quickreply" },
  { value: "url", label: "uiText.quick.carouselBtn.url" },
  { value: "call", label: "uiText.quick.carouselBtn.call" },
  { value: "copy", label: "uiText.quick.carouselBtn.copy" },
];

// ─── validation ───────────────────────────────────────────────────────────────

export interface FormErrors {
  shortcut?: string;
  body?: string;
  question?: string;
  url?: string;
  buttons?: string;
  options?: string;
  [key: string]: string | undefined;
}

export function validate(
  shortcut: string,
  type: QuickAnswerType,
  textBody: string,
  buttons: QuickAnswerContentButtons,
  list: QuickAnswerContentList,
  media: QuickAnswerContentMedia,
  poll: QuickAnswerContentPoll,
  carousel: QuickAnswerContentCarousel,
  pix: QuickAnswerContentPix
): FormErrors {
  const errors: FormErrors = {};
  if (!shortcut || shortcut.length < 2) errors.shortcut = tr("uiText.quick.validation.shortcutMin");
  if (type === "text" && !textBody.trim()) errors.body = tr("uiText.quick.validation.bodyRequired");
  if (type === "interactive_buttons") {
    if (!buttons.body.trim()) errors.body = tr("uiText.quick.validation.bodyRequired");
    if (buttons.buttons.length < 1) errors.buttons = tr("uiText.quick.validation.minButton");
  }
  if (type === "list" && !list.body.trim()) errors.body = tr("uiText.quick.validation.bodyRequired");
  if (type === "media" && !media.url.trim()) errors.url = tr("uiText.quick.validation.urlRequired");
  if (type === "poll") {
    if (!poll.question.trim()) errors.question = tr("uiText.quick.validation.questionRequired");
    if (poll.options.length < 2) errors.options = tr("uiText.quick.validation.minOptions");
  }
  if (type === "carousel") {
    if (carousel.cards.length < 1) errors.cards = tr("uiText.quick.validation.minCard");
    else if (carousel.cards.some((c) => !c.image.trim()))
      errors.cards = tr("uiText.quick.validation.cardImage");
    else if (carousel.cards.some((c) => !c.title.trim()))
      errors.cards = tr("uiText.quick.validation.cardText");
  }
  if (type === "pix" && !pix.pixKey.trim()) errors.pixKey = tr("uiText.quick.validation.pixRequired");
  return errors;
}
