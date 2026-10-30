import React from 'react';
import { Database } from 'lucide-react';
import BaseNode from './BaseNode';

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface DatabaseNodeProps {
  data: Record<string, unknown>;
  isConnectable?: boolean;
}

const OPERATION_LABELS: Record<string, string> = {
  read: tr("uiText.flow.ui.opRead"),
  update: tr("uiText.flow.ui.opUpdate"),
  create: tr("uiText.common.create"),
  delete: tr("uiText.common.delete"),
};

const DatabaseNode: React.FC<DatabaseNodeProps> = ({ data, isConnectable }) => {
  const operation = (data?.operation as string) || 'read';
  const table = (data?.tableName as string) || '';
  const opLabel = OPERATION_LABELS[operation] ?? operation;

  return (
    <BaseNode
      data={data}
      icon={Database}
      colorClass="colorDatabase"
      defaultLabel="uiText.flow.node.database"
      sublabel={table ? `${opLabel}: ${table}` : opLabel}
      isConnectable={isConnectable}
    />
  );
};

export default DatabaseNode;
