import React, { useEffect } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';

import { i18n } from '../../../translate/i18n';

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
export interface Condition {
  id: number;
  field: string;
  operator: string;
  value: string;
  logic?: 'AND' | 'OR';
}

interface ConditionBuilderProps {
  conditions: Condition[];
  onChange: (conditions: Condition[]) => void;
  maxConditions?: number;
  title?: string;
  showLogic?: boolean;
}

const AVAILABLE_FIELDS = [
  { value: 'lastInput', label: 'uiText.flow.condField.lastInput' },
  { value: 'contactName', label: 'uiText.flow.condField.contactName' },
  { value: 'contactNumber', label: 'uiText.flow.condField.contactNumber' },
  { value: 'ticketStatus', label: 'uiText.flow.condField.ticketStatus' },
  { value: 'queueName', label: 'uiText.flow.condField.queueName' },
  { value: 'tagName', label: 'uiText.flow.condField.tagName' },
  { value: 'dayOfWeek', label: 'uiText.flow.condField.dayOfWeek' },
  { value: 'currentHour', label: 'uiText.flow.condField.currentHour' },
];

const OPERATORS = [
  { value: 'equals', label: 'uiText.flow.op.equals', needsValue: true },
  { value: 'notEquals', label: 'uiText.flow.op.notEquals', needsValue: true },
  { value: 'contains', label: 'uiText.flow.op.contains', needsValue: true },
  { value: 'notContains', label: 'uiText.flow.op.notContains', needsValue: true },
  { value: 'startsWith', label: 'uiText.flow.op.startsWith', needsValue: true },
  { value: 'endsWith', label: 'uiText.flow.op.endsWith', needsValue: true },
  { value: 'isEmpty', label: 'uiText.flow.op.isEmpty', needsValue: false },
  { value: 'isNotEmpty', label: 'uiText.flow.op.isNotEmpty', needsValue: false },
  { value: 'greaterThan', label: 'uiText.flow.op.greaterThan', needsValue: true },
  { value: 'lessThan', label: 'uiText.flow.op.lessThan', needsValue: true },
];

const PREDEFINED_VALUES: Record<string, { value: string; label: string }[]> = {
  ticketStatus: [
    { value: 'open', label: 'uiText.flow.condValue.open' },
    { value: 'pending', label: 'uiText.flow.condValue.pending' },
    { value: 'closed', label: 'uiText.flow.condValue.closed' },
  ],
  dayOfWeek: [
    { value: '0', label: 'uiText.flow.weekday.0' },
    { value: '1', label: 'uiText.flow.weekday.1' },
    { value: '2', label: 'uiText.flow.weekday.2' },
    { value: '3', label: 'uiText.flow.weekday.3' },
    { value: '4', label: 'uiText.flow.weekday.4' },
    { value: '5', label: 'uiText.flow.weekday.5' },
    { value: '6', label: 'uiText.flow.weekday.6' },
  ],
};

const ConditionBuilder: React.FC<ConditionBuilderProps> = ({
  conditions = [],
  onChange,
  maxConditions = 5,
  title = tr("uiText.flow.cond.title"),
  showLogic = true,
}) => {
  useEffect(() => {
    if (conditions.length === 0) {
      onChange([{ id: Date.now(), field: 'lastInput', operator: 'isNotEmpty', value: '' }]);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const addCondition = () => {
    if (conditions.length >= maxConditions) return;
    onChange([
      ...conditions,
      { id: Date.now(), field: 'lastInput', operator: 'contains', value: '', logic: 'AND' },
    ]);
  };

  const removeCondition = (id: number) => {
    if (conditions.length <= 1) return;
    onChange(conditions.filter((c) => c.id !== id));
  };

  const updateCondition = (id: number, key: keyof Condition, value: string) => {
    onChange(conditions.map((c) => (c.id === id ? { ...c, [key]: value } : c)));
  };

  const operatorNeedsValue = (op: string) => OPERATORS.find((o) => o.value === op)?.needsValue ?? true;

  return (
    <div className="space-y-2">
      {title && <p className="text-xs font-medium text-muted-foreground mb-2">{title}</p>}

      {conditions.map((condition, index) => (
        <div key={condition.id} className="flex flex-wrap gap-1 items-center">
          {/* Logic selector */}
          {showLogic && index > 0 && (
            <Select
              value={condition.logic ?? 'AND'}
              onValueChange={(v) => updateCondition(condition.id, 'logic', v)}
            >
              <SelectTrigger className="w-[70px] h-8 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="AND">E</SelectItem>
                <SelectItem value="OR">OU</SelectItem>
              </SelectContent>
            </Select>
          )}

          {/* Field selector */}
          <Select
            value={condition.field || 'lastInput'}
            onValueChange={(v) => updateCondition(condition.id, 'field', v)}
          >
            <SelectTrigger className="w-[130px] h-8 text-xs">
              <SelectValue placeholder="Campo" />
            </SelectTrigger>
            <SelectContent>
              {AVAILABLE_FIELDS.map((f) => (
                <SelectItem key={f.value} value={f.value}>
                  {tr(f.label)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* Operator selector */}
          <Select
            value={condition.operator || 'equals'}
            onValueChange={(v) => updateCondition(condition.id, 'operator', v)}
          >
            <SelectTrigger className="w-[120px] h-8 text-xs">
              <SelectValue placeholder="Operador" />
            </SelectTrigger>
            <SelectContent>
              {OPERATORS.map((op) => (
                <SelectItem key={op.value} value={op.value}>
                  {tr(op.label)}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {/* Value input */}
          {operatorNeedsValue(condition.operator) && (
            PREDEFINED_VALUES[condition.field] ? (
              <Select
                value={condition.value || ''}
                onValueChange={(v) => updateCondition(condition.id, 'value', v)}
              >
                <SelectTrigger className="flex-1 min-w-[100px] h-8 text-xs">
                  <SelectValue placeholder="Valor" />
                </SelectTrigger>
                <SelectContent>
                  {PREDEFINED_VALUES[condition.field].map((v) => (
                    <SelectItem key={v.value} value={v.value}>
                      {tr(v.label)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            ) : (
              <Input
                className="flex-1 min-w-[100px] h-8 text-xs"
                placeholder="Valor"
                value={condition.value || ''}
                onChange={(e) => updateCondition(condition.id, 'value', e.target.value)}
              />
            )
          )}

          {/* Remove button */}
          {conditions.length > 1 && (
            <Button
              size="icon"
              variant="ghost"
              className="h-8 w-8 text-destructive hover:text-destructive"
              onClick={() => removeCondition(condition.id)}
            >
              <Trash2 className="h-3 w-3" />
            </Button>
          )}
        </div>
      ))}

      {conditions.length < maxConditions && (
        <Button size="sm" variant="ghost" className="mt-1 h-7 text-xs gap-1" onClick={addCondition}>
          <Plus className="h-3 w-3" />
          {tr("uiText.flow.cond.add")}
        </Button>
      )}
    </div>
  );
};

export default ConditionBuilder;
