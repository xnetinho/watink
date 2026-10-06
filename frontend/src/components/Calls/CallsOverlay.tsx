import React, { useContext, useEffect, useState } from "react";
import api from "@/services/api";
import { WhatsAppsContext } from "@/context/WhatsApp/WhatsAppsContext";
import { check } from "@/components/Can";
import { AuthContext } from "@/context/Auth/AuthContext";
import { useCalls } from "@/context/Calls/CallsContext";
import IncomingCallModal from "./IncomingCallModal";
import ActiveCallPanel from "./ActiveCallPanel";

interface RecordingConfig {
  mode: string;
  available: boolean;
}

/**
 * Camada global das chamadas: toque e tela da chamada em curso, visíveis em
 * qualquer página. O modo de gravação só é consultado por quem pode ligar a
 * gravação por chamada (precisa ser o operador da chamada, não o gestor), via a
 * mesma rota de configuração quando o usuário tem calls:manage; os demais não
 * veem o botão (a gravação "opcional" é decidida pelo servidor, que rejeita).
 */
const CallsOverlay: React.FC = () => {
  const { canReceive, canPlace, active } = useCalls();
  const { user } = useContext(AuthContext);
  const { whatsApps } = useContext(WhatsAppsContext);
  const [rec, setRec] = useState<RecordingConfig>({ mode: "off", available: false });

  const canManage = check(user, "calls:manage");
  const inCall = !!active;
  useEffect(() => {
    if (!canManage || !inCall) return;
    api
      .get<RecordingConfig>("/calls/recording-config")
      .then(({ data }) => setRec({ mode: data.mode, available: data.available }))
      .catch(() => undefined);
  }, [canManage, inCall]);

  if (!canReceive && !canPlace) return null;

  return (
    <>
      {canReceive && <IncomingCallModal connectionName={(id) => whatsApps.find((w) => w.id === id)?.name} />}
      <ActiveCallPanel recordingAvailable={rec.available} recordingMode={rec.mode} />
    </>
  );
};

export default CallsOverlay;
