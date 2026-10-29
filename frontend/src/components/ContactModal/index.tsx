import React, { useState, useEffect } from "react";
import * as Yup from "yup";
import { Formik, Form, Field, FieldArray } from "formik";
import { toast } from "react-toastify";
import { Loader2, Plus, Trash2 } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import { i18n } from "../../translate/i18n";
import api from "../../services/api";
import toastError from "../../errors/toastError";
import { useConnectionOptions } from "../../hooks/useConnectionOptions";
import { channelDefinition } from "../../config/channels";
import { connectionKey, parseConnectionKey } from "../../lib/connections/connectionRef";
import { TicketConnectionBadge } from "../TicketConnectionBadge";

const ContactSchema = Yup.object().shape({
  name: Yup.string().min(2, "Too Short!").max(50, "Too Long!").required("Required"),
  number: Yup.string().min(8, "Too Short!").max(50, "Too Long!"),
  connection: Yup.string().required("Required"),
  email: Yup.string().email("Invalid email"),
});

interface ExtraInfo {
  name: string;
  value: string;
}

interface ContactValues {
  /** Conexão de origem ("whatsapp:1", "channel:2"): o contato pertence a ela e é por ela que recebe e envia. */
  connection: string;
  name: string;
  number: string;
  email: string;
  extraInfo: ExtraInfo[];
}

interface ContactModalProps {
  open: boolean;
  onClose: () => void;
  contactId?: number | string;
  initialValues?: Partial<ContactValues>;
  onSave?: (contact: ContactValues) => void;
}

const ContactModal = ({ open, onClose, contactId, initialValues, onSave }: ContactModalProps) => {
  const [contact, setContact] = useState<ContactValues>({ connection: "", name: "", number: "", email: "", extraInfo: [] });
  const { options: allConnections } = useConnectionOptions(open);
  // Criação: só conexões de canais onde a empresa pode cadastrar contato (o bot do Telegram só responde a quem escreveu
  // antes, então não há o que cadastrar). Edição: apenas a conexão do próprio contato, que não muda.
  const connections = allConnections.filter((c) =>
    contactId ? connectionKey(c) === contact.connection : channelDefinition(c.kind).canCreateContacts,
  );

  useEffect(() => {
    if (initialValues) {
      setContact((prev) => ({ ...prev, ...initialValues }));
    }
    if (!contactId) return;
    api.get(`/contacts/${contactId}`)
      .then(({ data }) => setContact({ ...data, connection: connectionKey({ source: data.connectionSource, id: data.connectionId }) }))
      .catch(toastError);
  }, [contactId, open, initialValues]);

  const handleClose = () => {
    onClose();
    setContact({ connection: "", name: "", number: "", email: "", extraInfo: [] });
  };

  const handleSaveContact = async (values: ContactValues) => {
    try {
      if (contactId) {
        // A conexão do contato não muda depois de criado.
        const { connection: _connection, ...rest } = values;
        await api.put(`/contacts/${contactId}`, rest);
      } else {
        const { connection, ...rest } = values;
        const ref = parseConnectionKey(connection);
        const { data } = await api.post("/contacts", { ...rest, connectionSource: ref?.source, connectionId: ref?.id });
        if (onSave) onSave(data);
      }
      toast.success(i18n.t("contactModal.success"));
      handleClose();
    } catch (err) {
      toastError(err);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) handleClose(); }}>
      <DialogContent className="max-w-md max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {contactId ? i18n.t("contactModal.title.edit") : i18n.t("contactModal.title.add")}
          </DialogTitle>
        </DialogHeader>

        <Formik
          initialValues={contact}
          enableReinitialize
          validationSchema={ContactSchema}
          onSubmit={(values, actions) => {
            handleSaveContact(values);
            actions.setSubmitting(false);
          }}
        >
          {({ values, errors, touched, isSubmitting, setFieldValue }) => (
            <Form className="space-y-4">
              <div className="space-y-1">
                <Label>{String(i18n.t("contactModal.form.connection") ?? "")}</Label>
                <Select
                  value={values.connection}
                  onValueChange={(v) => setFieldValue("connection", v)}
                  disabled={!!contactId}
                >
                  <SelectTrigger aria-label={String(i18n.t("contactModal.form.connection") ?? "")} aria-invalid={!!(errors.connection && touched.connection)}>
                    <SelectValue placeholder={String(i18n.t("contactModal.form.connectionPlaceholder") ?? "")} />
                  </SelectTrigger>
                  <SelectContent>
                    {connections.map((c) => (
                      <SelectItem key={connectionKey(c)} value={connectionKey(c)}>
                        <span className="flex items-center gap-2">
                          <TicketConnectionBadge connection={{ kind: c.kind, id: c.id, name: c.name, source: c.source }} size="sm" />
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-[hsl(var(--text-muted))]">
                  {String(i18n.t(contactId ? "contactModal.form.connectionLocked" : "contactModal.form.connectionHint") ?? "")}
                </p>
                {errors.connection && touched.connection && (
                  <p className="text-xs text-destructive">{String(i18n.t("contactModal.form.connectionRequired") ?? "")}</p>
                )}
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-1">
                  <Label>{i18n.t("contactModal.form.name")}</Label>
                  <Field as={Input} name="name" autoFocus aria-invalid={!!(errors.name && touched.name)} />
                  {errors.name && touched.name && <p className="text-xs text-destructive">{errors.name as string}</p>}
                </div>
                <div className="space-y-1">
                  <Label>{i18n.t("contactModal.form.number")}</Label>
                  <Field as={Input} name="number" placeholder="5513912344321" aria-invalid={!!(errors.number && touched.number)} />
                  {errors.number && touched.number && <p className="text-xs text-destructive">{errors.number as string}</p>}
                </div>
              </div>

              <div className="space-y-1">
                <Label>{i18n.t("contactModal.form.email")}</Label>
                <Field as={Input} name="email" aria-invalid={!!(errors.email && touched.email)} />
                {errors.email && touched.email && <p className="text-xs text-destructive">{errors.email as string}</p>}
              </div>

              <div className="pt-4 border-t">
                <p className="text-sm font-semibold mb-2">{i18n.t("contactModal.form.extraInfo")}</p>
                <FieldArray name="extraInfo">
                  {({ push, remove }) => (
                    <div className="space-y-2">
                      {values.extraInfo && values.extraInfo.length > 0 &&
                        values.extraInfo.map((info: ExtraInfo, index: number) => (
                          <div key={index} className="flex gap-2 items-start">
                            <Field as={Input} name={`extraInfo[${index}].name`} placeholder={i18n.t("contactModal.form.extraName")} className="flex-1" />
                            <Field as={Input} name={`extraInfo[${index}].value`} placeholder={i18n.t("contactModal.form.extraValue")} className="flex-1" />
                            <Button type="button" variant="ghost" size="icon" onClick={() => remove(index)}>
                              <Trash2 className="h-4 w-4 text-destructive" />
                            </Button>
                          </div>
                        ))}
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        className="mt-2"
                        onClick={() => push({ name: "", value: "" })}
                      >
                        <Plus className="h-4 w-4 mr-1" /> {i18n.t("contactModal.buttons.addExtraInfo")}
                      </Button>
                    </div>
                  )}
                </FieldArray>
              </div>

              <DialogFooter className="pt-4">
                <Button type="button" variant="outline" onClick={handleClose}>
                  {i18n.t("contactModal.buttons.cancel")}
                </Button>
                {contactId && (
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={async () => {
                      try {
                        await api.post(`/contacts/${contactId}/sync`);
                        toast.success("Sync scheduled!");
                      } catch (err) {
                        toastError(err);
                      }
                    }}
                  >
                    Sync
                  </Button>
                )}
                <Button type="submit" disabled={isSubmitting}>
                  {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  {contactId ? i18n.t("contactModal.buttons.okEdit") : i18n.t("contactModal.buttons.okAdd")}
                </Button>
              </DialogFooter>
            </Form>
          )}
        </Formik>
      </DialogContent>
    </Dialog>
  );
};

export default ContactModal;
