import { i18n } from "../../translate/i18n";
export interface StageFormItem {
    name: string;
    colorKey: string;
}

export interface PipelineFormData {
    name: string;
    description: string;
    type: "kanban" | "funnel";
    stages: StageFormItem[];
}

export interface ChatMessage {
    role: "ai" | "user" | "assistant";
    content: string;
}

export const STAGE_COLOR_PALETTE = [
    { key: "blue",   label: "Azul",     bg: "hsl(var(--status-info-bg))",    color: "hsl(var(--status-info))" },
    { key: "yellow", label: "Amarelo",  bg: "hsl(var(--status-warning-bg))", color: "hsl(var(--status-warning))" },
    { key: "green",  label: "Verde",    bg: "hsl(var(--status-success-bg))", color: "hsl(var(--status-success))" },
    { key: "red",    label: "Vermelho", bg: "hsl(var(--status-error-bg))",   color: "hsl(var(--status-error))" },
    { key: "gray",   label: "Cinza",    bg: "hsl(var(--status-default-bg))", color: "hsl(var(--status-default-text))" },
] as const;

export type StageColorKey = (typeof STAGE_COLOR_PALETTE)[number]["key"];

export function getColorByKey(key: string) {
    return STAGE_COLOR_PALETTE.find((c) => c.key === key) ?? STAGE_COLOR_PALETTE[0];
}

export function getDefaultColorKey(index: number): string {
    return STAGE_COLOR_PALETTE[index % STAGE_COLOR_PALETTE.length].key;
}

// Legacy alias — used in PipelineKanban (by-index color)
export const STAGE_COLORS = STAGE_COLOR_PALETTE.map((c) => ({
    bg: c.bg,
    border: c.color,
    text: c.color,
}));

export const getStageColor = (index: number) => STAGE_COLORS[index % STAGE_COLORS.length];

export const INITIAL_FORM_DATA: PipelineFormData = {
    name: "",
    description: "",
    type: "kanban",
    stages: [
        { name: "Novo", colorKey: "blue" }, // i18n-ignore: etapas iniciais gravadas no pipeline
        { name: "Em Andamento", colorKey: "yellow" }, // i18n-ignore: etapas iniciais gravadas no pipeline
        { name: "Concluído", colorKey: "green" }, // i18n-ignore: etapas iniciais gravadas no pipeline
    ],
};

export const initialChatMessage = (): ChatMessage => ({
    role: "ai",
    content: String(i18n.t("uiText.pipelines.initialChat") ?? ""),
});
