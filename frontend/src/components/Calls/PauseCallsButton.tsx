import React from "react";
import { PhoneOff, Phone } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { t } from "@/lib/calls/t";
import { useCalls } from "@/context/Calls/CallsContext";

/**
 * Pausar/retomar o recebimento de chamadas neste navegador. A preferência é local
 * (localStorage) e avisada ao servidor, que então não conta este operador como
 * elegível. Só aparece para quem pode receber chamadas.
 */
const PauseCallsButton: React.FC = () => {
  const { canReceive, paused, setPaused } = useCalls();
  if (!canReceive) return null;
  const label = paused ? t("calls.pause.resume") : t("calls.pause.pause");
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant={paused ? "secondary" : "ghost"}
          size="icon"
          onClick={() => setPaused(!paused)}
          aria-pressed={paused}
          aria-label={label}
          data-testid="pause-calls"
        >
          {paused ? <PhoneOff /> : <Phone />}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{paused ? t("calls.pause.paused") : label}</TooltipContent>
    </Tooltip>
  );
};

export default PauseCallsButton;
