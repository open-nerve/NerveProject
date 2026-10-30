/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef } from "react";
import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { useTranslation } from "@nerve/i18n";
// nerve types
import { Button } from "@nerve/propel/button";
import { followInSession } from "@/lib/in-session";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useDetailScope } from "../../issue-detail-widgets/use-detail-scope";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
// nerve ui
import { ModalCore } from "@nerve/ui";
// hooks
import { useIssueDetailModals } from "@/hooks/store/use-issue-detail";
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
  const { workspaceSlug, projectId, issueId, isModalOpen, handleOnClose, linkOperations } = props;
  // i18n
  const { t } = useTranslation();
  // react hook form
  const {
    formState: { errors, isSubmitting },
    handleSubmit,
    control,
    reset,
  } = useForm<TIssueLinkCreateFormFieldOptions>({
    defaultValues,
  });
  // store hooks
  const { setIssueLinkData } = useIssueDetailModals();
  const { issueLinkData } = useIssueDetailModals();
  const preloadedData = issueLinkData?.issue_id === issueId ? issueLinkData : null;
  const capture = useDetailScope(workspaceSlug, projectId, `${issueId}:${isModalOpen}:${preloadedData?.id ?? "new"}`);
  const sending = useRef(false);
  const refusal = useRefusalToast();

  const onClose = () => {
    setIssueLinkData(null);
    if (handleOnClose) handleOnClose();
  };

  const handleFormSubmit = async (formData: TIssueLinkCreateFormFieldOptions) => {
    const current = capture();
    if (!current() || sending.current) return;
    sending.current = true;
    const parsedUrl = /^https?:\/\//i.test(formData.url) ? formData.url : `http://${formData.url}`;
    const body = { title: formData.title || null, url: parsedUrl };
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
          onClose();
        },
        failed: (error) => {
          if (current() && !(error instanceof SessionChangedError)) refusal(error);
        },
      }
    );
    if (current()) sending.current = false;
  };

  useEffect(() => {
    sending.current = false;
    if (isModalOpen)
      reset(
        preloadedData
          ? { id: preloadedData.id, url: preloadedData.url, title: preloadedData.title ?? "" }
          : defaultValues
      );
  }, [preloadedData, reset, isModalOpen, issueId]);

  return (
    <ModalCore isOpen={isModalOpen} handleClose={onClose}>
      <form onSubmit={handleSubmit(handleFormSubmit)}>
        <div className="space-y-5 p-5">
          <h3 className="text-h4-medium text-secondary">
            {preloadedData?.id ? t("common.update_link") : t("common.add_link")}
          </h3>
          <div className="mt-2 space-y-3">
            <div>
              <label htmlFor="url" className="mb-2 text-secondary">
                {t("common.url")}
              </label>
              <Controller
                control={control}
                name="url"
                rules={{
                  required: "URL is required",
                }}
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
                  </Field>
                )}
              />
              {errors.url && (
                <span className="text-caption-sm-regular text-danger-primary">{t("common.url_is_invalid")}</span>
              )}
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
                        value={value}
                        onChange={onChange}
                        ref={ref}
                        placeholder={t("common.link_title_placeholder")}
                      />
                    </InputGroup>
                  </Field>
                )}
              />
            </div>
          </div>
        </div>
        <div className="flex items-center justify-end gap-2 border-t-[0.5px] border-subtle px-5 py-4">
          <Button type="button" variant="secondary" size="lg" onClick={onClose}>
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
});
