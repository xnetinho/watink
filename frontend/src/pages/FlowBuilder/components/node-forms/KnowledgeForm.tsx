import React from 'react';
import { Label } from '@/components/ui/label';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { NodeData, KnowledgeBase } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface KnowledgeFormProps {
  formData: NodeData;
  knowledgeBases: KnowledgeBase[];
  onChange: (field: string, value: unknown) => void;
}

const KnowledgeForm: React.FC<KnowledgeFormProps> = ({ formData, knowledgeBases, onChange }) => (
  <div className="space-y-3">
    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.responseMode")}</Label>
      <Select value={formData.responseMode || 'auto'} onValueChange={(v) => onChange('responseMode', v)}>
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="auto">{tr("uiText.flow.form.autoReply")}</SelectItem>
          <SelectItem value="suggest">{tr("uiText.flow.form.agentSuggestion")}</SelectItem>
          <SelectItem value="search">Apenas Busca</SelectItem>
        </SelectContent>
      </Select>
    </div>

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
  </div>
);

export default KnowledgeForm;
