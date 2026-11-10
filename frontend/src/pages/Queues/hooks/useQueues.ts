import { useEffect, useReducer, useState } from "react";
import { toast } from "react-toastify";

import api from "../../../services/api";
import { subscribeToSocket } from "../../../services/sse-client";
import { i18n } from "../../../translate/i18n";
import toastError from "../../../errors/toastError";
import { Queue, QueuesAction } from "../queuesTypes";

function reducer(state: Queue[], action: QueuesAction): Queue[] {
  switch (action.type) {
    case "LOAD_QUEUES":
      return [...action.payload];
    case "UPDATE_QUEUES": {
      const queue = action.payload;
      const idx = state.findIndex((q) => q.id === queue.id);
      if (idx !== -1) {
        const next = [...state];
        next[idx] = queue;
        return next;
      }
      return [queue, ...state];
    }
    case "DELETE_QUEUE":
      return state.filter((q) => q.id !== action.payload);
    default:
      return state;
  }
}

export interface QueueDependents {
  Tickets: number;
  Users: number;
  Connections: number;
  Setores: number;
  Children: number;
}

export interface UseQueuesReturn {
  moveRequest: { queue: Queue; dependents: QueueDependents } | null;
  closeMoveRequest: () => void;
  handleMoveAndDelete: (queueId: number, moveTo: number) => Promise<void>;
  queues: Queue[];
  loading: boolean;
  queueModalOpen: boolean;
  selectedQueue: Queue | null;
  confirmModalOpen: boolean;
  handleOpenQueueModal: () => void;
  handleCloseQueueModal: () => void;
  handleEditQueue: (queue: Queue) => void;
  handleDeleteQueue: (queueId: number) => Promise<void>;
  setSelectedQueue: (queue: Queue | null) => void;
  setConfirmModalOpen: (open: boolean) => void;
}

export function useQueues(): UseQueuesReturn {
  const [loading, setLoading] = useState(false);
  const [queues, dispatch] = useReducer(reducer, []);
  const [queueModalOpen, setQueueModalOpen] = useState(false);
  const [selectedQueue, setSelectedQueue] = useState<Queue | null>(null);
  const [confirmModalOpen, setConfirmModalOpen] = useState(false);
  const [moveRequest, setMoveRequest] = useState<{ queue: Queue; dependents: QueueDependents } | null>(null);

  useEffect(() => {
    (async () => {
      setLoading(true);
      try {
        const { data } = await api.get<Queue[]>("/queue");
        dispatch({ type: "LOAD_QUEUES", payload: data });
      } catch (err) {
        toastError(err);
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  useEffect(() => {
    const handleQueue = (data: { action: string; queue?: Queue; queueId?: number }) => {
      if (data.action === "update" || data.action === "create") {
        dispatch({ type: "UPDATE_QUEUES", payload: data.queue! });
      }
      if (data.action === "delete") {
        dispatch({ type: "DELETE_QUEUE", payload: data.queueId! });
      }
    };

    return subscribeToSocket({ queue: handleQueue });
  }, []);

  const handleOpenQueueModal = () => {
    setSelectedQueue(null);
    setQueueModalOpen(true);
  };

  const handleCloseQueueModal = () => {
    setSelectedQueue(null);
    setQueueModalOpen(false);
  };

  const handleEditQueue = (queue: Queue) => {
    setSelectedQueue(queue);
    setQueueModalOpen(true);
  };

  const handleDeleteQueue = async (queueId: number) => {
    try {
      await api.delete(`/queue/${queueId}`);
      toast.success(i18n.t("queues.toasts.deleted"));
    } catch (err) {
      // 409: a fila ainda tem ticket, usuário, conexão ou setor. O servidor devolve os números e a tela pergunta para qual fila mover.
      const e = err as { response?: { status?: number; data?: { dependents?: QueueDependents } } };
      const queue = queues.find((q) => q.id === queueId);
      if (e.response?.status === 409 && e.response.data?.dependents && queue) {
        setMoveRequest({ queue, dependents: e.response.data.dependents });
      } else {
        toastError(err);
      }
    }
    setSelectedQueue(null);
  };

  const handleMoveAndDelete = async (queueId: number, moveTo: number) => {
    try {
      await api.delete(`/queue/${queueId}`, { params: { moveTo } });
      toast.success(i18n.t("queues.toasts.deleted"));
      setMoveRequest(null);
    } catch (err) {
      toastError(err);
    }
  };

  return {
    moveRequest,
    closeMoveRequest: () => setMoveRequest(null),
    handleMoveAndDelete,
    queues,
    loading,
    queueModalOpen,
    selectedQueue,
    confirmModalOpen,
    handleOpenQueueModal,
    handleCloseQueueModal,
    handleEditQueue,
    handleDeleteQueue,
    setSelectedQueue,
    setConfirmModalOpen,
  };
}
