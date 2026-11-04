import React from 'react';
import { Badge } from '@/components/ui/badge';
import { CONTEXT_VARIABLES } from './filterBuilderTypes';

import { i18n } from '../../../translate/i18n';

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const FilterContextVariables: React.FC = () => {
  return (
    <div className="mt-2 p-2 bg-muted/50 rounded-md">
      <p className="text-[10px] text-muted-foreground mb-1">{tr("uiText.flow.data.availableVars")}</p>
      <div className="flex flex-wrap gap-1">
        {CONTEXT_VARIABLES.map((v) => (
          <Badge
            key={v.value}
            variant="secondary"
            className="text-[10px] cursor-pointer hover:bg-primary hover:text-primary-foreground"
            onClick={() => navigator.clipboard.writeText(v.value)}
          >
            {tr(v.label)}
          </Badge>
        ))}
      </div>
    </div>
  );
};

export default FilterContextVariables;
