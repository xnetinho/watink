import { useEffect, useState } from "react";
import api from "../services/api";
import toastError from "../errors/toastError";
import { ConnectionOption, toConnectionOptions } from "../lib/connections/connectionRef";

/** Todas as conexões da empresa (WhatsApp e demais canais) com a origem explícita. */
export const useConnectionOptions = (enabled = true) => {
  const [options, setOptions] = useState<ConnectionOption[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!enabled) return;
    let alive = true;
    setLoading(true);
    Promise.allSettled([api.get("/whatsapp"), api.get("/channels/connections")])
      .then(([wa, ch]) => {
        if (!alive) return;
        const list = (r: PromiseSettledResult<{ data: unknown }>) =>
          r.status === "fulfilled" && Array.isArray(r.value.data) ? (r.value.data as never[]) : [];
        setOptions(toConnectionOptions(list(wa), list(ch)));
      })
      .catch(toastError)
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [enabled]);

  return { options, loading };
};
