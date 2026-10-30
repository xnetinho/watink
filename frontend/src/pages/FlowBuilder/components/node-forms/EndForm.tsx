import React from 'react';
import { Textarea } from '@/components/ui/textarea';
import { Label } from '@/components/ui/label';
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select';
import { NodeData } from '../../nodeEditorTypes';

import { i18n } from "../../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface EndFormProps {
  formData: NodeData;
  onChange: (field: string, value: unknown) => void;
}

const EndForm: React.FC<EndFormProps> = ({ formData, onChange }) => (
  <div className="space-y-3">
    <div className="space-y-1">
      <Label className="text-xs">{tr("uiText.flow.form.endAction")}</Label>
      <Select value={formData.endAction || 'none'} onValueChange={(v) => onChange('endAction', v)}>
        <SelectTrigger className="h-8 text-xs"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectItem value="none">Apenas Finalizar</SelectItem>
          <SelectItem value="closeTicket">Fechar Ticket</SelectItem>
          <SelectItem value="transferQueue">{tr("uiText.flow.form.transferQueue")}</SelectItem>
          <SelectItem value="sendMessage">Enviar Mensagem Final</SelectItem>
        </SelectContent>
      </Select>
    </div>

    {formData.endAction === 'sendMessage' && (
      <Textarea
        className="text-xs min-h-[60px]"
        placeholder="Obrigado pelo contato!"
        value={formData.endMessage || ''}
        onChange={(e) => onChange('endMessage', e.target.value)}
      />
    )}
  </div>
);

export default EndForm;
