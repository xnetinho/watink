/* @jsxImportSource react */
import React from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import PersonaSectionsEditor from "../PersonaSectionsEditor";
import { type PersonaSectionValues } from "../personaSections";
import type { FlowOption, PipelineOption, KnowledgeBaseOption, AiGateway, AssistantMode } from "../assistantCreatorTypes";

export const PIPELINE_EVENTS = [
  { value: "deal_created", label: "Ticket/negócio criado" },
  { value: "stage_changed", label: "Mudança de estágio" },
  { value: "idle", label: "Parado há X dias" },
  { value: "closed", label: "Finalizado" },
];

interface AssistantBehaviorTabProps {
  mode: AssistantMode;
  onModeChange: (m: AssistantMode) => void;
  flowId: string;
  onFlowIdChange: (v: string) => void;
  flows: FlowOption[];
  pipelineId: string;
  onPipelineIdChange: (v: string) => void;
  pipelines: PipelineOption[];
  pipelineEvents: string[];
  onToggleEvent: (v: string) => void;
  idleThresholdDays: string;
  onIdleThresholdDaysChange: (v: string) => void;
  respondsAfterProactive: boolean;
  onRespondsAfterProactiveChange: (v: boolean) => void;
  showPersonaSections: boolean;
  personaSections: PersonaSectionValues;
  onPersonaSectionChange: (k: string, v: string) => void;
  knowledgeBaseId: string;
  onKnowledgeBaseIdChange: (v: string) => void;
  knowledgeBases: KnowledgeBaseOption[];
  aiGatewayId: string;
  onAiGatewayIdChange: (v: string) => void;
  aiGateways: AiGateway[];
  maxTurns: string;
  onMaxTurnsChange: (v: string) => void;
  ragFallbackBehavior: string;
  onRagFallbackBehaviorChange: (v: string) => void;
  ragFallbackMessage: string;
  onRagFallbackMessageChange: (v: string) => void;
}

export const AssistantBehaviorTab: React.FC<AssistantBehaviorTabProps> = ({
  mode,
  onModeChange,
  flowId,
  onFlowIdChange,
  flows,
  pipelineId,
  onPipelineIdChange,
  pipelines,
  pipelineEvents,
  onToggleEvent,
  idleThresholdDays,
  onIdleThresholdDaysChange,
  respondsAfterProactive,
  onRespondsAfterProactiveChange,
  showPersonaSections,
  personaSections,
  onPersonaSectionChange,
  knowledgeBaseId,
  onKnowledgeBaseIdChange,
  knowledgeBases,
  aiGatewayId,
  onAiGatewayIdChange,
  aiGateways,
  maxTurns,
  onMaxTurnsChange,
  ragFallbackBehavior,
  onRagFallbackBehaviorChange,
  ragFallbackMessage,
  onRagFallbackMessageChange,
}) => {
  return (
    <div className="flex flex-col gap-4 pt-2">
      <div className="flex flex-col gap-1.5">
        <Label>Ao que este assistente está ligado?</Label>
        <Tabs value={mode} onValueChange={(v) => onModeChange(v as AssistantMode)}>
          <TabsList className="grid grid-cols-4 w-full">
            <TabsTrigger value="pipeline">Pipeline</TabsTrigger>
            <TabsTrigger value="flow">Flow</TabsTrigger>
            <TabsTrigger value="persona">Persona</TabsTrigger>
            <TabsTrigger value="router">Roteador</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {mode === "flow" && (
        <div className="flex flex-col gap-1.5">
          <Label>Flow</Label>
          <Select value={flowId} onValueChange={onFlowIdChange}>
            <SelectTrigger>
              <SelectValue placeholder="Selecione um flow" />
            </SelectTrigger>
            <SelectContent>
              {flows.map((f) => (
                <SelectItem key={f.id} value={String(f.id)}>
                  {f.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      {mode === "pipeline" && (
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label>Pipeline</Label>
            <Select value={pipelineId} onValueChange={onPipelineIdChange}>
              <SelectTrigger>
                <SelectValue placeholder="Selecione um pipeline" />
              </SelectTrigger>
              <SelectContent>
                {pipelines.map((p) => (
                  <SelectItem key={p.id} value={String(p.id)}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-2">
            <Label>Eventos que disparam a notificação</Label>
            {PIPELINE_EVENTS.map((ev) => (
              <label
                key={ev.value}
                className="flex items-center gap-2 text-sm cursor-pointer"
              >
                <input
                  type="checkbox"
                  checked={pipelineEvents.includes(ev.value)}
                  onChange={() => onToggleEvent(ev.value)}
                />
                {ev.label}
              </label>
            ))}
          </div>

          {pipelineEvents.includes("idle") && (
            <div className="flex flex-col gap-1.5 max-w-[220px]">
              <Label>Dias parado até disparar</Label>
              <Input
                type="number"
                value={idleThresholdDays}
                onChange={(e) => onIdleThresholdDaysChange(e.target.value)}
              />
            </div>
          )}

          <div className="flex items-center justify-between rounded-xl border p-3">
            <div>
              <p className="text-sm font-medium">
                Responder ativamente após a notificação
              </p>
              <p className="text-xs text-muted-foreground">
                Se desligado, é só uma notificação — respostas do
                contato caem para a fila humana
              </p>
            </div>
            <Switch
              checked={respondsAfterProactive}
              onCheckedChange={onRespondsAfterProactiveChange}
            />
          </div>
        </div>
      )}

      {showPersonaSections && (
        <div className="flex flex-col gap-4 border-t pt-4">
          <PersonaSectionsEditor
            values={personaSections}
            onChange={onPersonaSectionChange}
          />
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 border-t pt-4">
            <div className="flex flex-col gap-1.5">
              <Label>Base de conhecimento</Label>
              <Select value={knowledgeBaseId} onValueChange={onKnowledgeBaseIdChange}>
                <SelectTrigger>
                  <SelectValue placeholder="Nenhuma" />
                </SelectTrigger>
                <SelectContent>
                  {knowledgeBases.map((kb) => (
                    <SelectItem key={kb.id} value={String(kb.id)}>
                      {kb.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Gateway de IA</Label>
              <Select value={aiGatewayId} onValueChange={onAiGatewayIdChange}>
                <SelectTrigger>
                  <SelectValue placeholder="Selecione um gateway" />
                </SelectTrigger>
                <SelectContent>
                  {aiGateways.map((g) => (
                    <SelectItem key={g.id} value={String(g.id)}>
                      {g.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <Label>Máximo de turnos</Label>
              <Input
                type="number"
                value={maxTurns}
                onChange={(e) => onMaxTurnsChange(e.target.value)}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label>Quando o RAG não souber responder</Label>
              <Select
                value={ragFallbackBehavior}
                onValueChange={onRagFallbackBehaviorChange}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="handoff">Passar para atendente</SelectItem>
                  <SelectItem value="apologize">Avisar que não sabe</SelectItem>
                  <SelectItem value="ignore">Não responder</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          {ragFallbackBehavior === "apologize" && (
            <div className="flex flex-col gap-1.5">
              <Label>Mensagem quando não souber</Label>
              <Textarea
                value={ragFallbackMessage}
                onChange={(e) => onRagFallbackMessageChange(e.target.value)}
                placeholder="Desculpe, não tenho essa informação no momento."
                rows={2}
              />
            </div>
          )}
        </div>
      )}
    </div>
  );
};
