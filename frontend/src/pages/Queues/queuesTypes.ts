export interface Queue {
  id: number;
  name: string;
  color: string | null;
  greetingMessage: string | null;
  /** Derivado de Tenants.defaultQueueId no GET /queue. */
  isDefault?: boolean;
}

export type QueuesAction =
  | { type: "LOAD_QUEUES"; payload: Queue[] }
  | { type: "UPDATE_QUEUES"; payload: Queue }
  | { type: "DELETE_QUEUE"; payload: number };
