import React from 'react';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { NodeData } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const FIELD_GROUPS = [
  { key: 'contactFields', label: "uiText.flow.form.contactData", fields: ['name', 'number', 'email', 'profilePicUrl', 'id'] },
  { key: 'ticketFields', label: "uiText.flow.form.ticketData", fields: ['id', 'status', 'queueId', 'userId', 'lastMessage', 'chatbot'] },
  { key: 'pipelineFields', label: "uiText.flow.form.pipelineData", fields: ['dealTitle', 'dealValue', 'pipelineName', 'stageName', 'dealId', 'priority'] },
];

interface WebhookFormProps {
  formData: NodeData;
  onChange: (field: string, value: unknown) => void;
}

const WebhookForm: React.FC<WebhookFormProps> = ({ formData, onChange }) => (
  <div className="space-y-3">
    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.method")}</Label>
      <Select value={formData.method || 'POST'} onValueChange={(v) => onChange('method', v)}>
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          {['GET', 'POST', 'PUT', 'DELETE', 'PATCH'].map((m) => (
            <SelectItem key={m} value={m}>{m}</SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>

    <div className="space-y-1">
      <Label className="text-xs">URL</Label>
      <Input className="h-8 text-xs" placeholder="https://api.exemplo.com/webhook" value={formData.url || ''} onChange={(e) => onChange('url', e.target.value)} />
    </div>

    <div className="space-y-1">
      <p className="text-xs font-medium">Headers (JSON)</p>
      <Textarea className="text-xs min-h-[60px]" placeholder='{ "Authorization": "Bearer 123" }' value={formData.headers || ''} onChange={(e) => onChange('headers', e.target.value)} />
    </div>

    <div className="space-y-1">
      <p className="text-xs font-medium">Body (JSON)</p>
      <Textarea className="text-xs min-h-[100px]" placeholder='{ "nome": "{{contact.name}}" }' value={formData.body || ''} onChange={(e) => onChange('body', e.target.value)} />
    </div>

    {FIELD_GROUPS.map(({ key, label, fields }) => (
      <div key={key} className="space-y-1">
        <p className="text-xs font-medium">{tr(label)}</p>
        <div className="flex flex-wrap gap-x-3 gap-y-1">
          {fields.map((name) => {
            const selected = ((formData[key] as string[]) || []).includes(name);
            return (
              <div key={name} className="flex items-center gap-1">
                <Checkbox
                  id={`${key}-${name}`}
                  checked={selected}
                  onCheckedChange={(checked) => {
                    const current = (formData[key] as string[]) || [];
                    onChange(key, checked ? [...current, name] : current.filter((f) => f !== name));
                  }}
                />
                <Label htmlFor={`${key}-${name}`} className="text-xs cursor-pointer">{name}</Label>
              </div>
            );
          })}
        </div>
      </div>
    ))}

    <div className="space-y-1">
      <div className="flex items-center gap-2">
        <Checkbox id="includeContext" checked={!!formData.includeContext} onCheckedChange={(v) => onChange('includeContext', v)} />
        <Label htmlFor="includeContext" className="text-xs cursor-pointer">{tr("uiText.flow.form.includeContext")}</Label>
      </div>
      <div className="flex items-center gap-2">
        <Checkbox id="fullData" checked={!!formData.fullData} onCheckedChange={(v) => onChange('fullData', v)} />
        <Label htmlFor="fullData" className="text-xs cursor-pointer">{tr("uiText.flow.form.sendAll")}</Label>
      </div>
    </div>
  </div>
);

export default WebhookForm;
