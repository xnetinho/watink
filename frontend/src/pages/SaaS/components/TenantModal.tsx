import React from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
    Dialog,
    DialogContent,
    DialogHeader,
    DialogTitle,
    DialogFooter,
} from "@/components/ui/dialog";
import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select";
import type { Tenant, Plan, TenantForm } from "../saasTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface TenantModalProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    selectedTenant: Tenant | null;
    plans: Plan[];
    form: TenantForm;
    onFormChange: (form: TenantForm) => void;
    onSave: () => void;
}

export function TenantModal({
    open,
    onOpenChange,
    selectedTenant,
    plans,
    form,
    onFormChange,
    onSave,
}: TenantModalProps) {
    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-[425px]">
                <DialogHeader>
                    <DialogTitle>Editar Plano - {selectedTenant?.name}</DialogTitle>
                </DialogHeader>
                <div className="grid gap-4 py-4">
                    <div className="grid gap-2">
                        <Label htmlFor="planName">Plano</Label>
                        <Select
                            value={form.planName}
                            onValueChange={(value) => onFormChange({ ...form, planName: value })}
                        >
                            <SelectTrigger id="planName">
                                <SelectValue placeholder={tr("uiText.saas.selectPlan")} />
                            </SelectTrigger>
                            <SelectContent>
                                {plans.map((p) => (
                                    <SelectItem key={p.id} value={p.name}>
                                        {p.name}
                                    </SelectItem>
                                ))}
                                <SelectItem value="Start">Start (Default)</SelectItem>
                                <SelectItem value="Pro">Pro (Default)</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>

                    <div className="grid gap-2">
                        <Label htmlFor="pluginQuota">{tr("uiText.saas.pluginQuota")}</Label>
                        <Input
                            id="pluginQuota"
                            type="number"
                            value={form.pluginQuota}
                            onChange={(e) =>
                                onFormChange({ ...form, pluginQuota: parseInt(e.target.value) || 0 })
                            }
                        />
                    </div>

                    <div className="grid gap-2">
                        <Label htmlFor="tenantStatus">{tr("uiText.saas.subStatus")}</Label>
                        <Select
                            value={form.status}
                            onValueChange={(value) => onFormChange({ ...form, status: value })}
                        >
                            <SelectTrigger id="tenantStatus">
                                <SelectValue placeholder="Selecione o status" />
                            </SelectTrigger>
                            <SelectContent>
                                <SelectItem value="active">{tr("uiText.common.active")}</SelectItem>
                                <SelectItem value="overdue">Inadimplente</SelectItem>
                            </SelectContent>
                        </Select>
                    </div>

                    <div className="grid gap-2">
                        <Label htmlFor="expiresAt">{tr("uiText.saas.expiresAt")}</Label>
                        <Input
                            id="expiresAt"
                            type="date"
                            value={form.expiresAt}
                            onChange={(e) => onFormChange({ ...form, expiresAt: e.target.value })}
                        />
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
