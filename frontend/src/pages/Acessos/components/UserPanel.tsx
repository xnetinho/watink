import React, { useState, useEffect } from "react";
import { Formik, Form } from "formik";
import * as Yup from "yup";
import { Loader2 } from "lucide-react";

import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetFooter,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";

import UserDataSection from "./UserDataSection";
import UserCargoSection from "./UserCargoSection";
import UserSetoresSection from "./UserSetoresSection";
import UserAlcanceSection from "./UserAlcanceSection";
import type {
  AcessosUserDetail,
  CargoListItem,
  SetorListItem,
  UserSavePayload,
  UserSetorVinculo,
} from "../acessosTypes";

import { i18n } from "../../../translate/i18n";

const tr = (key: string, opts: Record<string, unknown> = {}): string => String(i18n.t(key, opts) ?? "");
const UserSchema = Yup.object().shape({
  name: Yup.string().min(2, () => tr("uiText.validation.tooShort")).max(255, () => tr("uiText.validation.tooLong")).required(() => tr("uiText.acessos.nameRequired")),
  email: Yup.string().email(() => tr("uiText.utils.invalidEmail")).required(() => tr("uiText.acessos.emailRequired")),
  password: Yup.string().min(5, () => tr("uiText.validation.tooShort")).max(128, () => tr("uiText.validation.tooLong")),
});

interface UserPanelProps {
  open: boolean;
  loading: boolean;
  editingUser: AcessosUserDetail | null;
  editingUserSetores: UserSetorVinculo[];
  cargos: CargoListItem[];
  setores: SetorListItem[];
  onClose: () => void;
  onSave: (id: number | null, payload: UserSavePayload) => Promise<boolean>;
}

interface FormValues {
  name: string;
  email: string;
  password: string;
}

const UserPanel: React.FC<UserPanelProps> = ({
  open,
  loading,
  editingUser,
  editingUserSetores,
  cargos,
  setores,
  onClose,
  onSave,
}) => {
  const [showPassword, setShowPassword] = useState(false);
  const [cargoId, setCargoId] = useState<number | null>(null);
  const [alcance, setAlcance] = useState<string>("proprio");
  const [vinculos, setVinculos] = useState<UserSetorVinculo[]>([]);

  useEffect(() => {
    setCargoId(editingUser?.cargoId ?? null);
    setAlcance(editingUser?.alcance ?? "proprio");
  }, [editingUser]);

  useEffect(() => {
    setVinculos(editingUserSetores);
  }, [editingUserSetores]);

  const isNew = !editingUser;

  const initialValues: FormValues = {
    name: editingUser?.name ?? "",
    email: editingUser?.email ?? "",
    password: "",
  };

  return (
    <Sheet open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
      <SheetContent side="right" className="w-full sm:max-w-xl overflow-y-auto p-6">
        <SheetHeader>
          <SheetTitle>{isNew ? tr("uiText.acessos.newUser") : tr("uiText.acessos.editUser")}</SheetTitle>
        </SheetHeader>

        {loading ? (
          <div className="flex min-h-[300px] items-center justify-center">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        ) : (
          <Formik
            initialValues={initialValues}
            enableReinitialize
            validationSchema={UserSchema}
            onSubmit={async (values, actions) => {
              if (isNew && !values.password) {
                actions.setFieldError("password", tr("uiText.acessos.passwordRequired"));
                actions.setSubmitting(false);
                return;
              }
              const payload: UserSavePayload = {
                name: values.name,
                email: values.email,
                alcance,
                cargoId,
                setores: vinculos,
              };
              if (values.password) payload.password = values.password;
              await onSave(editingUser?.id ?? null, payload);
              actions.setSubmitting(false);
            }}
          >
            {({ touched, errors, isSubmitting }) => (
              <Form className="mt-4 flex flex-col gap-5">
                <UserDataSection
                  isNew={isNew}
                  touched={touched}
                  errors={errors}
                  showPassword={showPassword}
                  onTogglePassword={() => setShowPassword((p) => !p)}
                />

                <Separator />

                <UserCargoSection cargos={cargos} cargoId={cargoId} onChange={setCargoId} />

                <Separator />

                <UserSetoresSection setores={setores} vinculos={vinculos} onChange={setVinculos} />

                <Separator />

                <UserAlcanceSection alcance={alcance} onChange={setAlcance} />

                <SheetFooter className="mt-2 gap-2">
                  <Button type="button" variant="outline" onClick={onClose} disabled={isSubmitting}>
                    {tr("uiText.common.cancel")}
                  </Button>
                  <Button type="submit" disabled={isSubmitting}>
                    {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                    {tr("uiText.common.save")}
                  </Button>
                </SheetFooter>
              </Form>
            )}
          </Formik>
        )}
      </SheetContent>
    </Sheet>
  );
};

export default UserPanel;
