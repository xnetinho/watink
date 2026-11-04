import React, { useEffect, useState } from "react";
import { toast } from "react-toastify";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import {
    Assistant,
    AssistantRouterOption,
    createRouterOption,
    deleteRouterOption,
    listAssistants,
    listRouterOptions,
    updateRouterOption,
} from "../../services/assistantService";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface Props {
    assistantId: number;
}

const AssistantRouterOptions: React.FC<Props> = ({ assistantId }) => {
    const [options, setOptions] = useState<AssistantRouterOption[]>([]);
    const [candidates, setCandidates] = useState<Assistant[]>([]);
    const [label, setLabel] = useState("");
    const [targetAssistantId, setTargetAssistantId] = useState("");

    const load = async () => {
        try {
            const [opts, all] = await Promise.all([
                listRouterOptions(assistantId),
                listAssistants(),
            ]);
            setOptions(opts);
            setCandidates(all.filter((a) => a.id !== assistantId));
        } catch {
            toast.error(tr("uiText.assistants.menuLoadError"));
        }
    };

    useEffect(() => {
        load();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [assistantId]);

    const handleAdd = async () => {
        if (!label.trim() || !targetAssistantId) {
            toast.error(tr("uiText.assistants.menuFill"));
            return;
        }
        try {
            await createRouterOption(assistantId, {
                label,
                order: options.length,
                targetAssistantId: Number(targetAssistantId),
            });
            setLabel("");
            setTargetAssistantId("");
            load();
        } catch (err) {
            const message =
                (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
                tr("uiText.assistants.optionAddError");
            toast.error(message);
        }
    };

    const handleRemove = async (optionId: number) => {
        try {
            await deleteRouterOption(assistantId, optionId);
            load();
        } catch {
            toast.error(tr("uiText.assistants.optionRemoveError"));
        }
    };

    const handleReorderLabel = async (option: AssistantRouterOption, value: string) => {
        try {
            await updateRouterOption(assistantId, option.id, {
                label: value,
                order: option.order,
                targetAssistantId: option.targetAssistantId,
            });
            load();
        } catch {
            toast.error(tr("uiText.assistants.optionUpdateError"));
        }
    };

    return (
        <div className="flex flex-col gap-3">
            <Label>{tr("uiText.assistants.menuOptions")}</Label>
            {options.length === 0 && (
                <p className="text-sm text-muted-foreground">{tr("uiText.assistants.noOptions")}</p>
            )}
            {options.map((option) => (
                <div key={option.id} className="flex items-center gap-2">
                    <Input
                        value={option.label}
                        onChange={(e) => handleReorderLabel(option, e.target.value)}
                        className="flex-1"
                    />
                    <span className="text-sm text-muted-foreground min-w-[160px]">
                        →{" "}
                        {candidates.find((c) => c.id === option.targetAssistantId)?.name ??
                            `Assistant #${option.targetAssistantId}`}
                    </span>
                    <Button variant="ghost" size="icon" onClick={() => handleRemove(option.id)}>
                        <Trash2 className="h-4 w-4 text-destructive" />
                    </Button>
                </div>
            ))}

            <div className="flex items-center gap-2 border-t pt-3">
                <Input
                    placeholder={tr("uiText.assistants.optionLabel")}
                    value={label}
                    onChange={(e) => setLabel(e.target.value)}
                    className="flex-1"
                />
                <Select value={targetAssistantId} onValueChange={setTargetAssistantId}>
                    <SelectTrigger className="w-[220px]">
                        <SelectValue placeholder={tr("uiText.assistants.targetAssistant")} />
                    </SelectTrigger>
                    <SelectContent>
                        {candidates.map((a) => (
                            <SelectItem key={a.id} value={String(a.id)}>
                                {a.name}
                            </SelectItem>
                        ))}
                    </SelectContent>
                </Select>
                <Button variant="outline" onClick={handleAdd}>
                    <Plus className="mr-2 h-4 w-4" />
                    {tr("uiText.common.add")}
                </Button>
            </div>
        </div>
    );
};

export default AssistantRouterOptions;
