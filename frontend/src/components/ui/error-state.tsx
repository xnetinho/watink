import * as React from "react";
import { AlertTriangle } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
export interface ErrorStateProps extends React.HTMLAttributes<HTMLDivElement> {
  /** Mensagem principal, ex: "Não foi possível carregar os dados" */
  title?: string;
  /** Texto de apoio opcional com detalhe do erro */
  description?: string;
  /** Chamado ao clicar em "Tentar novamente" — se omitido, o botão não é exibido */
  onRetry?: () => void;
}

const ErrorState = React.forwardRef<HTMLDivElement, ErrorStateProps>(
  ({ className, title = String(i18n.t("uiText.ui.loadError")), description, onRetry, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "flex flex-col items-center justify-center gap-2 py-12 text-center",
        className
      )}
      {...props}
    >
      <div className="mb-1 flex h-12 w-12 items-center justify-center rounded-full bg-status-error-bg text-status-error-text [&>svg]:h-6 [&>svg]:w-6">
        <AlertTriangle aria-hidden="true" />
      </div>
      <p className="text-sm font-medium text-foreground">{title}</p>
      {description && (
        <p className="max-w-sm text-sm text-muted-foreground">{description}</p>
      )}
      {onRetry && (
        <Button variant="outline" size="sm" className="mt-2" onClick={onRetry}>
          {tr("uiText.ui.retry")}
        </Button>
      )}
    </div>
  )
);
ErrorState.displayName = "ErrorState";

export { ErrorState };
export default ErrorState;
