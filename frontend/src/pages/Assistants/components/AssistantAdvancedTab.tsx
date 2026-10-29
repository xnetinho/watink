/* @jsxImportSource react */
import React from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";

interface AssistantAdvancedTabProps {
  sessionExpiryMinutes: string;
  onSessionExpiryMinutesChange: (v: string) => void;
  typingDelayMs: string;
  onTypingDelayMsChange: (v: string) => void;
  debounceSeconds: string;
  onDebounceSecondsChange: (v: string) => void;
  endKeyword: string;
  onEndKeywordChange: (v: string) => void;
  expiryMessage: string;
  onExpiryMessageChange: (v: string) => void;
  closingMessage: string;
  onClosingMessageChange: (v: string) => void;
  stopOnHumanReply: boolean;
  onStopOnHumanReplyChange: (v: boolean) => void;
  groupsMode: "legacy" | "selective";
  ignoreGroups: boolean;
  onIgnoreGroupsChange: (v: boolean) => void;
  acceptsAudio: boolean;
  onAcceptsAudioChange: (v: boolean) => void;
  respondsWithAudio: boolean;
  onRespondsWithAudioChange: (v: boolean) => void;
}

export const AssistantAdvancedTab: React.FC<AssistantAdvancedTabProps> = ({
  sessionExpiryMinutes,
  onSessionExpiryMinutesChange,
  typingDelayMs,
  onTypingDelayMsChange,
  debounceSeconds,
  onDebounceSecondsChange,
  endKeyword,
  onEndKeywordChange,
  expiryMessage,
  onExpiryMessageChange,
  closingMessage,
  onClosingMessageChange,
  stopOnHumanReply,
  onStopOnHumanReplyChange,
  groupsMode,
  ignoreGroups,
  onIgnoreGroupsChange,
  acceptsAudio,
  onAcceptsAudioChange,
  respondsWithAudio,
  onRespondsWithAudioChange,
}) => {
  return (
    <div className="flex flex-col gap-4 pt-2">
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="flex flex-col gap-1.5">
          <Label>Expiração de sessão (min)</Label>
          <Input
            type="number"
            value={sessionExpiryMinutes}
            onChange={(e) => onSessionExpiryMinutesChange(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Atraso de digitação (ms)</Label>
          <Input
            type="number"
            value={typingDelayMs}
            onChange={(e) => onTypingDelayMsChange(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Debounce (s)</Label>
          <Input
            type="number"
            value={debounceSeconds}
            onChange={(e) => onDebounceSecondsChange(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1.5">
          <Label>Palavra para encerrar</Label>
          <Input
            value={endKeyword}
            onChange={(e) => onEndKeywordChange(e.target.value)}
            placeholder="Ex: sair"
          />
        </div>
      </div>
      <div className="flex flex-col gap-1.5">
        <Label>Mensagem de expiração</Label>
        <Textarea
          value={expiryMessage}
          onChange={(e) => onExpiryMessageChange(e.target.value)}
          rows={2}
        />
      </div>
      <div className="flex flex-col gap-1.5">
        <Label>Mensagem de encerramento</Label>
        <Textarea
          value={closingMessage}
          onChange={(e) => onClosingMessageChange(e.target.value)}
          rows={2}
        />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="flex items-center justify-between rounded-xl border p-3">
          <p className="text-sm font-medium">Parar assistente quando eu responder</p>
          <Switch checked={stopOnHumanReply} onCheckedChange={onStopOnHumanReplyChange} />
        </div>
        {groupsMode === "legacy" && (
          <div className="flex items-center justify-between rounded-xl border p-3">
            <p className="text-sm font-medium">Ignorar grupos</p>
            <Switch checked={ignoreGroups} onCheckedChange={onIgnoreGroupsChange} />
          </div>
        )}
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="flex items-center justify-between rounded-xl border p-3">
          <div>
            <p className="text-sm font-medium">Entende áudio</p>
            <p className="text-xs text-muted-foreground">
              Transcreve mensagens de voz recebidas via o AiGateway configurado.
              Se desligado, responde avisando que não entende áudio.
            </p>
          </div>
          <Switch checked={acceptsAudio} onCheckedChange={onAcceptsAudioChange} />
        </div>
        <div className="flex items-center justify-between rounded-xl border p-3">
          <div>
            <p className="text-sm font-medium">Responde com áudio</p>
            <p className="text-xs text-muted-foreground">
              Sintetiza as respostas como mensagem de voz em vez de texto.
            </p>
          </div>
          <Switch checked={respondsWithAudio} onCheckedChange={onRespondsWithAudioChange} />
        </div>
      </div>
    </div>
  );
};
