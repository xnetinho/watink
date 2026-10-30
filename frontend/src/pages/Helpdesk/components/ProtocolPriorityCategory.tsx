import React from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ProtocolFormData } from "../hooks/useProtocolModal";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface ProtocolPriorityCategoryProps {
  formData: ProtocolFormData;
  helpdeskEnabled: boolean;
  categories: string[];
  onSelectChange: (name: keyof ProtocolFormData) => (value: string) => void;
  onInputChange: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
}

export function ProtocolPriorityCategory({
  formData,
  helpdeskEnabled,
  categories,
  onSelectChange,
  onInputChange,
}: ProtocolPriorityCategoryProps) {
  return (
    <div className={`grid gap-4 ${helpdeskEnabled ? "grid-cols-2" : "grid-cols-1"}`}>
      <div className="flex flex-col gap-1.5">
        <Label>Prioridade</Label>
        <Select value={formData.priority} onValueChange={onSelectChange("priority")}>
          <SelectTrigger>
            <SelectValue placeholder="Selecione a prioridade" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="low">{tr("uiText.helpdesk.priority.low")}</SelectItem>
            <SelectItem value="medium">{tr("uiText.helpdesk.priority.medium")}</SelectItem>
            <SelectItem value="high">{tr("uiText.helpdesk.priority.high")}</SelectItem>
            <SelectItem value="urgent">{tr("uiText.helpdesk.priority.urgent")}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {helpdeskEnabled && (
        <div className="flex flex-col gap-1.5">
          <Label>Categoria</Label>
          {categories.length > 0 ? (
            <Select value={formData.category} onValueChange={onSelectChange("category")}>
              <SelectTrigger>
                <SelectValue placeholder="Selecione a categoria" />
              </SelectTrigger>
              <SelectContent>
                {categories.map((cat) => (
                  <SelectItem key={cat} value={cat}>
                    {cat}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <Input
              name="category"
              value={formData.category}
              onChange={onInputChange}
              placeholder="Categoria"
            />
          )}
        </div>
      )}
    </div>
  );
}
