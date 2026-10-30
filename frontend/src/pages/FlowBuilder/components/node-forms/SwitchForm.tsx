import React from 'react';
import ConditionBuilder from '../ConditionBuilder';
import { NodeData } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface SwitchFormProps {
  formData: NodeData;
  onChange: (field: string, value: unknown) => void;
}

const SwitchForm: React.FC<SwitchFormProps> = ({ formData, onChange }) => (
  <div className="space-y-3">
    <p className="text-xs text-muted-foreground">
      {tr("uiText.flow.form.switchIntro")}
    </p>
    <div className="space-y-1">
      <p className="text-xs font-medium">{tr("uiText.flow.form.conditionForA")}</p>
      <ConditionBuilder
        conditions={formData.conditionsA || []}
        onChange={(c) => onChange('conditionsA', c)}
        title=""
        maxConditions={3}
      />
    </div>
  </div>
);

export default SwitchForm;
