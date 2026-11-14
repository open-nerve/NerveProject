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
// hooks
import { getUpdateFormDataForReset } from "@nerve/utils";
import { ModalTitle } from "@nerve/ui";
import type { PlanningListRefusal } from "@/hooks/use-planning-names";
import { useRefusedFocus } from "@/lib/refused-focus";
import type { PlanningDomain } from "@/lib/planning-switch";
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
import { DraftFiles, type DraftContent } from "./draft-files";
import { useFormSubmission } from "./form-submission";
import { IssueFormFooter } from "./form-footer";

export interface IssueFormProps {
  data?: IssueFormData;
  sourceIssueId?: string;
  hasDraftSeed?: boolean;
  canSaveDraft?: boolean;
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
  onPublish?: (values: IssueFormData, options?: Pick<IssueFormSubmitOptions, "onRefused">) => Promise<void>;
  modalTitle?: string;
  primaryButtonText?: {
    default: string;
    loading: string;
  };
  isProjectSelectionDisabled?: boolean;
  /** A stored draft's files and images; a form without it creates no file request. */
  content?: DraftContent;
  publishBlocked?: boolean;
  /** The state of a command that outlives one submission, such as a kept copy. */
  notice?: React.ReactNode;
  /** Whether the modal waits for the lists of the project the form opened with (P7a spec 2.6, re-72 F7). */
  planningLoading?: boolean;
  /** Each of those lists nerve refused, which its field says, with a retry. */
  planningRefusals?: Partial<Record<PlanningDomain, PlanningListRefusal>>;
}

export const IssueFormRoot = observer(function IssueFormRoot(props: IssueFormProps) {
  const { t } = useTranslation();
  const {
    data,
    sourceIssueId,
    hasDraftSeed = false,
    canSaveDraft = false,
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
    content,
    publishBlocked = false,
    notice,
    planningLoading = false,
    planningRefusals,
  } = props;

  // states: the form's attempts beside its submit (saved to drafts, a draft published), for a refused field's focus
  const [draftSubmitCount, setDraftSubmitCount] = useState(0);

  // refs
  const submitBtnRef = useRef<HTMLButtonElement | null>(null);
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
    formState: { isDirty, isSubmitting },
    handleSubmit,
    reset,
    watch,
    control,
    getValues,
    setValue,
    getFieldState,
    resetField,
    clearErrors,
    register,
  } = methods;
  const formRef = useRefusedFocus<HTMLFormElement>(formState.submitCount + draftSubmitCount);
  const { isMoving, submissionData, handleFormSubmit, handleMoveToProjects } = useFormSubmission({
    methods,
    data,
    hasDraftSeed,
    isDraft,
    descriptionLoadState,
    editorRef,
    descriptionMigration,
    isCreateMoreToggleEnabled,
    onSubmit,
    onPublish,
    isPending,
  });

  const projectId = watch("project_id");
  // The lists of the project the form opened with, while it holds that project (the fields' and the save's).
  const planningPending = planningLoading && projectId === defaultProjectId;

  // On a change of its project a new work item's form starts over but for a few fields; a draft's leaves its cycle and
  // modules, which are its project's, as an edit, which its save sends (P7a spec 2.8, R6: no cycle, no modules). Back
  // at the project it opened with, they are the draft's own again, as stored: no edit, which its save leaves (re-73 M2).
  useEffect(() => {
    if (isDraft) {
      if (getFieldState("project_id").isDirty) {
        setValue("cycle_id", null, { shouldDirty: true });
        setValue("module_ids", [], { shouldDirty: true });
      } else {
        resetField("cycle_id");
        resetField("module_ids");
      }
    } else if (isDirty)
      reset(getUpdateFormDataForReset(projectId ?? undefined, { ...getValues(), project_id: projectId ?? undefined }));

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  // Each time the lists of the project the form opened with settle (answered, or refused: a retry may answer later),
  // the modal's start of its cycle and modules (data) is what was given that is an option, beside the page's (P7a spec
  // 2.6, re-72 F7): their defaults, not an edit, whether or not fields show them (registered here). A field changed
  // meanwhile keeps its value, a form whose project changed its own.
  const refused = Object.keys(planningRefusals ?? {}).join();
  const planningSettled = useRef(planningLoading ? undefined : refused);
  useEffect(() => {
    if (planningLoading) planningSettled.current = undefined;
    if (planningLoading || planningSettled.current === refused) return;
    planningSettled.current = refused;
    if (getValues("project_id") !== defaultProjectId) return;
    register("cycle_id");
    register("module_ids");
    if (!getFieldState("cycle_id").isDirty) resetField("cycle_id", { defaultValue: data?.cycle_id ?? null });
    if (!getFieldState("module_ids").isDirty) resetField("module_ids", { defaultValue: data?.module_ids ?? [] });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [planningLoading, refused]);

  const condition =
    (watch("name") && watch("name") !== "") || (watch("description_html") && watch("description_html") !== "<p></p>");

  const handleFormChange = () => {
    if (!onChange) return;

    if (isDirty && condition) onChange(submissionData(watch()));
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

    if (isDirty && condition) onChange(submissionData(watch()));
    else onChange(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDirty]);

  return (
    <FormProvider {...methods}>
      <div className="flex gap-2 bg-transparent">
        <div className="w-full rounded-lg">
          <form
            ref={formRef}
            onSubmit={handleSubmit((values) => handleFormSubmit(values))}
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
                    images={content?.images}
                    submitBtnRef={submitBtnRef}
                    workspaceSlug={workspaceSlug}
                    projectId={projectId}
                    handleDescriptionHTMLDataChange={handleDescriptionChange}
                  />
                )}
              </div>
              {content?.view ? (
                <div className="px-5">
                  <DraftFiles content={content} publishBlocked={publishBlocked} />
                </div>
              ) : (
                !data?.id && (
                  <p className="px-5 text-13 text-secondary">
                    {canSaveDraft
                      ? t("content_files.draft.files_after_save")
                      : t("content_files.draft.files_after_create")}
                  </p>
                )
              )}
              {notice && <div className="px-5 text-13">{notice}</div>}
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
                  planningLoading={planningPending}
                  planningRefusals={projectId === defaultProjectId ? planningRefusals : undefined}
                  handleFormChange={handleFormChange}
                  setSelectedParentIssue={setSelectedParentIssue}
                />
              </div>
              <IssueFormFooter
                id={data?.id}
                isDraft={isDraft}
                canSaveDraft={canSaveDraft}
                descriptionLoadState={descriptionLoadState}
                planningLoading={planningPending}
                isCreateMoreToggleEnabled={isCreateMoreToggleEnabled}
                onCreateMoreToggleChange={onCreateMoreToggleChange}
                isSubmitting={isSubmitting}
                isPending={isPending}
                isMoving={isMoving}
                moveToIssue={moveToIssue}
                primaryButtonText={primaryButtonText}
                submitBtnRef={submitBtnRef}
                onClose={onClose}
                onSaveDraft={async () => {
                  clearErrors();
                  await handleFormSubmit(getValues(), true);
                  setDraftSubmitCount((count) => count + 1);
                }}
                onMoveToProjects={async () => {
                  clearErrors();
                  await handleMoveToProjects();
                  setDraftSubmitCount((count) => count + 1);
                }}
              />
            </div>
          </form>
        </div>
      </div>
    </FormProvider>
  );
});
