/* @jsxImportSource react */
import React, { useEffect, useState } from "react";
import { useNavigate, useSearchParams, Link as RouterLink } from "react-router";
import { Loader2, ArrowLeft, Eye, EyeOff, PartyPopper } from "lucide-react";

import api from "../../services/api";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../../components/ui/card";
import { usePublicPlans, formatBRL } from "../Plans/planCatalog";

import { i18n } from "../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
interface FormState {
  companyName: string;
  document: string;
  firstName: string;
  lastName: string;
  email: string;
  password: string;
}

const emptyForm: FormState = {
  companyName: "",
  document: "",
  firstName: "",
  lastName: "",
  email: "",
  password: "",
};

// Mensagens amigáveis para os códigos de erro do proxy público de registro
// (business/internal/controllers/register.go) — nunca mostrar o código cru.
const ERROR_MESSAGES: Record<string, string> = {
  weak_password: "A senha precisa ter pelo menos 8 caracteres.",
  captcha_required: "uiText.register.err.captchaRequired",
  captcha_failed: "uiText.register.err.captchaFailed",
  plan_not_found: "uiText.register.err.planNotFound",
  plan_not_eligible: "uiText.register.err.planNotEligible",
  email_already_registered: "uiText.register.err.emailTaken",
  registration_closed: "uiText.register.err.closed",
  too_many_attempts: "uiText.register.err.tooMany",
  invalid_request: "uiText.register.err.invalid",
};

function friendlyError(code: string | undefined): string {
  if (!code) return tr("uiText.register.err.generic");
  return ERROR_MESSAGES[code] ? tr(ERROR_MESSAGES[code]) : tr("uiText.register.err.generic");
}

type Stage = "provisioning" | "ready" | "failed";

const RegisterPage: React.FC = () => {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const plansState = usePublicPlans();

  const [selectedPlanId, setSelectedPlanId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);
  const [showPassword, setShowPassword] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [registrationId, setRegistrationId] = useState<string | null>(null);
  const [stage, setStage] = useState<Stage | null>(null);

  // Plano pré-selecionado via ?planId= (vindo do botão "Contratar" em
  // /planos) — cai no primeiro plano elegível se o parâmetro faltar ou não
  // corresponder a nenhum plano da lista (ex.: link antigo, plano desativado).
  useEffect(() => {
    if (plansState.status !== "ready" || selectedPlanId) return;
    const fromQuery = searchParams.get("planId");
    const match = fromQuery && plansState.plans.some((p) => p.id === fromQuery);
    setSelectedPlanId(match ? fromQuery : plansState.plans[0].id);
  }, [plansState, searchParams, selectedPlanId]);

  // Poll do status até o tenant ficar utilizável (ou falhar).
  useEffect(() => {
    if (stage !== "provisioning" || !registrationId) return;
    let cancelled = false;
    let attempts = 0;
    const poll = async () => {
      if (cancelled) return;
      attempts += 1;
      try {
        const { data } = await api.get(`/register/${registrationId}/status`);
        if (cancelled) return;
        if (data?.status === "active") {
          setStage("ready");
          return;
        }
        if (attempts >= 40) {
          setStage("failed");
          return;
        }
        setTimeout(poll, 3000);
      } catch {
        if (!cancelled) setStage("failed");
      }
    };
    poll();
    return () => {
      cancelled = true;
    };
  }, [stage, registrationId]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedPlanId) return;
    setErrorMessage(null);
    setSubmitting(true);
    try {
      const { data } = await api.post("/register", {
        planId: selectedPlanId,
        companyName: form.companyName,
        document: form.document || undefined,
        firstName: form.firstName,
        lastName: form.lastName,
        email: form.email,
        password: form.password,
      });
      setRegistrationId(data.registrationId);
      setStage("provisioning");
    } catch (err: any) {
      setErrorMessage(friendlyError(err?.response?.data?.error));
    } finally {
      setSubmitting(false);
    }
  };

  const goToLogin = () => {
    navigate(`/login?email=${encodeURIComponent(form.email)}`);
  };

  if (plansState.status === "loading") {
    return (
      <div className="min-h-screen flex items-center justify-center bg-slate-50 dark:bg-slate-950">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (plansState.status === "unavailable") {
    return (
      <div className="min-h-screen flex items-center justify-center p-4 bg-slate-50 dark:bg-slate-950">
        <Card className="w-full max-w-md">
          <CardContent className="pt-8 pb-8 px-8 text-center space-y-4">
            <p className="text-muted-foreground">
              {tr("uiText.plans.selfServiceOff")}
            </p>
            <RouterLink to="/login" className="inline-flex items-center gap-2 text-primary font-medium hover:underline">
              <ArrowLeft className="h-4 w-4" /> Voltar para o login
            </RouterLink>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (stage === "provisioning" || stage === "ready" || stage === "failed") {
    return (
      <div className="min-h-screen flex items-center justify-center p-4 bg-slate-50 dark:bg-slate-950">
        <Card className="w-full max-w-md">
          <CardContent className="pt-10 pb-10 px-8 text-center space-y-5">
            {stage === "provisioning" && (
              <>
                <Loader2 className="h-10 w-10 animate-spin text-primary mx-auto" />
                <div>
                  <p className="font-medium">{tr("uiText.register.preparing")}</p>
                  <p className="text-sm text-muted-foreground mt-1">{tr("uiText.register.fewSeconds")}</p>
                </div>
              </>
            )}
            {stage === "ready" && (
              <>
                <PartyPopper className="h-10 w-10 text-primary mx-auto" />
                <div>
                  <p className="font-medium">Tudo pronto!</p>
                  <p className="text-sm text-muted-foreground mt-1">{tr("uiText.register.created")}</p>
                </div>
                <Button className="w-full h-11" onClick={goToLogin}>
                  Entrar agora
                </Button>
              </>
            )}
            {stage === "failed" && (
              <>
                <p className="font-medium">{tr("uiText.register.slow")}</p>
                <p className="text-sm text-muted-foreground">
                  {tr("uiText.register.tryLater")}
                </p>
                <Button variant="outline" className="w-full h-11" onClick={goToLogin}>
                  {tr("uiText.register.goLogin")}
                </Button>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    );
  }

  const { plans } = plansState;

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-slate-50 dark:bg-slate-950">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle>{tr("uiText.register.title")}</CardTitle>
          <CardDescription>{tr("uiText.register.subtitle")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="planId">Plano</Label>
              <select
                id="planId"
                value={selectedPlanId ?? ""}
                onChange={(e) => setSelectedPlanId(e.target.value)}
                className="flex h-11 w-full rounded-lg border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              >
                {plans.map((plan) => (
                  <option key={plan.id} value={plan.id}>
                    {plan.name} — {plan.priceCents === 0 ? tr("uiText.plans.free") : formatBRL(plan.priceCents)}
                    {plan.priceCents > 0 ? (plan.billingCycle === "yearly" ? "/ano" : "/mês") : ""}
                    {plan.trialDays > 0 ? tr("uiText.register.trial", { days: plan.trialDays }) : ""}
                  </option>
                ))}
              </select>
              <RouterLink to="/planos" className="text-xs text-primary hover:underline w-fit">
                Ver detalhes e comparar planos
              </RouterLink>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="companyName">{tr("uiText.register.company")}</Label>
              <Input id="companyName" name="companyName" required value={form.companyName} onChange={handleChange} className="h-11" />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="document">CNPJ/CPF (opcional)</Label>
              <Input id="document" name="document" value={form.document} onChange={handleChange} className="h-11" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="firstName">{tr("uiText.common.name")}</Label>
                <Input id="firstName" name="firstName" required value={form.firstName} onChange={handleChange} className="h-11" />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="lastName">Sobrenome</Label>
                <Input id="lastName" name="lastName" required value={form.lastName} onChange={handleChange} className="h-11" />
              </div>
            </div>
            <div className="grid gap-2">
              <Label htmlFor="email">E-mail</Label>
              <Input id="email" name="email" type="email" required value={form.email} onChange={handleChange} className="h-11" />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="password">Senha</Label>
              <div className="relative">
                <Input
                  id="password"
                  name="password"
                  type={showPassword ? "text" : "password"}
                  required
                  value={form.password}
                  onChange={handleChange}
                  className="h-11 pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
              <p className="text-xs text-muted-foreground">{tr("uiText.register.passwordHint")}</p>
            </div>

            {errorMessage && <p className="text-sm text-destructive">{errorMessage}</p>}

            <Button type="submit" className="h-11 w-full text-base font-bold" disabled={submitting || !selectedPlanId}>
              {submitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" /> Criando conta...
                </>
              ) : (
                "Criar minha conta"
              )}
            </Button>
          </form>

          <div className="text-center text-sm text-muted-foreground mt-6">
            Já tem uma conta?{" "}
            <RouterLink to="/login" className="font-medium text-primary hover:underline">
              Entrar
            </RouterLink>
          </div>
        </CardContent>
      </Card>
    </div>
  );
};

export default RegisterPage;
