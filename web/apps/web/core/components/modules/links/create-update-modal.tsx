/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
// nerve imports
import type { ModuleLink } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { ModalCore, ModalTitle } from "@nerve/ui";
// hooks
import { useModules } from "@/hooks/store/use-modules";
// lib
import { useFormRefusal } from "@/lib/form-refusal";
import { followInSession } from "@/lib/in-session";
import { useRefusedFocus } from "@/lib/refused-focus";

/** What the form holds of a link: its address and its title, empty for none. */
type LinkFormValues = { url: string; title: string };
type LinkFormField = keyof LinkFormValues;
/** Where the form says a message of nerve's refusal: under the field (its setError). */
type LinkUnderField = (field: LinkFormField, message: string) => void;

/**
 * The fields under which nerve's refusals show (422 validation_failed); the others show in a toast, a link the module
 * has already (planning.module_link_exists) among them.
 */
const FIELDS: readonly LinkFormField[] = ["url", "title"];
const TAKEN: Readonly<Record<string, LinkFormField>> = {};

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The module a new link goes to. */
  moduleId: string;
  /** The link to change, as it was as the dialog opened; none, to add one. */
  link?: ModuleLink;
};

/**
 * The dialog that adds a link to a module, or changes one of its links (P6a spec 2.8, R11): its address, required, sent
 * as typed (nerve trims it and gives it the scheme it lacks), and its title, empty for none. It closes only once nerve
 * has added or changed the link, and says so; refused, it stays with what it holds, nerve's reason under the field it
 * names (422) or in a toast (a link the module has already, an archived module, modules turned off, a role that may
 * not, nerve busy; errorMessageKey). While what it sent is out, nothing closes it. Its follow-ups are those of the
 * session and the sidebar it was sent from (followInSession): once it has gone with its sidebar, nerve's late answer
 * reaches the store alone. Closed, it gives the focus back to what opened it.
 */
export function ModuleLinkModal(props: Props) {
  const { isOpen, handleClose, moduleId, link } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { addLink, changeLink } = useModules();
  const showRefusal = useFormRefusal(FIELDS, TAKEN);
  // states
  const [busy, setBusy] = useState(false);
  const out = useRef(false);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const submit = async ({ url, title }: LinkFormValues, underField: LinkUnderField) => {
    if (out.current) return;
    out.current = true;
    setBusy(true);
    const body = { url, title: title === "" ? null : title };
    await followInSession(() => (link ? changeLink(link.id, body) : addLink(moduleId, body)), {
      done: () => {
        if (!mounted.current) return;
        const toast = link ? "links.toasts.updated" : "links.toasts.created";
        setToast({ type: TOAST_TYPE.SUCCESS, title: t(`${toast}.title`), message: t(`${toast}.message`) });
        handleClose();
      },
      failed: (error) => {
        if (mounted.current) showRefusal(error, underField);
      },
    });
    out.current = false;
    if (mounted.current) setBusy(false);
  };

  return (
    <ModalCore isOpen={isOpen} handleClose={busy ? undefined : handleClose}>
      <LinkForm key={link?.id ?? "new"} link={link} busy={busy} onSubmit={submit} onCancel={handleClose} />
    </ModalCore>
  );
}

type FormProps = {
  /** The link a change starts from; none, for a new one. */
  link?: ModuleLink;
  /** Whether what the form sent is out: it is not cancelled until nerve has answered. */
  busy: boolean;
  /** Sends what the form holds; settles once the dialog has followed nerve's answer. */
  onSubmit: (values: LinkFormValues, underField: LinkUnderField) => Promise<void>;
  onCancel: () => void;
};

/**
 * A link's form: its address, which may not be left empty (or blank), and its title. A field its rule refuses, or one
 * nerve refuses, shows its message under it and takes the focus (useRefusedFocus); nothing it holds is reset by a
 * refusal.
 */
function LinkForm(props: FormProps) {
  const { link, busy, onSubmit, onCancel } = props;
  // nerve hooks
  const { t } = useTranslation();
  // form
  const {
    control,
    handleSubmit,
    setError,
    formState: { errors, submitCount },
  } = useForm<LinkFormValues>({
    defaultValues: { url: link?.url ?? "", title: link?.title ?? "" },
    shouldFocusError: false,
  });
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  const underField: LinkUnderField = (field, message) => setError(field, { type: "server", message });

  return (
    <form ref={formRef} onSubmit={(event) => void handleSubmit((values) => onSubmit(values, underField))(event)}>
      <div className="space-y-5 p-5">
        <ModalTitle className="text-18 font-medium text-secondary">
          {link ? t("common.update_link") : t("common.add_link")}
        </ModalTitle>
        <div className="mt-2 space-y-3">
          <div>
            <label htmlFor="url" className="mb-2 text-secondary">
              {t("common.url")}
            </label>
            <Controller
              control={control}
              name="url"
              rules={{ validate: (value) => value.trim() !== "" || t("common.errors.required") }}
              render={({ field: { value, onChange, ref } }) => (
                <Field name="url" invalid={Boolean(errors.url)}>
                  <InputGroup size="2xl">
                    <Input
                      size="2xl"
                      id="url"
                      type="text"
                      value={value}
                      onChange={onChange}
                      ref={ref}
                      placeholder={t("common.type_or_paste_a_url")}
                    />
                  </InputGroup>
                  {errors.url?.message && (
                    <FieldError size="2xl" match>
                      {errors.url.message}
                    </FieldError>
                  )}
                </Field>
              )}
            />
          </div>
          <div>
            <label htmlFor="title" className="mb-2 text-secondary">
              {t("common.display_title")}
              <span className="block text-10">{t("common.optional")}</span>
            </label>
            <Controller
              control={control}
              name="title"
              render={({ field: { value, onChange, ref } }) => (
                <Field name="title" invalid={Boolean(errors.title)}>
                  <InputGroup size="2xl">
                    <Input
                      size="2xl"
                      id="title"
                      type="text"
                      value={value}
                      onChange={onChange}
                      ref={ref}
                      placeholder={t("common.link_title_placeholder")}
                    />
                  </InputGroup>
                  {errors.title?.message && (
                    <FieldError size="2xl" match>
                      {errors.title.message}
                    </FieldError>
                  )}
                </Field>
              )}
            />
          </div>
        </div>
      </div>
      <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
        <Button type="button" variant="secondary" size="lg" disabled={busy} onClick={onCancel}>
          {t("common.cancel")}
        </Button>
        <Button variant="primary" size="lg" type="submit" loading={busy}>
          {link ? t("common.update_link") : t("common.add_link")}
        </Button>
      </div>
    </form>
  );
}
