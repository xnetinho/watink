import React from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { NodeData, KnowledgeBase } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface AgentFormProps {
  formData: NodeData;
  knowledgeBases: KnowledgeBase[];
  onChange: (field: string, value: unknown) => void;
}

const AgentForm: React.FC<AgentFormProps> = ({ formData, knowledgeBases, onChange }) => (
  <div className="space-y-3">
    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.knowledgeBase")}</Label>
      <Select
        value={formData.knowledgeBaseId || undefined}
        onValueChange={(v) => onChange('knowledgeBaseId', v)}
      >
        <SelectTrigger className="h-8 text-xs"><SelectValue placeholder={tr("uiText.flow.form.select")} /></SelectTrigger>
        <SelectContent>
          {knowledgeBases.map((kb) => (
            <SelectItem key={kb.id} value={kb.id}>{kb.name}</SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>

    <div className="space-y-1">
      <Label className="text-xs">Persona</Label>
      <Textarea
        className="text-xs min-h-[96px]"
        placeholder={tr("uiText.flow.form.agentPrompt")}
        value={formData.persona || ''}
        onChange={(e) => onChange('persona', e.target.value)}
      />
      <p className="text-[10px] text-muted-foreground">{tr("uiText.flow.form.agentPromptHint")}</p>
    </div>

    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.maxTurns")}</Label>
      <Input
        type="number"
        min={1}
        className="h-8 text-xs"
        placeholder="10"
        value={formData.maxTurns ?? ''}
        onChange={(e) => {
          const raw = e.target.value;
          onChange('maxTurns', raw === '' ? undefined : Number(raw));
        }}
      />
      <p className="text-[10px] text-muted-foreground">{tr("uiText.flow.form.maxTurnsHint")}</p>
    </div>
  </div>
);

export default AgentForm;
