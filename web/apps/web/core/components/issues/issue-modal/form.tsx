// oxlint-disable promise/always-return
// oxlint-disable no-shadow
// oxlint-disable jsx_a11y/prefer-tag-over-role
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { pick } from "lodash-es";
import { ISSUE_FORM_FIELDS } from "./types";
import type {
  IssueFormData,
  IssueFormValues,
  IssueDescriptionLoadState,
  IssueDescriptionChangeSource,
  IssueFormSubmitOptions,
} from "./types";
import React, { useState, useRef, useEffect } from "react";
import { observer } from "mobx-react";
import { FormProvider, useForm } from "react-hook-form";
// editor
import { DEFAULT_WORK_ITEM_FORM_VALUES } from "@nerve/constants";
import type { EditorRefApi } from "@nerve/editor";
// i18n
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { Switch } from "@makeplane/propel/components/switch";
import { getUpdateFormDataForReset, getChangedIssuefields } from "@nerve/utils";
import { ModalTitle } from "@nerve/ui";
import { useRefusedFocus } from "@/lib/refused-focus";
import { useFormRefusal } from "@/lib/form-refusal";
// components
import {
  IssueDefaultProperties,
  IssueDescriptionEditor,
  IssueParentTag,
  IssueProjectSelect,
  IssueTitleInput,
} from "@/components/issues/issue-modal/components";
// hooks
import { useIssueModal } from "@/hooks/context/use-issue-modal";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";

export interface IssueFormProps {
  data?: IssueFormData;
  sourceIssueId?: string;
  descriptionLoadState?: IssueDescriptionLoadState;
  onRetryDescription?: () => void;
  issueTitleRef: React.MutableRefObject<HTMLInputElement | null>;
  editorRef: React.RefObject<EditorRefApi | null>;
  isCreateMoreToggleEnabled: boolean;
  onCreateMoreToggleChange: (value: boolean) => void;
  onChange?: (formData: Partial<IssueFormValues> | null) => void;
  onClose: () => void;
  onSubmit: (
    values: Partial<IssueFormValues>,
    is_draft_issue?: boolean,
    options?: IssueFormSubmitOptions
  ) => Promise<void>;
  isPending?: boolean;
  workspaceSlug: string;
  projectId: string | null;
  isDraft: boolean;
  moveToIssue?: boolean;
  onPublish?: (values: IssueFormData) => Promise<void>;
  modalTitle?: string;
  primaryButtonText?: {
    default: string;
    loading: string;
  };
  isDuplicateModalOpen: boolean;
  handleDuplicateIssueModal: (isOpen: boolean) => void;
  isProjectSelectionDisabled?: boolean;
}

export const IssueFormRoot = observer(function IssueFormRoot(props: IssueFormProps) {
  const { t } = useTranslation();
  const {
    data,
    sourceIssueId,
    descriptionLoadState = "ready",
    onRetryDescription,
    issueTitleRef,
    editorRef,
    onChange,
    onClose,
    onSubmit,
    isPending = false,
    onPublish,
    workspaceSlug,
    projectId: defaultProjectId,
    isCreateMoreToggleEnabled,
    onCreateMoreToggleChange,
    isDraft,
    moveToIssue = false,
    modalTitle = `${data?.id ? t("update") : isDraft ? t("create_a_draft") : t("create_new_issue")}`,
    primaryButtonText = {
      default: `${data?.id ? t("update") : isDraft ? t("save_to_drafts") : t("save")}`,
      loading: `${data?.id ? t("updating") : t("saving")}`,
    },
    isProjectSelectionDisabled = false,
  } = props;

  // states
  const [isMoving, setIsMoving] = useState<boolean>(false);

  // refs
  const submitBtnRef = useRef<HTMLButtonElement | null>(null);
  const modalContainerRef = useRef<HTMLDivElement | null>(null);
  const descriptionMigration = useRef(false);

  // store hooks
  const { selectedParentIssue, setSelectedParentIssue } = useIssueModal();

  const {
    issue: { getIssueById },
  } = useIssueDetail();

  // form info
  const methods = useForm<IssueFormValues>({
    defaultValues: { ...DEFAULT_WORK_ITEM_FORM_VALUES, project_id: defaultProjectId, ...pick(data, ISSUE_FORM_FIELDS) },
    reValidateMode: "onChange",
    shouldFocusError: false,
  });
  const {
    formState,
    formState: { isDirty, isSubmitting, dirtyFields },
    handleSubmit,
    reset,
    watch,
    control,
    getValues,
    setValue,
    setError,
    getFieldState,
    resetField,
  } = methods;
  const formRef = useRefusedFocus<HTMLFormElement>(formState.submitCount);
  const refuse = useFormRefusal(["name", "project_id", "start_date", "target_date"] as const, {});

  const projectId = watch("project_id");

  //reset few fields on projectId change
  useEffect(() => {
    if (isDirty && !isDraft)
      reset(getUpdateFormDataForReset(projectId ?? undefined, { ...getValues(), project_id: projectId ?? undefined }));

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  const handleFormSubmit = async (formData: Partial<IssueFormValues>, is_draft_issue = isDraft) => {
    if (descriptionLoadState !== "ready") return;
    // Check if the editor is ready to discard
    if (!editorRef.current?.isEditorReadyToDiscard()) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("error"),
        message: t("editor_is_not_ready_to_discard_changes"),
      });
      return;
    }

    const submitData = !data?.id
      ? formData
      : {
          ...getChangedIssuefields(formData, dirtyFields as { [key: string]: boolean | undefined }),
          project_id: getValues<"project_id">("project_id"),
          id: data.id,
          description_html: formData.description_html ?? "<p></p>",
        };

    await onSubmit(submitData, is_draft_issue, {
      isDescriptionMigration: descriptionMigration.current,
      onRefused: (error) => refuse(error, (field, message) => setError(field, { type: "server", message })),
    })
      .then(() => {
        descriptionMigration.current = false;
        reset({
          ...DEFAULT_WORK_ITEM_FORM_VALUES,
          ...(isCreateMoreToggleEnabled ? pick(data, ISSUE_FORM_FIELDS) : {}),
          project_id: getValues<"project_id">("project_id"),
          description_html: data?.description_html ?? "<p></p>",
        });
        editorRef?.current?.clearEditor();
      })
      .catch(() => undefined);
  };

  const handleMoveToProjects = async () => {
    if (!data?.id || !onPublish || isMoving || isSubmitting || isPending || descriptionLoadState !== "ready") return;
    if (!editorRef.current?.isEditorReadyToDiscard()) return;
    setIsMoving(true);
    try {
      await onPublish({
        ...getChangedIssuefields(getValues(), dirtyFields as { [key: string]: boolean | undefined }),
        project_id: getValues("project_id"),
        description_html: getValues("description_html"),
      });
    } catch {
      /* The command owner displays the current refusal; retain the form. */
    } finally {
      setIsMoving(false);
    }
  };

  const condition =
    (watch("name") && watch("name") !== "") || (watch("description_html") && watch("description_html") !== "<p></p>");

  const handleFormChange = () => {
    if (!onChange) return;

    if (isDirty && condition) onChange(watch());
    else onChange(null);
  };

  const handleDescriptionChange = (html: string, source: IssueDescriptionChangeSource = "load") => {
    const edited = getFieldState("description_html").isDirty;
    if (source === "edit") {
      descriptionMigration.current = false;
      setValue("description_html", html, { shouldDirty: true });
    } else if (!edited) {
      // An initial load or a pure ID migration becomes the field's baseline, preserving all other dirty fields.
      resetField("description_html", { defaultValue: html });
      descriptionMigration.current = source === "migration";
    } else if (source === "migration") {
      setValue("description_html", html, { shouldDirty: true });
    }
    handleFormChange();
  };

  const parentId = watch("parent_id");
  const parent = parentId ? getIssueById(parentId) : undefined;
  useEffect(() => {
    if (selectedParentIssue === (parent ?? null)) return;
    setSelectedParentIssue(parent ?? null);
  }, [parent, selectedParentIssue, setSelectedParentIssue]);

  // executing this useEffect when isDirty changes
  useEffect(() => {
    if (!onChange) return;

    if (isDirty && condition) onChange(watch());
    else onChange(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDirty]);

  useEffect(() => {
    const formElement = formRef?.current;
    const modalElement = modalContainerRef?.current;

    if (!formElement || !modalElement) return;

    const resizeObserver = new ResizeObserver(() => {
      modalElement.style.maxHeight = `${formElement?.offsetHeight}px`;
    });

    resizeObserver.observe(formElement);

    return () => {
      resizeObserver.disconnect();
    };
  }, [formRef, modalContainerRef]);

  return (
    <FormProvider {...methods}>
      <div className="flex gap-2 bg-transparent">
        <div className="w-full rounded-lg">
          <form
            ref={formRef}
            onSubmit={handleSubmit((data) => handleFormSubmit(data))}
            className="flex w-full flex-col"
          >
            <div className="rounded-t-lg bg-surface-1 p-5">
              <ModalTitle className="pb-2 text-h4-medium text-secondary">{modalTitle}</ModalTitle>
              <div className="flex items-center justify-between pt-2 pb-4">
                <div className="flex items-center gap-x-1">
                  <IssueProjectSelect
                    control={control}
                    disabled={(!isDraft && !!data?.id) || !!sourceIssueId || isProjectSelectionDisabled}
                    isDraft={isDraft}
                    handleFormChange={handleFormChange}
                  />
                </div>
              </div>
              {watch("parent_id") && selectedParentIssue && (
                <div className="pb-4">
                  <IssueParentTag
                    control={control}
                    selectedParentIssue={selectedParentIssue}
                    handleFormChange={handleFormChange}
                    setSelectedParentIssue={setSelectedParentIssue}
                  />
                </div>
              )}
              <div className="space-y-1">
                <IssueTitleInput
                  control={control}
                  issueTitleRef={issueTitleRef}
                  isDraft={isDraft}
                  formState={formState}
                  handleFormChange={handleFormChange}
                />
              </div>
            </div>
            <div className="space-y-3 bg-surface-1 pb-4">
              <div className="px-5">
                {descriptionLoadState === "error" ? (
                  <div role="alert" className="space-y-2 rounded-md border border-danger-subtle p-3">
                    <p>{t("entity.fetch.failed", { entity: t("description") })}</p>
                    <Button type="button" variant="secondary" onClick={onRetryDescription}>
                      {t("common.retry")}
                    </Button>
                  </div>
                ) : (
                  <IssueDescriptionEditor
                    control={control}
                    isDraft={isDraft}
                    descriptionHtmlData={data?.description_html}
                    editorRef={editorRef}
                    submitBtnRef={submitBtnRef}
                    workspaceSlug={workspaceSlug}
                    projectId={projectId}
                    handleDescriptionHTMLDataChange={handleDescriptionChange}
                  />
                )}
              </div>
            </div>
            <div className="rounded-b-lg border-t-[0.5px] border-subtle bg-surface-1 px-4 py-3">
              <div className="pb-3">
                <IssueDefaultProperties
                  control={control}
                  id={data?.id}
                  projectId={projectId}
                  workspaceSlug={workspaceSlug}
                  selectedParentIssue={selectedParentIssue}
                  parentId={watch("parent_id") ?? null}
                  isDraft={isDraft}
                  handleFormChange={handleFormChange}
                  setSelectedParentIssue={setSelectedParentIssue}
                />
              </div>
              <div className="flex items-center justify-end gap-4 border-t-[0.5px] border-subtle pt-6 pb-3">
                {!data?.id && (
                  <div className="inline-flex items-center gap-1.5">
                    <Switch
                      size="sm"
                      checked={isCreateMoreToggleEnabled}
                      onCheckedChange={onCreateMoreToggleChange}
                      disabled={isSubmitting || isPending}
                      aria-label={t("create_more")}
                    />
                    <span className="text-caption-sm-regular">{t("create_more")}</span>
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <div>
                    <Button
                      variant="secondary"
                      size="lg"
                      disabled={isSubmitting || isPending || isMoving}
                      onClick={onClose}
                    >
                      {t("discard")}
                    </Button>
                  </div>
                  <div>
                    <Button
                      variant={moveToIssue ? "secondary" : "primary"}
                      size="lg"
                      type="submit"
                      ref={submitBtnRef}
                      loading={isSubmitting}
                      disabled={isSubmitting || isPending || isMoving || descriptionLoadState !== "ready"}
                    >
                      {isSubmitting ? primaryButtonText.loading : primaryButtonText.default}
                    </Button>
                  </div>

                  {moveToIssue && (
                    <Button
                      variant="primary"
                      type="button"
                      loading={isMoving}
                      onClick={handleMoveToProjects}
                      disabled={isMoving || isSubmitting || isPending || descriptionLoadState !== "ready"}
                      size="lg"
                    >
                      {t("add_to_project")}
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </form>
        </div>
      </div>
    </FormProvider>
  );
});
