import React, { memo } from 'react';
import { Globe } from 'lucide-react';
import BaseNode from './BaseNode';

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface APINodeProps {
  data: Record<string, unknown>;
  isConnectable?: boolean;
}

const APINode: React.FC<APINodeProps> = ({ data, isConnectable }) => (
  <BaseNode
    data={data}
    icon={Globe}
    colorClass="colorApi"
    defaultLabel="uiText.flow.node.api"
    sublabel={tr("uiText.flow.ui.externalIntegration")}
    isConnectable={isConnectable}
  />
);

export default memo(APINode);
