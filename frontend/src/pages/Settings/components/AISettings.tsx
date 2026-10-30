/* @jsxImportSource react */
import React from "react";
import { Brain, Database, MessageSquare, Captions } from "lucide-react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from "../../../components/ui/card";
import { Label } from "../../../components/ui/label";
import { Switch } from "../../../components/ui/switch";
import { Separator } from "../../../components/ui/separator";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../components/ui/select";
import { Input } from "../../../components/ui/input";
import { Textarea } from "../../../components/ui/textarea";
import { shouldSaveSecret } from "../utils/secretSettings";
import {
  listAiGateways,
  listAiGatewayModels,
  AiGateway,
} from "../../../services/aiGatewayService";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface AISettingsProps {
  getSettingValue: (key: string) => string;
  handleUpdateSetting: (key: string, value: string) => Promise<void>;
}

const isTranscriptionModelName = (id: string) =>
  /whisper|transcri|scribe|\bstt\b/i.test(id);

const providerModels: Record<string, string[]> = {
  openai: ["gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"],
  grok: ["grok-beta", "grok-2"],
  anthropic: ["claude-3-5-sonnet", "claude-3-opus", "claude-3-haiku"],
  custom: [],
};

const AISettings: React.FC<AISettingsProps> = ({
  getSettingValue,
  handleUpdateSetting,
}) => {
  const provider = getSettingValue("aiProvider") || "openai";
  const models = providerModels[provider] || [];
  const isCustomProvider = provider === "custom";

  // O embedding usa um gateway DEDICADO quando há uma URL base própria configurada;
  // caso contrário, reusa o gateway do chat (apenas com um modelo de embedding próprio).
  const [embeddingDedicated, setEmbeddingDedicated] = React.useState<boolean>(
    Boolean(getSettingValue("aiEmbeddingBaseURL"))
  );

  const handleToggleDedicated = (checked: boolean): void => {
    setEmbeddingDedicated(checked);
    if (!checked) {
      // Voltar a usar o gateway do chat → limpa o endpoint/chave dedicados.
      void handleUpdateSetting("aiEmbeddingBaseURL", "");
      void handleUpdateSetting("aiEmbeddingApiKey", "");
    }
  };

  // ── Transcrição de áudio (sob demanda, botão na bolha) ──────────────
  const transcriptionEnabled = getSettingValue("audioTranscriptionEnabled") === "true";
  const transcriptionGatewayId = getSettingValue("audioTranscriptionGatewayId");
  const [aiGateways, setAiGateways] = React.useState<AiGateway[]>([]);
  const [transcriptionModels, setTranscriptionModels] = React.useState<string[]>([]);

  React.useEffect(() => {
    if (!transcriptionEnabled) return;
    listAiGateways()
      .then(setAiGateways)
      .catch(() => setAiGateways([]));
  }, [transcriptionEnabled]);

  React.useEffect(() => {
    if (!transcriptionGatewayId) {
      setTranscriptionModels([]);
      return;
    }
    listAiGatewayModels(Number(transcriptionGatewayId))
      .then((res) =>
        setTranscriptionModels(
          res.success ? (res.models ?? []).filter(isTranscriptionModelName) : []
        )
      )
      .catch(() => setTranscriptionModels([]));
  }, [transcriptionGatewayId]);

  return (
    <div className="space-y-6">
      {/* ── Funcionalidades de IA ───────────────────────────── */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-primary">
            <Brain className="h-5 w-5" />
            {tr("uiText.settings.ai.title")}
          </CardTitle>
          <CardDescription>
            {tr("uiText.settings.ai.subtitle")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>Habilitar IA Global</Label>
                <p className="text-xs text-muted-foreground">
                  {tr("uiText.settings.ai.onboardingBot")}
                </p>
              </div>
              <Switch
                checked={getSettingValue("aiEnabled") === "true"}
                onCheckedChange={(checked) =>
                  handleUpdateSetting("aiEnabled", checked ? "true" : "false")
                }
              />
            </div>

            <Separator />

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>{tr("uiText.settings.ai.flowbuilder")}</Label>
                <p className="text-xs text-muted-foreground">
                  {tr("uiText.settings.ai.flowbuilderHint")}
                </p>
              </div>
              <Switch
                checked={getSettingValue("aiFlowBuilderEnabled") === "true"}
                onCheckedChange={(checked) =>
                  handleUpdateSetting(
                    "aiFlowBuilderEnabled",
                    checked ? "true" : "false"
                  )
                }
              />
            </div>

            <Separator />

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>{tr("uiText.settings.ai.pipeline")}</Label>
                <p className="text-xs text-muted-foreground">
                  {tr("uiText.settings.ai.pipelineHint")}
                </p>
              </div>
              <Switch
                checked={getSettingValue("aiPipelineEnabled") === "true"}
                onCheckedChange={(checked) =>
                  handleUpdateSetting(
                    "aiPipelineEnabled",
                    checked ? "true" : "false"
                  )
                }
              />
            </div>

            <Separator />

            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>{tr("uiText.settings.ai.copilot")}</Label>
                <p className="text-xs text-muted-foreground">
                  {tr("uiText.settings.ai.copilotHint")}
                </p>
              </div>
              <Switch
                checked={getSettingValue("aiAssistantEnabled") === "true"}
                onCheckedChange={(checked) =>
                  handleUpdateSetting(
                    "aiAssistantEnabled",
                    checked ? "true" : "false"
                  )
                }
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* ── Transcrição de Áudio ─────────────────────────────── */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-primary">
            <Captions className="h-5 w-5" />
            {tr("uiText.settings.ai.transcription")}
          </CardTitle>
          <CardDescription>
            {tr("uiText.settings.ai.transcriptionHint")}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label>{tr("uiText.settings.ai.enableTranscription")}</Label>
                <p className="text-xs text-muted-foreground">
                  {tr("uiText.settings.ai.enableTranscriptionHint")}
                </p>
              </div>
              <Switch
                checked={transcriptionEnabled}
                onCheckedChange={(checked) =>
                  handleUpdateSetting(
                    "audioTranscriptionEnabled",
                    checked ? "true" : "false"
                  )
                }
              />
            </div>

            {transcriptionEnabled && (
              <>
                <Separator />
                <div className="grid gap-4 md:grid-cols-2">
                  <div className="grid gap-2">
                    <Label htmlFor="audio-transcription-gateway">{tr("uiText.settings.ai.gateway")}</Label>
                    <Select
                      value={transcriptionGatewayId || undefined}
                      onValueChange={(v) =>
                        handleUpdateSetting("audioTranscriptionGatewayId", v)
                      }
                    >
                      <SelectTrigger id="audio-transcription-gateway">
                        <SelectValue placeholder={tr("uiText.settings.ai.selectGateway")} />
                      </SelectTrigger>
                      <SelectContent>
                        {aiGateways.map((g) => (
                          <SelectItem key={g.id} value={String(g.id)}>
                            {g.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <p className="text-xs text-muted-foreground">
                      Configurado em Configurações → Agentes de IA (Assistentes).
                    </p>
                  </div>

                  <div className="grid gap-2">
                    <Label htmlFor="audio-transcription-model">{tr("uiText.settings.ai.transcriptionModel")}</Label>
                    <Input
                      id="audio-transcription-model"
                      list="audio-transcription-model-suggestions"
                      placeholder="whisper-1"
                      defaultValue={getSettingValue("audioTranscriptionModel")}
                      onBlur={(e) =>
                        handleUpdateSetting("audioTranscriptionModel", e.target.value)
                      }
                    />
                    <datalist id="audio-transcription-model-suggestions">
                      {transcriptionModels.map((m) => (
                        <option key={m} value={m} />
                      ))}
                    </datalist>
                    <p className="text-xs text-muted-foreground">
                      Vazio = usa o modelo de transcrição já configurado no Gateway.
                    </p>
                  </div>
                </div>
              </>
            )}
          </div>
        </CardContent>
      </Card>

      {/* ── Gateway de Chat / LLM ───────────────────────────── */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-primary">
            <MessageSquare className="h-5 w-5" />
            {tr("uiText.settings.ai.chatGateway")}
          </CardTitle>
          <CardDescription>
            {tr("uiText.settings.ai.chatGatewayHint")}
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="grid gap-4 md:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="ai-provider">{tr("uiText.settings.ai.provider")}</Label>
              <Select
                value={provider}
                onValueChange={(v) => {
                  handleUpdateSetting("aiProvider", v);
                  const firstModel = providerModels[v]?.[0] || "";
                  if (firstModel) {
                    handleUpdateSetting("aiModel", firstModel);
                  }
                }}
              >
                <SelectTrigger id="ai-provider">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="openai">OpenAI (GPT-4/GPT-3.5)</SelectItem>
                  <SelectItem value="grok">xAI Grok (Grok-2/Grok-Beta)</SelectItem>
                  <SelectItem value="anthropic">Anthropic (Claude 3.5)</SelectItem>
                  <SelectItem value="custom">
                    Provedor Customizado (OpenAI-compatível)
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="ai-model">{tr("uiText.settings.ai.llmModel")}</Label>
              {isCustomProvider ? (
                <Input
                  id="ai-model"
                  placeholder="Ex: llama3, mistral, gpt-4o-mini"
                  defaultValue={getSettingValue("aiModel")}
                  onBlur={(e) => handleUpdateSetting("aiModel", e.target.value)}
                />
              ) : (
                <Select
                  value={getSettingValue("aiModel") || models[0] || ""}
                  onValueChange={(v) => handleUpdateSetting("aiModel", v)}
                  disabled={models.length === 0}
                >
                  <SelectTrigger id="ai-model">
                    <SelectValue placeholder="Selecione o modelo" />
                  </SelectTrigger>
                  <SelectContent>
                    {models.map((mod) => (
                      <SelectItem key={mod} value={mod}>
                        {mod}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
          </div>

          {isCustomProvider && (
            <div className="grid gap-2">
              <Label htmlFor="ai-base-url">{tr("uiText.settings.ai.baseUrl")}</Label>
              <Input
                id="ai-base-url"
                placeholder="http://localhost:20128/v1"
                defaultValue={getSettingValue("aiCustomBaseURL")}
                onBlur={(e) =>
                  handleUpdateSetting("aiCustomBaseURL", e.target.value)
                }
              />
            </div>
          )}

          <div className="grid gap-2">
            <Label htmlFor="ai-key">{tr("uiText.settings.ai.apiKey")}</Label>
            <Input
              id="ai-key"
              type="password"
              placeholder="sk-..."
              defaultValue={getSettingValue("aiApiKey")}
              onBlur={(e) =>
                shouldSaveSecret(e.target.value, getSettingValue("aiApiKey")) &&
                handleUpdateSetting("aiApiKey", e.target.value)
              }
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="ai-prompt">{tr("uiText.settings.ai.guidePrompt")}</Label>
            <Textarea
              id="ai-prompt"
              rows={6}
              placeholder={tr("uiText.settings.ai.guidePlaceholder")}
              defaultValue={getSettingValue("aiGuidePrompt")}
              onBlur={(e) => handleUpdateSetting("aiGuidePrompt", e.target.value)}
            />
          </div>
        </CardContent>
      </Card>

      {/* ── Gateway de Embedding (RAG) ──────────────────────── */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-primary">
            <Database className="h-5 w-5" />
            Gateway de Embedding (Base de Conhecimento / RAG)
          </CardTitle>
          <CardDescription>
            {tr("uiText.settings.ai.embeddingHint")}
          </CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4">
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>{tr("uiText.settings.ai.embeddingGateway")}</Label>
              <p className="text-xs text-muted-foreground">
                Desligado: usa o gateway do chat acima. Ligado: endpoint próprio
                (ex.: Ollama self-hosted em http://ollama:11434/v1).
              </p>
            </div>
            <Switch
              checked={embeddingDedicated}
              onCheckedChange={handleToggleDedicated}
            />
          </div>

          {embeddingDedicated && (
            <div className="grid gap-4 md:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="ai-embedding-base-url">
                  {tr("uiText.settings.ai.embeddingUrl")}
                </Label>
                <Input
                  id="ai-embedding-base-url"
                  placeholder="http://ollama:11434/v1"
                  defaultValue={getSettingValue("aiEmbeddingBaseURL")}
                  onBlur={(e) =>
                    handleUpdateSetting("aiEmbeddingBaseURL", e.target.value)
                  }
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="ai-embedding-key">
                  Chave da API do Embedding (opcional)
                </Label>
                <Input
                  id="ai-embedding-key"
                  type="password"
                  placeholder={tr("uiText.settings.ai.embeddingKeyHint")}
                  defaultValue={getSettingValue("aiEmbeddingApiKey")}
                  onBlur={(e) =>
                    shouldSaveSecret(e.target.value, getSettingValue("aiEmbeddingApiKey")) &&
                    handleUpdateSetting("aiEmbeddingApiKey", e.target.value)
                  }
                />
              </div>
            </div>
          )}

          <div className="grid gap-2">
            <Label htmlFor="ai-embedding-model">{tr("uiText.settings.ai.embeddingModel")}</Label>
            <Input
              id="ai-embedding-model"
              placeholder={tr("uiText.settings.ai.embeddingModelPlaceholder")}
              defaultValue={getSettingValue("aiEmbeddingModel")}
              onBlur={(e) =>
                handleUpdateSetting("aiEmbeddingModel", e.target.value)
              }
            />
            <p className="text-xs text-muted-foreground">
              {tr("uiText.settings.ai.dimensionHint")}
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default AISettings;
