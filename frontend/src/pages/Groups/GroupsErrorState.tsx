import React from "react";
import { Lock, Clock, PlugZap, AlertTriangle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useNavigate } from "react-router";
import { GroupsApiError } from "./groupTypes";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface GroupsErrorStateProps {
    error: GroupsApiError;
}

/**
 * Dedicated states for 402 (plugin blocked)/429 (throttled)/501 (provider
 * doesn't support groups) — plan §6.5: these are NEVER a generic error
 * toast, each has its own explanation and (where relevant) a CTA.
 */
const GroupsErrorState: React.FC<GroupsErrorStateProps> = ({ error }) => {
    const navigate = useNavigate();

    if (error.kind === "blocked") {
        return (
            <div className="flex flex-col items-center justify-center gap-4 h-full text-center px-6">
                <div className="rounded-full bg-[hsl(var(--status-warning-bg))] p-4">
                    <Lock className="h-8 w-8 text-[hsl(var(--status-warning-text))]" />
                </div>
                <div>
                    <h3 className="text-lg font-semibold">Plugin bloqueado</h3>
                    <p className="text-sm text-muted-foreground mt-1 max-w-md">
                        {tr("uiText.groups.pluginRequired")}
                    </p>
                </div>
                <Button onClick={() => navigate("/admin/settings/plugins")}>{tr("uiText.groups.goPlugins")}</Button>
            </div>
        );
    }

    if (error.kind === "rateLimited") {
        return (
            <div className="flex flex-col items-center justify-center gap-4 h-full text-center px-6">
                <div className="rounded-full bg-[hsl(var(--status-warning-bg))] p-4">
                    <Clock className="h-8 w-8 text-[hsl(var(--status-warning-text))]" />
                </div>
                <div>
                    <h3 className="text-lg font-semibold">{tr("uiText.groups.tooMany")}</h3>
                    <p className="text-sm text-muted-foreground mt-1 max-w-md">
                        {error.message || tr("uiText.groups.tooManyHint")}
                    </p>
                </div>
            </div>
        );
    }

    if (error.kind === "notSupported") {
        return (
            <div className="flex flex-col items-center justify-center gap-4 h-full text-center px-6">
                <div className="rounded-full bg-[hsl(var(--status-default-bg))] p-4">
                    <PlugZap className="h-8 w-8 text-[hsl(var(--status-default-text))]" />
                </div>
                <div>
                    <h3 className="text-lg font-semibold">{tr("uiText.groups.noSupport")}</h3>
                    <p className="text-sm text-muted-foreground mt-1 max-w-md">
                        {tr("uiText.groups.noSupportHint")}
                    </p>
                </div>
            </div>
        );
    }

    return (
        <div className="flex flex-col items-center justify-center gap-4 h-full text-center px-6">
            <div className="rounded-full bg-[hsl(var(--status-error-bg))] p-4">
                <AlertTriangle className="h-8 w-8 text-[hsl(var(--status-error-text))]" />
            </div>
            <div>
                <h3 className="text-lg font-semibold">{tr("uiText.groups.loadFail")}</h3>
                <p className="text-sm text-muted-foreground mt-1 max-w-md">{error.message}</p>
            </div>
        </div>
    );
};

export default GroupsErrorState;
