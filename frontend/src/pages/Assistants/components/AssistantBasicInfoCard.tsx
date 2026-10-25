/* @jsxImportSource react */
import React from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import type { WhatsappOption } from "../assistantCreatorTypes";

interface AssistantBasicInfoCardProps {
  name: string;
  onNameChange: (val: string) => void;
  whatsappId: string;
  onWhatsappIdChange: (val: string) => void;
  whatsapps: WhatsappOption[];
  description: string;
  onDescriptionChange: (val: string) => void;
  active: boolean;
  onActiveChange: (val: boolean) => void;
  allowMultipleOnConnection: boolean;
  onAllowMultipleOnConnectionChange: (val: boolean) => void;
}

export const AssistantBasicInfoCard: React.FC<AssistantBasicInfoCardProps> = ({
  name,
  onNameChange,
  whatsappId,
  onWhatsappIdChange,
  whatsapps,
  description,
  onDescriptionChange,
  active,
  onActiveChange,
  allowMultipleOnConnection,
  onAllowMultipleOnConnectionChange,
}) => {
  return (
    <Card className="rounded-2xl shadow-[0px_4px_20px_rgba(0,0,0,0.06)]">
      <CardContent className="p-5 flex flex-col gap-4">
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>Nome</Label>
            <Input value={name} onChange={(e) => onNameChange(e.target.value)} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label>Conexão</Label>
            <Select value={whatsappId} onValueChange={onWhatsappIdChange}>
              <SelectTrigger>
                <SelectValue placeholder="Selecione" />
              </SelectTrigger>
              <SelectContent>
                {whatsapps.map((w) => (
                  <SelectItem key={w.id} value={String(w.id)}>
                    {w.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label>Descrição (opcional)</Label>
          <Input value={description} onChange={(e) => onDescriptionChange(e.target.value)} />
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="flex items-center justify-between rounded-xl border p-3">
            <p className="text-sm font-medium">Assistente ativo</p>
            <Switch checked={active} onCheckedChange={onActiveChange} />
          </div>
          <div className="flex items-center justify-between rounded-xl border p-3">
            <p className="text-sm font-medium">Permitir múltiplos bots na mesma conexão</p>
            <Switch checked={allowMultipleOnConnection} onCheckedChange={onAllowMultipleOnConnectionChange} />
          </div>
        </div>
      </CardContent>
    </Card>
  );
};
