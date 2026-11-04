import React from "react";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import type {
    GroupCampaignScheduleMode,
    GroupCampaignRecurrenceRule,
} from "@/services/groupCampaignService";

import { localeTag } from "../../../lib/dateLocale";
import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const WEEKDAYS: { value: string; label: string }[] = [
    { value: "0", label: "uiText.campaigns.wd0" },
    { value: "1", label: "uiText.campaigns.wd1" },
    { value: "2", label: "uiText.campaigns.wd2" },
    { value: "3", label: "uiText.campaigns.wd3" },
    { value: "4", label: "uiText.campaigns.wd4" },
    { value: "5", label: "uiText.campaigns.wd5" },
    { value: "6", label: "uiText.campaigns.wd6" },
];

export interface CampaignScheduleState {
    scheduleMode: GroupCampaignScheduleMode;
    startAt: string; // datetime-local value
    recurrenceRule: GroupCampaignRecurrenceRule;
    recurrenceDays: string; // CSV
    recurrenceTime: string; // HH:MM
    recurrenceEndAt: string; // date value, empty = indeterminado
}

interface CampaignScheduleFormProps {
    value: CampaignScheduleState;
    onChange: (patch: Partial<CampaignScheduleState>) => void;
}

function toggleDay(csv: string, day: string): string {
    const days = csv ? csv.split(",").filter(Boolean) : [];
    return days.includes(day) ? days.filter((d) => d !== day).join(",") : [...days, day].join(",");
}

function summarize(v: CampaignScheduleState): string {
    if (v.scheduleMode === "immediate") return tr("uiText.campaigns.fireNow");
    if (v.scheduleMode === "once") {
        if (!v.startAt) return "Selecione a data e hora.";
        return tr("uiText.campaigns.fireAt", { date: new Date(v.startAt).toLocaleString(localeTag()) });
    }
    const time = v.recurrenceTime || "--:--";
    if (v.recurrenceRule === "weekly") {
        const days = v.recurrenceDays
            .split(",")
            .filter(Boolean)
            .map((d) => WEEKDAYS.find((w) => w.value === d)?.label ? tr(WEEKDAYS.find((w) => w.value === d)!.label) : d);
        const daysLabel = days.length > 0 ? days.join(", ") : tr("uiText.campaigns.noDay");
        return tr("uiText.campaigns.weeklySummary", { days: daysLabel, time, end: v.recurrenceEndAt ? tr("uiText.campaigns.until", { date: v.recurrenceEndAt }) : tr("uiText.campaigns.noEnd") });
    }
    const days = v.recurrenceDays.split(",").filter(Boolean).join(", ") || tr("uiText.campaigns.noDay");
    return tr("uiText.campaigns.monthlySummary", { days, time, end: v.recurrenceEndAt ? tr("uiText.campaigns.until", { date: v.recurrenceEndAt }) : tr("uiText.campaigns.noEnd") });
}

const CampaignScheduleForm: React.FC<CampaignScheduleFormProps> = ({ value, onChange }) => {
    return (
        <div className="space-y-4">
            <RadioGroup
                value={value.scheduleMode}
                onValueChange={(v) => onChange({ scheduleMode: v as GroupCampaignScheduleMode })}
            >
                <div className="flex items-center gap-2">
                    <RadioGroupItem value="immediate" id="sched-immediate" />
                    <Label htmlFor="sched-immediate" className="font-normal cursor-pointer">
                        Imediato
                    </Label>
                </div>
                <div className="flex items-center gap-2">
                    <RadioGroupItem value="once" id="sched-once" />
                    <Label htmlFor="sched-once" className="font-normal cursor-pointer">
                        {tr("uiText.campaigns.specificDate")}
                    </Label>
                </div>
                <div className="flex items-center gap-2">
                    <RadioGroupItem value="recurring" id="sched-recurring" />
                    <Label htmlFor="sched-recurring" className="font-normal cursor-pointer">
                        Recorrente
                    </Label>
                </div>
            </RadioGroup>

            {value.scheduleMode === "once" && (
                <div className="space-y-1.5 pl-6">
                    <Label htmlFor="sched-startat">Data e hora</Label>
                    <Input
                        id="sched-startat"
                        type="datetime-local"
                        value={value.startAt}
                        onChange={(e) => onChange({ startAt: e.target.value })}
                        className="w-[240px]"
                    />
                </div>
            )}

            {value.scheduleMode === "recurring" && (
                <div className="space-y-3 pl-6">
                    <RadioGroup
                        value={value.recurrenceRule || "weekly"}
                        onValueChange={(v) => onChange({ recurrenceRule: v as GroupCampaignRecurrenceRule, recurrenceDays: "" })}
                        className="flex gap-4"
                    >
                        <div className="flex items-center gap-2">
                            <RadioGroupItem value="weekly" id="rec-weekly" />
                            <Label htmlFor="rec-weekly" className="font-normal cursor-pointer">
                                Semanal
                            </Label>
                        </div>
                        <div className="flex items-center gap-2">
                            <RadioGroupItem value="monthly" id="rec-monthly" />
                            <Label htmlFor="rec-monthly" className="font-normal cursor-pointer">
                                Mensal
                            </Label>
                        </div>
                    </RadioGroup>

                    {value.recurrenceRule === "monthly" ? (
                        <div className="space-y-1.5">
                            <Label>{tr("uiText.campaigns.dayOfMonth")}</Label>
                            <Input
                                type="number"
                                min={1}
                                max={31}
                                placeholder="1-31"
                                className="w-[100px]"
                                value={value.recurrenceDays}
                                onChange={(e) => onChange({ recurrenceDays: e.target.value })}
                            />
                        </div>
                    ) : (
                        <div className="space-y-1.5">
                            <Label>{tr("uiText.campaigns.weekdays")}</Label>
                            <div className="flex gap-1.5 flex-wrap">
                                {WEEKDAYS.map((d) => {
                                    const active = value.recurrenceDays.split(",").includes(d.value);
                                    return (
                                        <button
                                            key={d.value}
                                            type="button"
                                            onClick={() => onChange({ recurrenceDays: toggleDay(value.recurrenceDays, d.value) })}
                                            className={`rounded-lg border px-2.5 py-1.5 text-xs font-medium transition-colors ${
                                                active
                                                    ? "border-primary bg-primary/10 text-primary"
                                                    : "border-border text-muted-foreground hover:bg-muted/40"
                                            }`}
                                        >
                                            {tr(d.label)}
                                        </button>
                                    );
                                })}
                            </div>
                        </div>
                    )}

                    <div className="flex items-end gap-4">
                        <div className="space-y-1.5">
                            <Label htmlFor="sched-time">{tr("uiText.campaigns.time")}</Label>
                            <Input
                                id="sched-time"
                                type="time"
                                className="w-[120px]"
                                value={value.recurrenceTime}
                                onChange={(e) => onChange({ recurrenceTime: e.target.value })}
                            />
                        </div>
                        <div className="space-y-1.5">
                            <Label htmlFor="sched-end">Data final (opcional)</Label>
                            <Input
                                id="sched-end"
                                type="date"
                                className="w-[160px]"
                                value={value.recurrenceEndAt}
                                onChange={(e) => onChange({ recurrenceEndAt: e.target.value })}
                            />
                        </div>
                    </div>
                </div>
            )}

            <p className="text-xs text-muted-foreground bg-muted/40 rounded-lg px-3 py-2">{summarize(value)}</p>
        </div>
    );
};

export default CampaignScheduleForm;
