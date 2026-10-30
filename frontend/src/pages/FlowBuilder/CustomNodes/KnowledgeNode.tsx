import React from 'react';
import { Lightbulb } from 'lucide-react';
import BaseNode from './BaseNode';

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface KnowledgeNodeProps {
  data: Record<string, unknown>;
  isConnectable?: boolean;
}

const KnowledgeNode: React.FC<KnowledgeNodeProps> = ({ data, isConnectable }) => {
  const mode = (data?.responseMode as string) || 'auto';

  return (
    <BaseNode
      data={data}
      icon={Lightbulb}
      colorClass="colorKnowledge"
      defaultLabel="uiText.flow.node.ai"
      sublabel={mode === 'auto' ? tr("uiText.flow.ui.auto") : mode}
      isConnectable={isConnectable}
    />
  );
};

export default KnowledgeNode;
