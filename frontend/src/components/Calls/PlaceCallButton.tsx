import React, { useContext } from "react";
import { Phone } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { t } from "@/lib/calls/t";
import { PLACE_BLOCK_KEY, placeBlock, type PlaceInput } from "@/lib/calls/canPlace";
import { useCalls } from "@/context/Calls/CallsContext";
import { WhatsAppsContext } from "@/context/WhatsApp/WhatsAppsContext";

interface Props {
  ticket: PlaceInput & { id: number; whatsappId?: number };
}

/**
 * Botão de ligar do cabeçalho do ticket. Some em grupo/comunidade/canal e sem
 * calls:place; fica DESABILITADO, com o motivo, quando a conexão está desconectada
 * ou tem proxy, ou quando o operador já está em outra chamada.
 */
const PlaceCallButton: React.FC<Props> = ({ ticket }) => {
  const { canPlace, place, active } = useCalls();
  const { whatsApps } = useContext(WhatsAppsContext);

  if (!canPlace) return null;
  if (ticket.isGroup || ticket.isCommunity || ticket.isSubGroup) return null;

  const connection = whatsApps.find((w) => w.id === ticket.whatsappId);
  const inCall = !!active && active.phase !== "ended";
  const block = placeBlock(ticket, connection, inCall);
  const label = block ? t(PLACE_BLOCK_KEY[block]) : t("calls.place.button");

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex">
          <Button
            variant="ghost"
            size="icon"
            disabled={block !== null}
            onClick={() => void place(ticket.id)}
            aria-label={t("calls.place.button")}
            data-testid="place-call"
            data-block={block ?? ""}
          >
            <Phone />
          </Button>
        </span>
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
};

export default PlaceCallButton;
