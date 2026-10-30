import React from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from "@/components/ui/dialog";
import type { Plan, PlanForm } from "../saasTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface PlanModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    selectedPlan: Plan | null;
    form: PlanForm;
    onFormChange: (form: PlanForm) => void;
    onSave: () => void;
}

export function PlanModal({
    open,
    onOpenChange,
    selectedPlan,
    form,
    onFormChange,
    onSave,
}: PlanModalProps) {
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>{selectedPlan ? "Editar Plano" : tr("uiText.saas.newPlan")}</DialogTitle>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                    <div className="grid gap-2">
                        <Label htmlFor="planNameInput">{tr("uiText.saas.planName")}</Label>
                        <Input
                            id="planNameInput"
                            value={form.name}
                            onChange={(e) => onFormChange({ ...form, name: e.target.value })}
                        />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div className="grid gap-2">
                            <Label htmlFor="planPluginQuota">{tr("uiText.saas.pluginQuota")}</Label>
                            <Input
                                id="planPluginQuota"
                                type="number"
                                value={form.pluginQuota}
                                onChange={(e) =>
                                    onFormChange({ ...form, pluginQuota: parseInt(e.target.value) || 0 })
                                }
                            />
                        </div>
                        <div className="grid gap-2">
                            <Label htmlFor="planPrice">{tr("uiText.saas.price")}</Label>
                            <Input
                                id="planPrice"
                                type="number"
                                step="0.01"
                                value={form.price}
                                onChange={(e) =>
                                    onFormChange({ ...form, price: parseFloat(e.target.value) || 0 })
                                }
                            />
                        </div>
                    </div>

                    <div className="flex items-center space-x-2 pt-2">
                        <Switch
                            id="planActive"
                            checked={form.active}
                            onCheckedChange={(checked) => onFormChange({ ...form, active: checked })}
                        />
                        <Label htmlFor="planActive">Plano Ativo</Label>
                    </div>
                </div>
                <DialogFooter>
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                        {tr("uiText.common.cancel")}
                    </Button>
                    <Button onClick={onSave}>{tr("uiText.common.save")}</Button>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
