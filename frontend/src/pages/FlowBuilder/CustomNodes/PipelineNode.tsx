import React from 'react';
import { GitCommit } from 'lucide-react';
import BaseNode from './BaseNode';

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface PipelineNodeProps {
  data: Record<string, unknown>;
  isConnectable?: boolean;
}

const getActionLabel = (action?: string): string => {
  switch (action) {
    case 'createDeal': return 'Criar Oportunidade';
    case 'moveDeal': return 'Mover Etapa';
    case 'updateDeal': return 'Atualizar Oportunidade';
    default: return tr("uiText.flow.ui.kanbanIntegration");
  }
};

const PipelineNode: React.FC<PipelineNodeProps> = ({ data, isConnectable }) => (
  <BaseNode
    data={data}
    icon={GitCommit}
    colorClass="colorPipeline"
    defaultLabel="uiText.flow.node.pipeline"
    sublabel={getActionLabel(data?.kanbanAction as string)}
    isConnectable={isConnectable}
  />
);

export default PipelineNode;
