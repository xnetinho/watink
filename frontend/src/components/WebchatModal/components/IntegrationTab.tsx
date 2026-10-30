import React from "react";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface IntegrationTabProps {
  whatsAppId?: number;
}

const IntegrationTab: React.FC<IntegrationTabProps> = ({ whatsAppId }) => {
  if (!whatsAppId) {
    return <p className="text-sm text-muted-foreground mt-4">{tr("uiText.webchatModal.saveFirst")}</p>;
  }

  const backendUrl =
    (import.meta.env.VITE_BACKEND_URL as string | undefined) ||
    window.location.origin.replace("app", "api").replace("3000", "8082");

  const script = `<script>
  window.watinkWebchatConfig = {
    url: "${backendUrl}",
    whatsappId: "${whatsAppId}"
  };
  (function(d, s, id) {
    var js, fjs = d.getElementsByTagName(s)[0];
    if (d.getElementById(id)) return;
    js = d.createElement(s); js.id = id;
    js.src = "${backendUrl}/public/webchat";
    fjs.parentNode.insertBefore(js, fjs);
  }(document, 'script', 'watink-webchat-sdk'));
</script>`;

  return (
    <div className="space-y-2 mt-4">
      <Label>{tr("uiText.webchatModal.embedCode")}</Label>
      <Textarea
        readOnly
        value={script}
        rows={10}
        className="font-mono text-xs bg-muted"
      />
    </div>
  );
};

export default IntegrationTab;
