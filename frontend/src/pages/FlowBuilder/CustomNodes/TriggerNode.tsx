import React from 'react';
import { Position } from '@xyflow/react';
import { Bell } from 'lucide-react';
import BaseNode from './BaseNode';

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface TriggerNodeProps {
  data: Record<string, unknown>;
  isConnectable?: boolean;
}

const TRIGGER_LABELS: Record<string, string> = {
  time: 'uiText.flow.ui.trgTime',
  message: "uiText.flow.ui.message",
  webhook: 'uiText.flow.node.webhook',
  event: 'uiText.flow.ui.trgEvent',
};

const TriggerNode: React.FC<TriggerNodeProps> = ({ data, isConnectable }) => {
  const triggerType = (data?.triggerType as string) || 'message';
  const triggerLabel = TRIGGER_LABELS[triggerType] ? String(i18n.t(TRIGGER_LABELS[triggerType])) : triggerType;

  return (
    <BaseNode
      data={data}
      icon={Bell}
      colorClass="colorTrigger"
      defaultLabel="uiText.flow.node.trigger"
      sublabel={triggerLabel}
      isConnectable={isConnectable}
      targetHandles={[]}
      sourceHandles={[{ id: null, position: Position.Right }]}
    />
  );
};

export default TriggerNode;
