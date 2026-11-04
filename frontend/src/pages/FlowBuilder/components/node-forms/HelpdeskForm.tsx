import React from 'react';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { NodeData } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface HelpdeskFormProps {
  formData: NodeData;
  onChange: (field: string, value: unknown) => void;
}

const HelpdeskForm: React.FC<HelpdeskFormProps> = ({ formData, onChange }) => (
  <div className="space-y-3">
    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.helpdeskAction")}</Label>
      <Select value={formData.helpdeskAction || 'createProtocol'} onValueChange={(v) => onChange('helpdeskAction', v)}>
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="createProtocol">Criar Protocolo</SelectItem>
          <SelectItem value="checkStatus">Verificar Status</SelectItem>
        </SelectContent>
      </Select>
    </div>

    {formData.helpdeskAction === 'createProtocol' && (
      <>
        <div className="space-y-1">
          <Label className="text-xs">Assunto</Label>
          <Input className="h-8 text-xs" placeholder="Protocolo via Fluxo" value={formData.subject || ''} onChange={(e) => onChange('subject', e.target.value)} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{tr("uiText.common.description")}</Label>
          <Textarea className="text-xs min-h-[60px]" placeholder={tr("uiText.flow.form.protocolDetails")} value={formData.description || ''} onChange={(e) => onChange('description', e.target.value)} />
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{tr("uiText.common.priority")}</Label>
          <Select value={formData.priority || 'medium'} onValueChange={(v) => onChange('priority', v)}>
            <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="low">{tr("uiText.common.low")}</SelectItem>
              <SelectItem value="medium">{tr("uiText.common.medium")}</SelectItem>
              <SelectItem value="high">{tr("uiText.common.high")}</SelectItem>
              <SelectItem value="urgent">{tr("uiText.common.urgent")}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="space-y-1">
          <Label className="text-xs">{tr("uiText.common.category")}</Label>
          <Input className="h-8 text-xs" placeholder="Fluxo Automatizado" value={formData.category || ''} onChange={(e) => onChange('category', e.target.value)} />
        </div>
      </>
    )}
  </div>
);

export default HelpdeskForm;
