import { useEffect, useState } from "react";
import api from "../services/api";
import toastError from "../errors/toastError";
import { ConnectionOption, toConnectionOptions } from "../lib/connections/connectionRef";

/** Todas as conexões da empresa, de qualquer canal. */
export const useConnectionOptions = (enabled = true) => {
  const [options, setOptions] = useState<ConnectionOption[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!enabled) return;
    let alive = true;
    setLoading(true);
    api
      .get("/channels/connections")
      .then(({ data }) => {
        if (alive) setOptions(Array.isArray(data) ? toConnectionOptions(data) : []);
      })
      .catch(toastError)
      .finally(() => alive && setLoading(false));
    return () => {
      alive = false;
    };
  }, [enabled]);

  return { options, loading };
};
