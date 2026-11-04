import React from "react";
import { Edit2, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/ui/table";
import type { Plan } from "../saasTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface PlansTabProps {
    plans: Plan[];
    onAdd: () => void;
    onEdit: (plan: Plan) => void;
    onDelete: (planId: number | string) => void;
}

export function PlansTab({ plans, onAdd, onEdit, onDelete }: PlansTabProps) {
    return (
        <div className="space-y-4">
            <div className="flex items-center justify-between">
                <h3 className="text-lg font-medium">{tr("uiText.saas.plansConfig")}</h3>
                <Button onClick={onAdd}>
                    <Plus className="mr-2 h-4 w-4" /> Novo Plano
                </Button>
            </div>
            <div className="rounded-md border">
                <Table>
                    <TableHeader>
                        <TableRow>
                            <TableHead>{tr("uiText.common.name")}</TableHead>
                            <TableHead>Plugins</TableHead>
                            <TableHead>{tr("uiText.saas.priceShort")}</TableHead>
                            <TableHead>{tr("uiText.common.status")}</TableHead>
                            <TableHead className="w-[120px] text-right">{tr("uiText.common.actions")}</TableHead>
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {plans.map((plan) => (
                            <TableRow key={plan.id}>
                                <TableCell className="font-medium">{plan.name}</TableCell>
                                <TableCell>{plan.pluginQuota}</TableCell>
                                <TableCell>R$ {plan.price.toFixed(2)}</TableCell>
                                <TableCell>
                                    <span
                                        className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                                            plan.active
                                                ? "bg-green-100 text-green-800"
                                                : "bg-gray-100 text-gray-800"
                                        }`}
                                    >
                                        {plan.active ? tr("uiText.common.active") : tr("uiText.common.inactive")}
                                    </span>
                                </TableCell>
                                <TableCell className="text-right">
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        onClick={() => onEdit(plan)}
                                    >
                                        <Edit2 className="h-4 w-4" />
                                    </Button>
                                    <Button
                                        variant="ghost"
                                        size="icon"
                                        onClick={() => onDelete(plan.id)}
                                    >
                                        <Trash2 className="h-4 w-4 text-destructive" />
                                    </Button>
                                </TableCell>
                            </TableRow>
                        ))}
                        {plans.length === 0 && (
                            <TableRow>
                                <TableCell colSpan={5} className="h-24 text-center">
                                    Nenhum plano encontrado.
                                </TableCell>
                            </TableRow>
                        )}
                    </TableBody>
                </Table>
            </div>
        </div>
    );
}
