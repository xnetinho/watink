export interface WhatsappOption {
  id: number;
  name: string;
}

export interface FlowOption {
  id: number;
  name: string;
}

export interface PipelineOption {
  id: number;
  name: string;
}

export interface KnowledgeBaseOption {
  id: number;
  name: string;
}

export interface AiGateway {
  id: number;
  name: string;
  provider: string;
  model: string;
}

export type AssistantMode = "pipeline" | "flow" | "persona" | "router";
