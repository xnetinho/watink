import React from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { NodeData } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface StartFormProps {
  formData: NodeData;
  onChange: (field: string, value: unknown) => void;
}

const StartForm: React.FC<StartFormProps> = ({ formData, onChange }) => (
  <div className="space-y-3">
    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.triggerType")}</Label>
      <Select value={formData.triggerType || 'time'} onValueChange={(v) => onChange('triggerType', v)}>
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="time">Tempo/Agendamento</SelectItem>
          <SelectItem value="action">{tr("uiText.flow.form.systemAction")}</SelectItem>
          <SelectItem value="message">Mensagem Recebida</SelectItem>
          <SelectItem value="webhook">Webhook Externo</SelectItem>
        </SelectContent>
      </Select>
    </div>

    {formData.triggerType === 'action' && (
      <div className="space-y-1">
        <Label className="text-xs">{tr("uiText.flow.form.actionType")}</Label>
        <Select value={formData.actionType || 'ticketCreated'} onValueChange={(v) => onChange('actionType', v)}>
          <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="ticketCreated">Ticket Criado</SelectItem>
            <SelectItem value="ticketClosed">Ticket Fechado</SelectItem>
            <SelectItem value="contactCreated">Contato Criado</SelectItem>
            <SelectItem value="queueChanged">{tr("uiText.flow.form.queueChanged")}</SelectItem>
          </SelectContent>
        </Select>
      </div>
    )}

    {formData.triggerType === 'message' && (
      <div className="space-y-1">
        <Label className="text-xs">{tr("uiText.flow.form.connection")}</Label>
        <Select value={formData.whatsappId || 'all'} onValueChange={(v) => onChange('whatsappId', v)}>
          <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{tr("uiText.flow.form.allConnections")}</SelectItem>
          </SelectContent>
        </Select>
      </div>
    )}

    {formData.triggerType === 'webhook' && (
      <div className="space-y-2">
        <p className="text-[10px] text-muted-foreground">{tr("uiText.flow.form.webhookUrl")}</p>
        <Input
          readOnly
          className="h-8 text-xs"
          value={`${window.location.origin}/api/v1/flows/webhook/${formData.webhookToken || 'SEU_TOKEN_AQUI'}`}
        />
        <div className="space-y-1">
          <Label className="text-xs">{tr("uiText.flow.form.webhookToken")}</Label>
          <Input
            className="h-8 text-xs"
            placeholder="meu-token-unico"
            value={formData.webhookToken || ''}
            onChange={(e) => onChange('webhookToken', e.target.value)}
          />
          <p className="text-[10px] text-muted-foreground">{tr("uiText.flow.form.webhookTokenHint")}</p>
        </div>
      </div>
    )}
  </div>
);

export default StartForm;
