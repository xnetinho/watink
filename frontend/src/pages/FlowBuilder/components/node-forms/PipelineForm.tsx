import React from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { NodeData, Pipeline } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface PipelineFormProps {
  formData: NodeData;
  pipelines: Pipeline[];
  onChange: (field: string, value: unknown) => void;
}

const PipelineForm: React.FC<PipelineFormProps> = ({ formData, pipelines, onChange }) => (
  <div className="space-y-3">
    <p className="text-xs text-muted-foreground">{tr("uiText.flow.form.pipelineIntro")}</p>

    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.kanbanAction")}</Label>
      <Select value={formData.kanbanAction || 'createDeal'} onValueChange={(v) => onChange('kanbanAction', v)}>
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="createDeal">Criar Oportunidade</SelectItem>
          <SelectItem value="moveDeal">Mover Oportunidade</SelectItem>
        </SelectContent>
      </Select>
    </div>

    {formData.kanbanAction === 'createDeal' && (
      <>
        <div className="space-y-1">
          <Label className="text-xs">{tr("uiText.flow.form.dealTitle")}</Label>
          <Input className="h-8 text-xs" placeholder={tr("uiText.flow.form.useVars")} value={formData.dealTitle || ''} onChange={(e) => onChange('dealTitle', e.target.value)} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">Valor (R$)</Label>
          <Input className="h-8 text-xs" placeholder="150.00" value={formData.dealValue || ''} onChange={(e) => onChange('dealValue', e.target.value)} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{tr("uiText.common.priority")}</Label>
          <Select value={formData.dealPriority || '1'} onValueChange={(v) => onChange('dealPriority', v)}>
            <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="1">{tr("uiText.common.low")}</SelectItem>
              <SelectItem value="2">{tr("uiText.common.medium")}</SelectItem>
              <SelectItem value="3">{tr("uiText.common.high")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </>
    )}

    <div className="space-y-1">
      <Label className="text-xs">Pipeline</Label>
      <Select value={formData.pipelineId || ''} onValueChange={(v) => onChange('pipelineId', v)}>
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          {pipelines.map((p) => (
            <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>

    <div className="space-y-1">
      <Label className="text-xs">Etapa (Coluna)</Label>
      <Select
        value={formData.stageId || ''}
        onValueChange={(v) => onChange('stageId', v)}
        disabled={!formData.pipelineId}
      >
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          {pipelines.find((p) => p.id === formData.pipelineId)?.stages?.map((s) => (
            <SelectItem key={s.id} value={s.id}>{s.name}</SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  </div>
);

export default PipelineForm;
