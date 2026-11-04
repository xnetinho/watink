/**
 * O campo `persona` salvo no backend continua sendo um único texto livre
 * (prompt de sistema) — nenhuma mudança de schema. Este módulo só organiza a
 * EDIÇÃO desse texto em 5 seções guiadas (framework de design de persona de
 * IA), compondo/decompondo por marcadores de cabeçalho reconhecíveis. Persona
 * salva antes desta feature (texto livre sem os marcadores) cai inteira na
 * seção "legacy" e continua editável sem perda de conteúdo.
 */

export interface PersonaSectionDef {
    key: string;
    marker: string;
    title: string;
    subtitle: string;
    placeholder: string;
}

// Os `marker` ficam em português DE PROPÓSITO: são gravados no texto da persona (banco) e usados para decompor/compor
// as seções; traduzi-los quebraria as personas já salvas. Só title/subtitle/placeholder são texto de tela.
export const PERSONA_SECTIONS: PersonaSectionDef[] = [
    {
        key: "identity",
        marker: "## Identidade e Propósito",
        title: "uiText.assistants.persona.identity.title",
        subtitle: "uiText.assistants.persona.identity.subtitle",
        placeholder: "uiText.assistants.persona.identity.placeholder",
    },
    {
        key: "personality",
        marker: "## Traços de Personalidade",
        title: "uiText.assistants.persona.personality.title",
        subtitle: "uiText.assistants.persona.personality.subtitle",
        placeholder: "uiText.assistants.persona.personality.placeholder",
    },
    {
        key: "tone",
        marker: "## Tom e Voz",
        title: "uiText.assistants.persona.tone.title",
        subtitle: "uiText.assistants.persona.tone.subtitle",
        placeholder: "uiText.assistants.persona.tone.placeholder",
    },
    {
        key: "boundaries",
        marker: "## Domínio de Conhecimento e Limitações",
        title: "uiText.assistants.persona.boundaries.title",
        subtitle: "uiText.assistants.persona.boundaries.subtitle",
        placeholder: "uiText.assistants.persona.boundaries.placeholder",
    },
    {
        key: "relationship",
        marker: "## Dinâmica de Relacionamento",
        title: "uiText.assistants.persona.relationship.title",
        subtitle: "uiText.assistants.persona.relationship.subtitle",
        placeholder: "uiText.assistants.persona.relationship.placeholder",
    },
];

export type PersonaSectionValues = Record<string, string> & { legacy: string };

const emptySectionValues = (): PersonaSectionValues => {
    const values = { legacy: "" } as PersonaSectionValues;
    PERSONA_SECTIONS.forEach((s) => {
        values[s.key] = "";
    });
    return values;
};

/** Decompõe o texto salvo em seções, pelos marcadores `## Título`. Texto sem
 * nenhum marcador reconhecido (persona legada, pré-feature) vai inteiro para
 * `legacy`. */
export function parsePersonaSections(raw: string): PersonaSectionValues {
    const values = emptySectionValues();
    if (!raw || !raw.trim()) return values;

    const markerPositions = PERSONA_SECTIONS.map((s) => ({
        key: s.key,
        index: raw.indexOf(s.marker),
    })).filter((m) => m.index !== -1);

    if (markerPositions.length === 0) {
        values.legacy = raw;
        return values;
    }

    markerPositions.sort((a, b) => a.index - b.index);

    const leading = raw.slice(0, markerPositions[0].index).trim();
    if (leading) values.legacy = leading;

    markerPositions.forEach((m, i) => {
        const def = PERSONA_SECTIONS.find((s) => s.key === m.key)!;
        const start = m.index + def.marker.length;
        const end = i + 1 < markerPositions.length ? markerPositions[i + 1].index : raw.length;
        values[m.key] = raw.slice(start, end).trim();
    });

    return values;
}

/** Recompõe o texto final salvo no backend a partir das seções preenchidas. */
export function buildPersonaFromSections(values: PersonaSectionValues): string {
    const parts: string[] = [];
    if (values.legacy.trim()) parts.push(values.legacy.trim());
    PERSONA_SECTIONS.forEach((s) => {
        const content = values[s.key]?.trim();
        if (content) parts.push(`${s.marker}\n${content}`);
    });
    return parts.join("\n\n");
}
