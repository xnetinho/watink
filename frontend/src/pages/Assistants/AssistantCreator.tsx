import React, { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { toast } from "react-toastify";
import { Save, PlayCircle } from "lucide-react";
import { PageContainer, PageHeader, PageContent } from "@/components/ui/page-layout";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import api from "../../services/api";
import {
    Assistant,
    AssistantMode,
    AssistantTestResult,
    createAssistant,
    getAssistant,
    testAssistant,
    updateAssistant,
} from "../../services/assistantService";
import { listAiGateways, AiGateway } from "../../services/aiGatewayService";
import AssistantRouterOptions from "./AssistantRouterOptions";
import AssistantGroupsPanel from "./AssistantGroupsPanel";
import { PersonaSectionValues, buildPersonaFromSections, parsePersonaSections } from "./personaSections";
import { AssistantTestDialog } from "./components/AssistantTestDialog";
import { AssistantAdvancedTab } from "./components/AssistantAdvancedTab";
import { AssistantBehaviorTab } from "./components/AssistantBehaviorTab";
import { AssistantTriggerTab } from "./components/AssistantTriggerTab";
import { AssistantBasicInfoCard } from "./components/AssistantBasicInfoCard";
import type { WhatsappOption, FlowOption, PipelineOption, KnowledgeBaseOption } from "./assistantCreatorTypes";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const emptyPersonaSections = (): PersonaSectionValues => parsePersonaSections("");

const AssistantCreator: React.FC = () => {
    const navigate = useNavigate();
    const { assistantId } = useParams<{ assistantId: string }>();
    const isEditing = !!assistantId;

    const [saving, setSaving] = useState(false);
    const [activeTab, setActiveTab] = useState("comportamento");
    const [testing, setTesting] = useState(false);
    const [testDialogOpen, setTestDialogOpen] = useState(false);
    const [testMessage, setTestMessage] = useState("");
    const [testResult, setTestResult] = useState<AssistantTestResult | null>(null);

    const [whatsapps, setWhatsapps] = useState<WhatsappOption[]>([]);
    const [flows, setFlows] = useState<FlowOption[]>([]);
    const [pipelines, setPipelines] = useState<PipelineOption[]>([]);
    const [knowledgeBases, setKnowledgeBases] = useState<KnowledgeBaseOption[]>([]);
    const [aiGateways, setAiGateways] = useState<AiGateway[]>([]);

    const [name, setName] = useState("");
    const [description, setDescription] = useState("");
    const [whatsappId, setWhatsappId] = useState<string>("");
    const [allowMultipleOnConnection, setAllowMultipleOnConnection] = useState(false);
    const [mode, setMode] = useState<AssistantMode>("persona");
    const [active, setActive] = useState(true);

    const [triggerType, setTriggerType] = useState<"any" | "keyword">("any");
    const [triggerOperator, setTriggerOperator] = useState("contains");
    const [triggerValue, setTriggerValue] = useState("");

    const [sessionExpiryMinutes, setSessionExpiryMinutes] = useState("");
    const [typingDelayMs, setTypingDelayMs] = useState("");
    const [debounceSeconds, setDebounceSeconds] = useState("");
    const [endKeyword, setEndKeyword] = useState("");
    const [expiryMessage, setExpiryMessage] = useState("");
    const [closingMessage, setClosingMessage] = useState("");
    const [stopOnHumanReply, setStopOnHumanReply] = useState(true);
    const [ignoreGroups, setIgnoreGroups] = useState(true);
    const [groupsMode, setGroupsMode] = useState<"legacy" | "selective">("legacy");
    const [acceptsAudio, setAcceptsAudio] = useState(false);
    const [respondsWithAudio, setRespondsWithAudio] = useState(false);

    const [flowId, setFlowId] = useState<string>("");
    const [personaSections, setPersonaSections] = useState<PersonaSectionValues>(emptyPersonaSections());
    const [knowledgeBaseId, setKnowledgeBaseId] = useState<string>("");
    const [maxTurns, setMaxTurns] = useState("6");
    const [aiGatewayId, setAiGatewayId] = useState<string>("");
    const [ragFallbackBehavior, setRagFallbackBehavior] = useState("handoff");
    const [ragFallbackMessage, setRagFallbackMessage] = useState("");

    const [pipelineId, setPipelineId] = useState<string>("");
    const [pipelineEvents, setPipelineEvents] = useState<string[]>(["stage_changed"]);
    const [idleThresholdDays, setIdleThresholdDays] = useState("3");
    const [respondsAfterProactive, setRespondsAfterProactive] = useState(false);

    useEffect(() => {
        const loadDependencies = async () => {
            try {
                const [wa, fl, pl, kb, gw] = await Promise.all([
                    api.get("/whatsapp"),
                    api.get("/flows"),
                    api.get("/pipelines"),
                    api.get("/knowledge-bases"),
                    listAiGateways(),
                ]);
                setWhatsapps(
                    (wa.data as Array<{ id: number; name: string }>).map((w) => ({
                        id: w.id,
                        name: w.name,
                    }))
                );
                setFlows(fl.data);
                setPipelines(pl.data);
                setKnowledgeBases(kb.data);
                setAiGateways(gw);
            } catch {
                toast.error(tr("uiText.assistants.loadDeps"));
            }
        };

        const loadAssistant = async () => {
            if (!assistantId) return;
            try {
                const a: Assistant = await getAssistant(Number(assistantId));
                setName(a.name);
                setDescription(a.description ?? "");
                setWhatsappId(a.whatsappId ? String(a.whatsappId) : "");
                setAllowMultipleOnConnection(a.allowMultipleOnConnection ?? false);
                setMode(a.mode);
                setActive(a.active);

                const t = (a as unknown as { trigger?: { type?: string; operator?: string; value?: string } }).trigger ?? {};
                setTriggerType(t.type === "keyword" ? "keyword" : "any");
                setTriggerOperator(t.operator ?? "contains");
                setTriggerValue(t.value ?? "");

                setSessionExpiryMinutes(
                    a.sessionExpiryMinutes !== undefined && a.sessionExpiryMinutes !== null
                        ? String(a.sessionExpiryMinutes)
                        : ""
                );
                setTypingDelayMs(a.typingDelayMs !== undefined && a.typingDelayMs !== null ? String(a.typingDelayMs) : "");
                setDebounceSeconds(
                    a.debounceSeconds !== undefined && a.debounceSeconds !== null ? String(a.debounceSeconds) : ""
                );
                setEndKeyword(a.endKeyword ?? "");
                setExpiryMessage(a.expiryMessage ?? "");
                setClosingMessage(a.closingMessage ?? "");
                setStopOnHumanReply(a.stopOnHumanReply ?? true);
                setIgnoreGroups(a.ignoreGroups ?? true);
                setGroupsMode(a.groupsMode ?? "legacy");
                setAcceptsAudio(a.acceptsAudio ?? false);
                setRespondsWithAudio(a.respondsWithAudio ?? false);

                const cfg = (a.config ?? {}) as Record<string, unknown>;
                if (a.mode === "flow" && typeof cfg.flowId === "number") {
                    setFlowId(String(cfg.flowId));
                }
                const hasPersona = a.mode === "persona" || (a.mode === "pipeline" && cfg.respondsAfterProactive);
                if (hasPersona) {
                    if (typeof cfg.persona === "string") {
                        setPersonaSections(parsePersonaSections(cfg.persona));
                    }
                    if (typeof cfg.knowledgeBaseId === "number") {
                        setKnowledgeBaseId(String(cfg.knowledgeBaseId));
                    }
                    if (typeof cfg.maxTurns === "number") {
                        setMaxTurns(String(cfg.maxTurns));
                    }
                    if (typeof cfg.aiGatewayId === "number") {
                        setAiGatewayId(String(cfg.aiGatewayId));
                    }
                    if (typeof cfg.ragFallbackBehavior === "string") {
                        setRagFallbackBehavior(cfg.ragFallbackBehavior);
                    }
                    if (typeof cfg.ragFallbackMessage === "string") {
                        setRagFallbackMessage(cfg.ragFallbackMessage);
                    }
                }
                if (a.mode === "pipeline") {
                    if (typeof cfg.pipelineId === "number") {
                        setPipelineId(String(cfg.pipelineId));
                    }
                    if (Array.isArray(cfg.events)) {
                        setPipelineEvents(cfg.events as string[]);
                    }
                    if (typeof cfg.idleThresholdDays === "number") {
                        setIdleThresholdDays(String(cfg.idleThresholdDays));
                    }
                    setRespondsAfterProactive(Boolean(cfg.respondsAfterProactive));
                }
            } catch {
                toast.error(tr("uiText.assistants.loadOne"));
            }
        };

        loadDependencies();
        loadAssistant();
    }, [assistantId]);

    const toggleEvent = (value: string) => {
        setPipelineEvents((prev) =>
            prev.includes(value) ? prev.filter((e) => e !== value) : [...prev, value]
        );
    };

    const handlePersonaSectionChange = (key: string, value: string) => {
        setPersonaSections((prev) => ({ ...prev, [key]: value }));
    };

    const buildConfig = (): Record<string, unknown> => {
        const persona = buildPersonaFromSections(personaSections);
        if (mode === "flow") {
            return { flowId: flowId ? Number(flowId) : undefined };
        }
        if (mode === "persona") {
            return {
                persona,
                knowledgeBaseId: knowledgeBaseId ? Number(knowledgeBaseId) : undefined,
                maxTurns: maxTurns ? Number(maxTurns) : 6,
                aiGatewayId: aiGatewayId ? Number(aiGatewayId) : undefined,
                ragFallbackBehavior,
                ragFallbackMessage: ragFallbackBehavior === "apologize" ? ragFallbackMessage : undefined,
            };
        }
        if (mode === "pipeline") {
            const cfg: Record<string, unknown> = {
                pipelineId: pipelineId ? Number(pipelineId) : undefined,
                events: pipelineEvents,
                idleThresholdDays: pipelineEvents.includes("idle") && idleThresholdDays ? Number(idleThresholdDays) : undefined,
                respondsAfterProactive,
            };
            if (respondsAfterProactive) {
                cfg.persona = persona;
                cfg.knowledgeBaseId = knowledgeBaseId ? Number(knowledgeBaseId) : undefined;
                cfg.maxTurns = maxTurns ? Number(maxTurns) : 6;
                cfg.aiGatewayId = aiGatewayId ? Number(aiGatewayId) : undefined;
                cfg.ragFallbackBehavior = ragFallbackBehavior;
                cfg.ragFallbackMessage = ragFallbackBehavior === "apologize" ? ragFallbackMessage : undefined;
            }
            return cfg;
        }
        return {};
    };

    const handleSave = async () => {
        if (!name.trim()) {
            toast.warning(tr("uiText.assistants.nameRequired"));
            return;
        }

        setSaving(true);
        try {
            const payload = {
                name: name.trim(),
                description: description.trim() || undefined,
                whatsappId: whatsappId ? Number(whatsappId) : undefined,
                allowMultipleOnConnection,
                mode,
                active,
                trigger: {
                    type: triggerType,
                    operator: triggerType === "keyword" ? triggerOperator : undefined,
                    value: triggerType === "keyword" ? triggerValue.trim() : undefined,
                },
                config: buildConfig(),
                sessionExpiryMinutes: sessionExpiryMinutes ? Number(sessionExpiryMinutes) : undefined,
                typingDelayMs: typingDelayMs ? Number(typingDelayMs) : undefined,
                debounceSeconds: debounceSeconds ? Number(debounceSeconds) : undefined,
                endKeyword: endKeyword.trim() || undefined,
                expiryMessage: expiryMessage.trim() || undefined,
                closingMessage: closingMessage.trim() || undefined,
                stopOnHumanReply,
                ignoreGroups: groupsMode === "selective" ? false : ignoreGroups,
                groupsMode,
                acceptsAudio,
                respondsWithAudio,
            };

            if (isEditing) {
                await updateAssistant(Number(assistantId), payload);
                toast.success("Assistente atualizado!");
            } else {
                const created = await createAssistant(payload);
                toast.success("Assistente criado!");
                navigate(`/assistants/${created.id}`);
            }
        } catch (err) {
            const message =
                (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
                tr("uiText.assistants.saveError");
            toast.error(message);
        } finally {
            setSaving(false);
        }
    };

    const openTestDialog = () => {
        setTestResult(null);
        setTestMessage("");
        setTestDialogOpen(true);
    };

    const handleRunTest = async () => {
        if (!assistantId) return;
        setTesting(true);
        setTestResult(null);
        try {
            const result = await testAssistant(Number(assistantId), testMessage.trim() || undefined);
            setTestResult(result);
        } catch (err) {
            const message =
                (err as { response?: { data?: { error?: string } } })?.response?.data?.error ??
                tr("uiText.assistants.testError");
            setTestResult({ mode, success: false, error: message });
        } finally {
            setTesting(false);
        }
    };

    const showPersonaSections = mode === "persona" || (mode === "pipeline" && respondsAfterProactive);

    return (
        <PageContainer>
            <PageHeader
                title={isEditing ? "Editar Assistente" : tr("uiText.assistants.new")}
                description={tr("uiText.assistants.newHint")}
            >
                <div className="flex gap-2">
                    {isEditing && (
                        <Button variant="outline" onClick={openTestDialog}>
                            <PlayCircle className="mr-2 h-4 w-4" />
                            Testar Bot
                        </Button>
                    )}
                    <Button onClick={handleSave} disabled={saving}>
                        <Save className="mr-2 h-4 w-4" />
                        {saving ? "Salvando..." : tr("uiText.common.save")}
                    </Button>
                </div>
            </PageHeader>

            <PageContent>
                <div className="max-w-4xl mx-auto flex flex-col gap-4">
                    <AssistantBasicInfoCard
                        name={name}
                        onNameChange={setName}
                        whatsappId={whatsappId}
                        onWhatsappIdChange={setWhatsappId}
                        whatsapps={whatsapps}
                        description={description}
                        onDescriptionChange={setDescription}
                        active={active}
                        onActiveChange={setActive}
                        allowMultipleOnConnection={allowMultipleOnConnection}
                        onAllowMultipleOnConnectionChange={setAllowMultipleOnConnection}
                    />

                    <Card className="rounded-2xl shadow-[0px_4px_20px_rgba(0,0,0,0.06)]">
                        <CardContent className="p-5 flex flex-col gap-4">
                            <Tabs value={activeTab} onValueChange={setActiveTab}>
                                <TabsList className="grid grid-cols-4 w-full">
                                    <TabsTrigger value="comportamento">Comportamento</TabsTrigger>
                                    <TabsTrigger value="gatilho">Gatilho</TabsTrigger>
                                    <TabsTrigger value="avancado">{tr("uiText.assistants.advanced")}</TabsTrigger>
                                    <TabsTrigger value="grupos">Grupos</TabsTrigger>
                                </TabsList>

                                <TabsContent value="comportamento">
                                    <AssistantBehaviorTab
                                        mode={mode}
                                        onModeChange={setMode}
                                        flowId={flowId}
                                        onFlowIdChange={setFlowId}
                                        flows={flows}
                                        pipelineId={pipelineId}
                                        onPipelineIdChange={setPipelineId}
                                        pipelines={pipelines}
                                        pipelineEvents={pipelineEvents}
                                        onToggleEvent={toggleEvent}
                                        idleThresholdDays={idleThresholdDays}
                                        onIdleThresholdDaysChange={setIdleThresholdDays}
                                        respondsAfterProactive={respondsAfterProactive}
                                        onRespondsAfterProactiveChange={setRespondsAfterProactive}
                                        showPersonaSections={showPersonaSections}
                                        personaSections={personaSections}
                                        onPersonaSectionChange={handlePersonaSectionChange}
                                        knowledgeBaseId={knowledgeBaseId}
                                        onKnowledgeBaseIdChange={setKnowledgeBaseId}
                                        knowledgeBases={knowledgeBases}
                                        aiGatewayId={aiGatewayId}
                                        onAiGatewayIdChange={setAiGatewayId}
                                        aiGateways={aiGateways}
                                        maxTurns={maxTurns}
                                        onMaxTurnsChange={setMaxTurns}
                                        ragFallbackBehavior={ragFallbackBehavior}
                                        onRagFallbackBehaviorChange={setRagFallbackBehavior}
                                        ragFallbackMessage={ragFallbackMessage}
                                        onRagFallbackMessageChange={setRagFallbackMessage}
                                    />
                                    {mode === "router" && (
                                        <div className="pt-2">
                                            {isEditing ? (
                                                <AssistantRouterOptions assistantId={Number(assistantId)} />
                                            ) : (
                                                <p className="text-sm text-muted-foreground">
                                                    {tr("uiText.assistants.saveFirstMenu")}
                                                </p>
                                            )}
                                        </div>
                                    )}
                                </TabsContent>

                                <TabsContent value="gatilho">
                                    <AssistantTriggerTab
                                        triggerType={triggerType}
                                        onTriggerTypeChange={setTriggerType}
                                        triggerOperator={triggerOperator}
                                        onTriggerOperatorChange={setTriggerOperator}
                                        triggerValue={triggerValue}
                                        onTriggerValueChange={setTriggerValue}
                                    />
                                </TabsContent>

                                <TabsContent value="avancado">
                                    <AssistantAdvancedTab
                                        sessionExpiryMinutes={sessionExpiryMinutes}
                                        onSessionExpiryMinutesChange={setSessionExpiryMinutes}
                                        typingDelayMs={typingDelayMs}
                                        onTypingDelayMsChange={setTypingDelayMs}
                                        debounceSeconds={debounceSeconds}
                                        onDebounceSecondsChange={setDebounceSeconds}
                                        endKeyword={endKeyword}
                                        onEndKeywordChange={setEndKeyword}
                                        expiryMessage={expiryMessage}
                                        onExpiryMessageChange={setExpiryMessage}
                                        closingMessage={closingMessage}
                                        onClosingMessageChange={setClosingMessage}
                                        stopOnHumanReply={stopOnHumanReply}
                                        onStopOnHumanReplyChange={setStopOnHumanReply}
                                        groupsMode={groupsMode}
                                        ignoreGroups={ignoreGroups}
                                        onIgnoreGroupsChange={setIgnoreGroups}
                                        acceptsAudio={acceptsAudio}
                                        onAcceptsAudioChange={setAcceptsAudio}
                                        respondsWithAudio={respondsWithAudio}
                                        onRespondsWithAudioChange={setRespondsWithAudio}
                                    />
                                </TabsContent>

                                <TabsContent value="grupos" className="flex flex-col gap-4 pt-2">
                                    <div className="flex flex-col gap-1.5">
                                        <Label>{tr("uiText.assistants.groupMode")}</Label>
                                        <Select
                                            value={groupsMode}
                                            onValueChange={(v) => setGroupsMode(v as "legacy" | "selective")}
                                        >
                                            <SelectTrigger>
                                                <SelectValue />
                                            </SelectTrigger>
                                            <SelectContent>
                                                <SelectItem value="legacy">
                                                    Padrão (ignora todos ou responde todos)
                                                </SelectItem>
                                                <SelectItem value="selective">
                                                    Grupos selecionados (escolha abaixo)
                                                </SelectItem>
                                            </SelectContent>
                                        </Select>
                                    </div>

                                    {groupsMode === "selective" && !isEditing && (
                                        <p className="text-sm text-muted-foreground">
                                            {tr("uiText.assistants.saveFirstGroups")}
                                        </p>
                                    )}

                                    {groupsMode === "selective" && isEditing && (
                                        <AssistantGroupsPanel assistantId={Number(assistantId)} />
                                    )}
                                </TabsContent>
                            </Tabs>
                        </CardContent>
                    </Card>
                </div>
            </PageContent>

            <AssistantTestDialog
                open={testDialogOpen}
                onOpenChange={setTestDialogOpen}
                name={name}
                testMessage={testMessage}
                onTestMessageChange={setTestMessage}
                testing={testing}
                testResult={testResult}
                onRunTest={handleRunTest}
            />
        </PageContainer>
    );
};

export default AssistantCreator;
