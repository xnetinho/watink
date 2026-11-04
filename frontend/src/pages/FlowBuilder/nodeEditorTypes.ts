import type { Condition } from './components/ConditionBuilder';
import type { Filter } from './components/FilterBuilder';
import type { DataField } from './components/DataBuilder';

export interface NodeData {
  label?: string;
  triggerType?: string;
  actionType?: string;
  whatsappId?: string;
  webhookToken?: string;
  conditions?: Condition[];
  conditionsA?: Condition[];
  contentType?: string;
  content?: string;
  mediaUrl?: string;
  menuTitle?: string;
  options?: { id: string; label: string }[];
  kanbanAction?: string;
  dealTitle?: string;
  dealValue?: string;
  dealPriority?: string;
  pipelineId?: string;
  stageId?: string;
  ticketAction?: string;
  newStatus?: string;
  queueId?: string;
  userId?: string;
  method?: string;
  url?: string;
  headers?: string;
  body?: string;
  contactFields?: string[];
  ticketFields?: string[];
  pipelineFields?: string[];
  includeContext?: boolean;
  fullData?: boolean;
  resultVariable?: string;
  responseMode?: string;
  knowledgeBaseId?: string;
  persona?: string;
  maxTurns?: number;
  helpdeskAction?: string;
  subject?: string;
  description?: string;
  priority?: string;
  category?: string;
  endAction?: string;
  endMessage?: string;
  operation?: string;
  tableName?: string;
  filters?: Filter[];
  dataFields?: DataField[];
  selectedFields?: string[];
  limit?: number;
  orderByField?: string;
  orderByDir?: string;
  outputVariable?: string;
  inputVariable?: string;
  filterConditions?: FilterCondition[];
  quickAnswerId?: string;
  [key: string]: unknown;
}

export interface FilterCondition {
  id: number;
  field: string;
  operator: string;
  value: string;
  logic?: 'AND' | 'OR';
}

export interface FlowNode {
  id: string;
  type: string;
  data: NodeData;
}

export interface Pipeline {
  id: string;
  name: string;
  stages?: { id: string; name: string }[];
}

export interface Queue {
  id: string;
  name: string;
  color?: string;
}

export interface User {
  id: string;
  name: string;
}

export interface KnowledgeBase {
  id: string;
  name: string;
}

export interface QuickAnswer {
  id: string;
  shortcut: string;
  message: string;
}

export interface NodeEditorSidebarProps {
  open: boolean;
  node: FlowNode | null;
  onClose: () => void;
  onSave: (nodeId: string, data: NodeData) => void;
  onDelete: (nodeId: string) => void;
}

export const NODE_TITLES: Record<string, string> = {
  start: "uiText.flow.editor.title.start",
  input: "uiText.flow.editor.title.start",
  trigger: 'uiText.flow.editor.title.trigger',
  message: 'uiText.flow.editor.title.message',
  menu: 'uiText.flow.editor.title.menu',
  switch: "uiText.flow.editor.title.switch",
  pipeline: 'uiText.flow.editor.title.pipeline',
  ticket: 'uiText.flow.editor.title.ticket',
  webhook: 'uiText.flow.editor.title.webhook',
  knowledge: 'uiText.flow.editor.title.knowledge',
  agent: 'uiText.flow.editor.title.agent',
  database: 'uiText.flow.editor.title.database',
  filter: "uiText.flow.editor.title.filter",
  api: "uiText.flow.editor.title.api",
  helpdesk: 'uiText.flow.editor.title.helpdesk',
  end: 'uiText.flow.editor.title.end',
  output: 'uiText.flow.editor.title.end',
  quickAnswer: 'flowBuilder.nodes.quickAnswer.title',
};

export const TABLE_ALL_FIELDS: Record<string, string[]> = {
  Contacts: ['id', 'name', 'number', 'email', 'isGroup', 'profilePicUrl', 'createdAt'],
  Tickets: ['id', 'status', 'queueId', 'userId', 'contactId', 'isGroup', 'createdAt', 'updatedAt'],
  Messages: ['id', 'body', 'fromMe', 'mediaType', 'ticketId', 'createdAt'],
  Users: ['id', 'name', 'email', 'profile', 'createdAt'],
  Queues: ['id', 'name', 'color', 'createdAt'],
  Whatsapps: ['id', 'name', 'status', 'isDefault', 'createdAt'],
  QuickAnswers: ['id', 'shortcut', 'message', 'createdAt'],
  Pipelines: ['id', 'name', 'createdAt'],
};

export const FILTERABLE_FIELDS = [
  { value: 'id', label: 'uiText.flow.field.id' },
  { value: 'name', label: 'uiText.flow.field.name' },
  { value: 'status', label: 'uiText.flow.field.status' },
  { value: 'email', label: 'uiText.flow.field.email' },
  { value: 'number', label: 'uiText.flow.field.number' },
  { value: 'body', label: 'uiText.flow.field.body' },
  { value: 'queueId', label: 'uiText.flow.field.queueId' },
  { value: 'userId', label: 'uiText.flow.field.userId' },
  { value: 'createdAt', label: 'uiText.flow.field.createdAt' },
];

export const FILTER_OPERATORS_NODE = [
  { value: 'equals', label: 'uiText.flow.op.equals' },
  { value: 'notEquals', label: 'uiText.flow.op.notEquals' },
  { value: 'contains', label: 'uiText.flow.op.contains' },
  { value: 'notContains', label: 'uiText.flow.op.notContains' },
  { value: 'startsWith', label: 'uiText.flow.op.startsWith' },
  { value: 'endsWith', label: 'uiText.flow.op.endsWith' },
  { value: 'greaterThan', label: 'uiText.flow.op.greaterThan' },
  { value: 'lessThan', label: 'uiText.flow.op.lessThan' },
  { value: 'isEmpty', label: 'uiText.flow.op.isEmpty' },
  { value: 'isNotEmpty', label: 'uiText.flow.op.isNotEmpty' },
];
