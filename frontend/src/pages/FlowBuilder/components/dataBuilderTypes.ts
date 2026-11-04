export interface DataField {
  id: number;
  field: string;
  value: string;
  useVariable: boolean;
}

export interface DataBuilderProps {
  dataFields: DataField[];
  onChange: (fields: DataField[]) => void;
  tableName?: string;
  maxFields?: number;
}

export interface FieldConfig {
  value: string;
  label: string;
  type: 'string' | 'boolean' | 'select' | 'variable';
  options?: string[];
}

export const EDITABLE_FIELDS: Record<string, FieldConfig[]> = {
  Contacts: [
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
    { value: 'email', label: 'uiText.flow.field.email', type: 'string' },
    { value: 'profilePicUrl', label: 'uiText.flow.field.profilePicUrl', type: 'string' },
  ],
  Tickets: [
    { value: 'status', label: 'uiText.flow.field.status', type: 'select', options: ['open', 'pending', 'closed'] },
    { value: 'queueId', label: 'uiText.flow.field.queueId', type: 'variable' },
    { value: 'userId', label: 'uiText.flow.field.userId', type: 'variable' },
  ],
  Messages: [
    { value: 'body', label: 'uiText.flow.field.body', type: 'string' },
    { value: 'read', label: 'uiText.flow.field.read', type: 'boolean' },
  ],
  Users: [
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
    { value: 'email', label: 'uiText.flow.field.email', type: 'string' },
  ],
  Queues: [
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
    { value: 'color', label: 'uiText.flow.field.color', type: 'string' },
  ],
  Whatsapps: [
    { value: 'name', label: 'uiText.flow.field.name', type: 'string' },
    { value: 'isDefault', label: 'uiText.flow.field.isDefault', type: 'boolean' },
  ],
  QuickAnswers: [
    { value: 'shortcut', label: 'uiText.flow.field.shortcut', type: 'string' },
    { value: 'message', label: 'uiText.flow.field.message', type: 'string' },
  ],
  Pipelines: [{ value: 'name', label: 'uiText.flow.field.name', type: 'string' }],
};

export const CONTEXT_VARIABLES = [
  { value: '{{contactId}}', label: 'uiText.flow.field.contactId' },
  { value: '{{ticketId}}', label: 'uiText.flow.field.ticketId' },
  { value: '{{userId}}', label: 'uiText.flow.field.userId' },
  { value: '{{queueId}}', label: 'uiText.flow.field.queueId' },
  { value: '{{contactName}}', label: 'uiText.flow.field.contactName' },
  { value: '{{lastInput}}', label: 'uiText.flow.field.lastInput' },
];
