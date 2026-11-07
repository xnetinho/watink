import { useEffect } from "react";
import { subscribeToSocket } from "../../../services/sse-client";
import type { WhatsApp } from "../connectionConfigTypes";

interface UseWhatsAppSocketOptions {
  connectionId: string | undefined;
  setWhatsapp: React.Dispatch<React.SetStateAction<WhatsApp | null>>;
  setShowQrCode: (v: boolean) => void;
  setShowPairingInput: (v: boolean) => void;
  setConnecting: (v: boolean) => void;
  setRestarting: (v: boolean) => void;
  setPairingCode: (v: string) => void;
  setPairingLoading: (v: boolean) => void;
  setPhoneNumber: (v: string) => void;
  fetchWhatsapp: () => Promise<void>;
}

export const useWhatsAppSocket = ({
  connectionId,
  setWhatsapp,
  setShowQrCode,
  setShowPairingInput,
  setConnecting,
  setRestarting,
  setPairingCode,
  setPairingLoading,
  setPhoneNumber,
  fetchWhatsapp,
}: UseWhatsAppSocketOptions): void => {
  useEffect(() => {
    const handleConnection = (data: { action: string; connection: WhatsApp & { id: number } }) => {
      if (data.action === "update" && data.connection.id === parseInt(connectionId ?? "0")) {
        setWhatsapp((prev) => prev ? { ...prev, ...data.connection } : (data.connection as WhatsApp));

        if (data.connection.status === "QRCODE") {
          setShowQrCode(true);
          setShowPairingInput(false);
          setConnecting(false);
          if (!data.connection.qrcode) void fetchWhatsapp();
        }

        if (data.connection.pairingCode) {
          setPairingCode(data.connection.pairingCode);
          setPairingLoading(false);
        }

        if (["CONNECTED", "QRCODE", "PAIRING", "DISCONNECTED", "TIMEOUT"].includes(data.connection.status)) {
          setConnecting(false);
          setRestarting(false);
        }

        if (data.connection.status === "CONNECTED") {
          setShowPairingInput(false);
          setShowQrCode(false);
          setPairingCode("");
          setPhoneNumber("");
          setPairingLoading(false);
          void fetchWhatsapp();
        }

        if (data.connection.status === "DISCONNECTED" || data.connection.status === "TIMEOUT") {
          setShowQrCode(false);
          setShowPairingInput(false);
          setPairingLoading(false);
        }
      }
    };

    // Sinal ao vivo de risco de ban/throttle (whatsmeow IQ 401/403/429/463).
    // O valor persistido (lastRiskCode/lastRiskAt) só chega no fetch seguinte,
    // então refazemos o fetch para o banner reagir sem esperar outro evento.
    const handleSessionRisk = (data: { sessionId: number }) => {
      if (data.sessionId === parseInt(connectionId ?? "0")) {
        void fetchWhatsapp();
      }
    };

    return subscribeToSocket({
      channelConnection: handleConnection,
      whatsappSessionRisk: handleSessionRisk,
    });
  }, [
    connectionId,
    fetchWhatsapp,
    setWhatsapp,
    setShowQrCode,
    setShowPairingInput,
    setConnecting,
    setRestarting,
    setPairingCode,
    setPairingLoading,
    setPhoneNumber,
  ]);
};
