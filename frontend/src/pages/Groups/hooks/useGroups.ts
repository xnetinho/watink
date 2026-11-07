import { useEffect, useState } from "react";
import api from "../../../services/api";
import { useLocalStorage } from "../../../hooks/useLocalStorage";
import { ConnectionOption } from "../groupTypes";

/**
 * Shared connection-selection state for Groups/Communities pages (plan §6.3
 * — a WhatsApp group cannot exist without a connection, so every screen is
 * scoped by connectionId, persisted across visits).
 */
export function useGroupsConnection() {
    const [connections, setConnections] = useState<ConnectionOption[]>([]);
    const [loadingConnections, setLoadingConnections] = useState(true);
    const [connectionId, setConnectionId] = useLocalStorage<number | null>("groupsConnectionId", null);

    useEffect(() => {
        let active = true;
        (async () => {
            try {
                const { data } = await api.get("/channels/connections");
                const list: ConnectionOption[] = Array.isArray(data)
                    ? data.filter((c: ConnectionOption) => c.kind === "whatsapp")
                    : [];
                if (!active) return;
                setConnections(list);
                if (connectionId === null && list.length > 0) {
                    setConnectionId(list[0].id);
                } else if (connectionId !== null && !list.some((w) => w.id === connectionId)) {
                    setConnectionId(list.length > 0 ? list[0].id : null);
                }
            } finally {
                if (active) setLoadingConnections(false);
            }
        })();
        return () => {
            active = false;
        };
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    return { connections, connectionId, setConnectionId, loadingConnections };
}
