import React from 'react';
import { Label } from '@/components/ui/label';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import ConditionBuilder from '../ConditionBuilder';
import { NodeData } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface TriggerFormProps {
  formData: NodeData;
  onChange: (field: string, value: unknown) => void;
}

const TriggerForm: React.FC<TriggerFormProps> = ({ formData, onChange }) => (
  <div className="space-y-3">
    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.triggerType")}</Label>
      <Select value={formData.triggerType || 'keyword'} onValueChange={(v) => onChange('triggerType', v)}>
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="keyword">Palavra-chave</SelectItem>
          <SelectItem value="any">Qualquer Mensagem</SelectItem>
          <SelectItem value="firstContact">Primeiro Contato</SelectItem>
        </SelectContent>
      </Select>
      <p className="text-[11px] text-muted-foreground">
        A conexão atrelada a este fluxo é definida na barra do editor (no topo) — assim você pode criar sem conexão e atrelar/trocar depois.
      </p>
    </div>

    {formData.triggerType === 'keyword' && (
      <div className="space-y-1 pt-2">
        <p className="text-xs font-medium text-muted-foreground">{tr("uiText.flow.form.activationCondition")}</p>
        <ConditionBuilder
          conditions={formData.conditions || []}
          onChange={(c) => onChange('conditions', c)}
          title=""
          maxConditions={3}
        />
      </div>
    )}
  </div>
);

export default TriggerForm;
