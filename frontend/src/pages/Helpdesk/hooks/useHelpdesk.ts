import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router";
import { toast } from "react-toastify";

import api from "../../../services/api";
import { ProtocolListItem, HelpdeskFiltersState } from "../helpdeskTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface UseHelpdeskReturn {
  protocols: ProtocolListItem[];
  loading: boolean;
  filters: HelpdeskFiltersState;
  modalOpen: boolean;
  setSearchParam: (value: string) => void;
  setStatusFilter: (value: string) => void;
  setPriorityFilter: (value: string) => void;
  handleOpenModal: () => void;
  handleCloseModal: () => void;
  handleViewProtocol: (id: number) => void;
}

export function useHelpdesk(): UseHelpdeskReturn {
  const navigate = useNavigate();
  const [protocols, setProtocols] = useState<ProtocolListItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchParam, setSearchParam] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [priorityFilter, setPriorityFilter] = useState("all");
  const [modalOpen, setModalOpen] = useState(false);

  const loadProtocols = useCallback(async () => {
    try {
      setLoading(true);
      const { data } = await api.get("/protocols", {
        params: {
          searchParam,
          status: statusFilter === "all" ? undefined : statusFilter,
          priority: priorityFilter === "all" ? undefined : priorityFilter,
        },
      });
      setProtocols(data.protocols);
    } catch {
      toast.error(tr("uiText.helpdesk.protocolsError"));
    } finally {
      setLoading(false);
    }
  }, [searchParam, statusFilter, priorityFilter]);

  useEffect(() => {
    loadProtocols();
  }, [loadProtocols]);

  const handleOpenModal = () => setModalOpen(true);

  const handleCloseModal = () => {
    setModalOpen(false);
    loadProtocols();
  };

  const handleViewProtocol = (id: number) => {
    navigate(`/helpdesk/${id}`);
  };

  return {
    protocols,
    loading,
    filters: { searchParam, statusFilter, priorityFilter },
    modalOpen,
    setSearchParam,
    setStatusFilter,
    setPriorityFilter,
    handleOpenModal,
    handleCloseModal,
    handleViewProtocol,
  };
}
