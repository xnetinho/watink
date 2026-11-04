export interface Filter {
  id: number;
  field: string;
  operator: string;
  value: string;
  logic?: 'AND' | 'OR';
}

export interface FilterBuilderProps {
  filters: Filter[];
  onChange: (filters: Filter[]) => void;
  tableName?: string;
  maxFilters?: number;
}

export interface FieldConfig {
  value: string;
  label: string;
  type: 'number' | 'string' | 'boolean' | 'select';
  options?: string[];
}

export const TABLE_FIELDS: Record<string, FieldConfig[]> = {
  Contacts: [
    { value: 'id', label: 'uiText.flow.field.id', type: 'number' },
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
    { value: 'number', label: 'uiText.flow.field.number', type: 'string' },
    { value: 'email', label: 'uiText.flow.field.email', type: 'string' },
    { value: 'isGroup', label: 'uiText.flow.field.isGroup', type: 'boolean' },
  ],
  Tickets: [
    { value: 'id', label: 'uiText.flow.field.id', type: 'number' },
    { value: 'status', label: 'uiText.flow.field.status', type: 'select', options: ['open', 'pending', 'closed'] },
    { value: 'queueId', label: 'uiText.flow.field.queueId', type: 'number' },
    { value: 'userId', label: 'uiText.flow.field.userId', type: 'number' },
    { value: 'contactId', label: 'uiText.flow.field.contactId', type: 'number' },
    { value: 'isGroup', label: 'uiText.flow.field.isGroup', type: 'boolean' },
  ],
  Messages: [
    { value: 'id', label: 'uiText.flow.field.id', type: 'number' },
    { value: 'body', label: 'uiText.flow.field.body', type: 'string' },
    { value: 'fromMe', label: 'uiText.flow.field.fromMe', type: 'boolean' },
    { value: 'mediaType', label: 'uiText.flow.field.mediaType', type: 'string' },
    { value: 'ticketId', label: 'uiText.flow.field.ticketId', type: 'number' },
  ],
  Users: [
    { value: 'id', label: 'uiText.flow.field.id', type: 'number' },
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
    { value: 'email', label: 'uiText.flow.field.email', type: 'string' },
    { value: 'profile', label: 'uiText.flow.field.profile', type: 'select', options: ['admin', 'user', 'super'] },
  ],
  Queues: [
    { value: 'id', label: 'uiText.flow.field.id', type: 'number' },
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
    { value: 'color', label: 'uiText.flow.field.color', type: 'string' },
  ],
  Whatsapps: [
    { value: 'id', label: 'uiText.flow.field.id', type: 'number' },
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
    { value: 'status', label: 'uiText.flow.field.status', type: 'select', options: ['CONNECTED', 'DISCONNECTED', 'OPENING'] },
    { value: 'isDefault', label: 'uiText.flow.field.isDefault', type: 'boolean' },
  ],
  QuickAnswers: [
    { value: 'id', label: 'uiText.flow.field.id', type: 'number' },
    { value: 'shortcut', label: 'uiText.flow.field.shortcut', type: 'string' },
    { value: 'message', label: 'uiText.flow.field.message', type: 'string' },
  ],
  Pipelines: [
    { value: 'id', label: 'uiText.flow.field.id', type: 'number' },
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
  ],
};

export const FILTER_OPERATORS = [
  { value: '=', label: 'uiText.flow.op.eq' },
  { value: '!=', label: 'uiText.flow.op.neq' },
  { value: '>', label: 'uiText.flow.op.gt' },
  { value: '<', label: 'uiText.flow.op.lt' },
  { value: '>=', label: 'uiText.flow.op.gte' },
  { value: '<=', label: 'uiText.flow.op.lte' },
  { value: 'like', label: 'uiText.flow.op.like' },
];

export const CONTEXT_VARIABLES = [
  { value: '{{contactId}}', label: 'uiText.flow.field.contactId' },
  { value: '{{ticketId}}', label: 'uiText.flow.field.ticketId' },
  { value: '{{userId}}', label: 'uiText.flow.field.userId' },
  { value: '{{queueId}}', label: 'uiText.flow.field.queueId' },
  { value: '{{lastInput}}', label: 'uiText.flow.field.lastInput' },
];
