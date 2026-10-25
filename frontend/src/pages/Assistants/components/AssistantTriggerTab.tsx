/* @jsxImportSource react */
import React from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

interface AssistantTriggerTabProps {
  triggerType: "any" | "keyword";
  onTriggerTypeChange: (t: "any" | "keyword") => void;
  triggerOperator: string;
  onTriggerOperatorChange: (op: string) => void;
  triggerValue: string;
  onTriggerValueChange: (val: string) => void;
}

export const AssistantTriggerTab: React.FC<AssistantTriggerTabProps> = ({
  triggerType,
  onTriggerTypeChange,
  triggerOperator,
  onTriggerOperatorChange,
  triggerValue,
  onTriggerValueChange,
}) => {
  return (
    <div className="flex flex-col gap-4 pt-2">
      <Label>Gatilho</Label>
      <Tabs
        value={triggerType}
        onValueChange={(v) => onTriggerTypeChange(v as "any" | "keyword")}
      >
        <TabsList className="grid grid-cols-2 w-full max-w-xs">
          <TabsTrigger value="any">Todas as mensagens</TabsTrigger>
          <TabsTrigger value="keyword">Palavra-chave</TabsTrigger>
        </TabsList>
      </Tabs>

      {triggerType === "keyword" && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>Operador</Label>
            <Select value={triggerOperator} onValueChange={onTriggerOperatorChange}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="contains">contém</SelectItem>
                <SelectItem value="equals">igual</SelectItem>
                <SelectItem value="starts_with">começa com</SelectItem>
                <SelectItem value="ends_with">termina com</SelectItem>
                <SelectItem value="regex">regex</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Valor do gatilho</Label>
            <Input
              value={triggerValue}
              onChange={(e) => onTriggerValueChange(e.target.value)}
              placeholder="Ex: oi, menu, começar"
            />
          </div>
        </div>
      )}
    </div>
  );
};
