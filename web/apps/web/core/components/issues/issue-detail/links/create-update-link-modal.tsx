/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, useState } from "react";
import type { FormEvent } from "react";
import type { IssueLink } from "@nerve/api-client";
import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { useTranslation } from "@nerve/i18n";
// nerve types
import { Button } from "@nerve/propel/button";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useFormRefusal } from "@/lib/form-refusal";
import { useRefusedFocus } from "@/lib/refused-focus";
import { useDetailScope } from "../../issue-detail-widgets/use-detail-scope";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
// nerve ui
import { ModalCore, ModalTitle } from "@nerve/ui";
// hooks
import { useIssueDetail, useIssueDetailModals } from "@/hooks/store/use-issue-detail";
// types
import type { TLinkOperations } from "./types";

type TLinkOperationsModal = Pick<TLinkOperations, "create" | "update">;

export type TIssueLinkCreateFormFieldOptions = {
  title: string;
  url: string;
  id?: string;
};

export type TIssueLinkCreateEditModal = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isModalOpen: boolean;
  handleOnClose?: () => void;
  linkOperations: TLinkOperationsModal;
};

const defaultValues: TIssueLinkCreateFormFieldOptions = {
  title: "",
  url: "",
};

export const IssueLinkCreateUpdateModal = observer(function IssueLinkCreateUpdateModal(
  props: TIssueLinkCreateEditModal
) {
  const { rootIssueStore: root } = useIssueDetail();
  const { issueLinkData } = useIssueDetailModals();
  const preloadedData = issueLinkData?.issue_id === props.issueId ? issueLinkData : null;
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(props.workspaceSlug);
  const permission = root.permissionReads.identity(workspace?.id, [props.projectId], true);
  return props.isModalOpen ? (
    <LinkForm
      key={JSON.stringify([props.workspaceSlug, props.projectId, props.issueId, preloadedData?.id, permission])}
      {...props}
      preloadedData={preloadedData}
    />
  ) : null;
});

function LinkForm({
  workspaceSlug,
  projectId,
  issueId,
  isModalOpen,
  handleOnClose,
  linkOperations,
  preloadedData,
}: TIssueLinkCreateEditModal & { preloadedData: IssueLink | null }) {
  // i18n
  const { t } = useTranslation();
  // react hook form
  const {
    formState: { errors, submitCount },
    handleSubmit,
    control,
    setError,
  } = useForm<TIssueLinkCreateFormFieldOptions>({
    defaultValues: preloadedData
      ? { id: preloadedData.id, url: preloadedData.url, title: preloadedData.title ?? "" }
      : defaultValues,
    shouldFocusError: false,
  });
  // store hooks
  const { setIssueLinkData } = useIssueDetailModals();
  const capture = useDetailScope(workspaceSlug, projectId, issueId, true);
  const sending = useRef(false);
  const [isSubmitting, setSubmitting] = useState(false);
  const formRef = useRefusedFocus<HTMLFormElement>(submitCount);
  const refusal = useFormRefusal(["url", "title"], { "issue.link_exists": "url" });

  const finishClose = () => {
    setIssueLinkData(null);
    if (handleOnClose) handleOnClose();
  };
  const onClose = () => {
    if (!sending.current) finishClose();
  };

  const handleFormSubmit = async (formData: TIssueLinkCreateFormFieldOptions) => {
    const current = capture();
    if (!current()) return;
    // The API owns scheme normalization and validation; retain its exact uniqueness semantics.
    const body = { title: formData.title || null, url: formData.url.trim() };
    await followInSession(
      () => (formData.id ? linkOperations.update(formData.id, body) : linkOperations.create(body)),
      {
        done: () => {
          if (!current()) return;
          const verb = formData.id ? "updated" : "created";
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t(`links.toasts.${verb}.title`),
            message: t(`links.toasts.${verb}.message`),
          });
          finishClose();
        },
        failed: (error) => {
          if (current() && !(error instanceof SessionChangedError))
            refusal(error, (field, message) => setError(field, { type: "server", message }));
        },
      }
    );
  };
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const current = capture();
    if (!current() || sending.current) return;
    sending.current = true;
    setSubmitting(true);
    try {
      await handleSubmit(handleFormSubmit)(event);
    } finally {
      sending.current = false;
      if (current()) setSubmitting(false);
    }
  };

  return (
    <ModalCore isOpen={isModalOpen} handleClose={onClose}>
      <form ref={formRef} onSubmit={(event) => void submit(event)}>
        <div className="space-y-5 p-5">
          <ModalTitle className="text-h4-medium text-secondary">
            {preloadedData?.id ? t("common.update_link") : t("common.add_link")}
          </ModalTitle>
          <div className="mt-2 space-y-3">
            <div>
              <label htmlFor="url" className="mb-2 text-secondary">
                {t("common.url")}
              </label>
              <Controller
                control={control}
                name="url"
                rules={{
                  validate: (value) => value.trim().length > 0 || t("common.url_is_invalid"),
                }}
                render={({ field: { value, onChange, ref } }) => (
                  <Field name="url" invalid={Boolean(errors.url)}>
                    <InputGroup size="2xl">
                      <Input
                        size="2xl"
                        id="url"
                        type="text"
                        disabled={isSubmitting}
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
                <span className="block text-caption-xs-regular">{t("common.optional")}</span>
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
                        disabled={isSubmitting}
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
          <Button type="button" variant="secondary" size="lg" disabled={isSubmitting} onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
            {`${
              preloadedData?.id
                ? isSubmitting
                  ? t("common.updating")
                  : t("common.update")
                : isSubmitting
                  ? t("common.adding")
                  : t("common.add")
            } ${t("common.link")}`}
          </Button>
        </div>
      </form>
    </ModalCore>
  );
}
