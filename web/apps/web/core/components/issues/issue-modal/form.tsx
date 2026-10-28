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
import type { IssueFormData, IssueFormValues, IssueDescriptionLoadState } from "./types";
import React, { useState, useRef, useEffect } from "react";
import { observer } from "mobx-react";
import { FormProvider, useForm } from "react-hook-form";
// editor
import { ETabIndices, DEFAULT_WORK_ITEM_FORM_VALUES } from "@nerve/constants";
import type { EditorRefApi } from "@nerve/editor";
// i18n
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { Switch } from "@makeplane/propel/components/switch";
import {
  convertWorkItemDataToSearchResponse,
  getUpdateFormDataForReset,
  getChangedIssuefields,
  getTabIndex,
} from "@nerve/utils";
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
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
import { usePlatformOS } from "@/hooks/use-platform-os";

export interface IssueFormProps {
  data?: IssueFormData;
  sourceIssueId?: string;
  descriptionLoadState?: IssueDescriptionLoadState;
  onRetryDescription?: () => void;
  issueTitleRef: React.MutableRefObject<HTMLInputElement | null>;
  isCreateMoreToggleEnabled: boolean;
  onAssetUpload: (assetId: string) => void;
  onCreateMoreToggleChange: (value: boolean) => void;
  onChange?: (formData: Partial<IssueFormValues> | null) => void;
  onClose: () => void;
  onSubmit: (values: Partial<IssueFormValues>, is_draft_issue?: boolean) => Promise<void>;
  workspaceSlug: string;
  projectId: string;
  isDraft: boolean;
  moveToIssue?: boolean;
  modalTitle?: string;
  primaryButtonText?: {
    default: string;
    loading: string;
  };
  isDuplicateModalOpen: boolean;
  handleDuplicateIssueModal: (isOpen: boolean) => void;
  handleDraftAndClose?: () => void;
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
    onAssetUpload,
    onChange,
    onClose,
    onSubmit,
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
  const editorRef = useRef<EditorRefApi>(null);
  const submitBtnRef = useRef<HTMLButtonElement | null>(null);
  const formRef = useRef<HTMLFormElement | null>(null);
  const modalContainerRef = useRef<HTMLDivElement | null>(null);

  // store hooks
  const { getProjectById } = useProject();
  const { selectedParentIssue, setSelectedParentIssue } = useIssueModal();
  const { isMobile } = usePlatformOS();
  const { moveIssue } = useWorkspaceDraftIssues();

  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const { getStateById } = useProjectState();

  // form info
  const methods = useForm<IssueFormValues>({
    defaultValues: { ...DEFAULT_WORK_ITEM_FORM_VALUES, project_id: defaultProjectId, ...pick(data, ISSUE_FORM_FIELDS) },
    reValidateMode: "onChange",
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
  } = methods;

  const projectId = watch("project_id");

  const { getIndex } = getTabIndex(ETabIndices.ISSUE_FORM, isMobile);

  //reset few fields on projectId change
  useEffect(() => {
    if (isDirty) reset(getUpdateFormDataForReset(projectId, getValues()));

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  const handleFormSubmit = async (formData: Partial<IssueFormValues>, is_draft_issue = false) => {
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

    await onSubmit(submitData, is_draft_issue)
      .then(() => {
        reset({
          ...DEFAULT_WORK_ITEM_FORM_VALUES,
          ...(isCreateMoreToggleEnabled ? pick(data, ISSUE_FORM_FIELDS) : {}),
          project_id: getValues<"project_id">("project_id"),
          description_html: data?.description_html ?? "<p></p>",
        });
        editorRef?.current?.clearEditor();
      })
      .catch((error) => {
        console.error(error);
      });
  };

  const handleMoveToProjects = async () => {
    if (!data?.id || !data?.project_id || !data || descriptionLoadState !== "ready") return;
    setIsMoving(true);
    try {
      await moveIssue(workspaceSlug, data.id, {
        ...data,
        ...getValues(),
      });
    } catch {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to move work item to project. Please try again.",
      });
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

  // executing this useEffect when the parent_id coming from the component prop
  useEffect(() => {
    const parentId = watch("parent_id") || undefined;
    if (!parentId) return;
    if (parentId === selectedParentIssue?.id || selectedParentIssue) return;

    const issue = getIssueById(parentId);
    if (!issue) return;

    const projectDetails = getProjectById(issue.project_id);
    if (!projectDetails) return;

    const stateDetails = getStateById(issue.state_id);

    setSelectedParentIssue(convertWorkItemDataToSearchResponse(workspaceSlug, issue, projectDetails, stateDetails));
    // oxlint-disable-next-line eslint-plugin-react-hooks/exhaustive-deps
  }, [watch, getIssueById, getProjectById, selectedParentIssue, getStateById]);

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
              <h3 className="pb-2 text-h4-medium text-secondary">{modalTitle}</h3>
              <div className="flex items-center justify-between pt-2 pb-4">
                <div className="flex items-center gap-x-1">
                  <IssueProjectSelect
                    control={control}
                    disabled={!!data?.id || !!sourceIssueId || isProjectSelectionDisabled}
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
                    issueId={data?.id}
                    descriptionHtmlData={data?.description_html}
                    editorRef={editorRef}
                    submitBtnRef={submitBtnRef}
                    workspaceSlug={workspaceSlug}
                    projectId={projectId}
                    handleFormChange={handleFormChange}
                    handleDescriptionHTMLDataChange={(description_html) => {
                      setValue<"description_html">("description_html", description_html);
                      handleFormChange();
                    }}
                    onAssetUpload={onAssetUpload}
                    onClose={onClose}
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
                  startDate={watch("start_date") ?? null}
                  targetDate={watch("target_date") ?? null}
                  parentId={watch("parent_id") ?? null}
                  isDraft={isDraft}
                  handleFormChange={handleFormChange}
                  setSelectedParentIssue={setSelectedParentIssue}
                />
              </div>
              <div
                className="flex items-center justify-end gap-4 border-t-[0.5px] border-subtle pt-6 pb-3"
                tabIndex={getIndex("create_more")}
              >
                {!data?.id && (
                  <div
                    className="inline-flex cursor-pointer items-center gap-1.5"
                    onClick={() => onCreateMoreToggleChange(!isCreateMoreToggleEnabled)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") onCreateMoreToggleChange(!isCreateMoreToggleEnabled);
                    }}
                    role="button"
                  >
                    <Switch
                      size="sm"
                      checked={isCreateMoreToggleEnabled}
                      onCheckedChange={() => {}}
                      aria-label={t("create_more")}
                    />
                    <span className="text-caption-sm-regular">{t("create_more")}</span>
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <div tabIndex={getIndex("discard_button")}>
                    <Button
                      variant="secondary"
                      size="lg"
                      onClick={() => {
                        if (descriptionLoadState !== "ready" || editorRef.current?.isEditorReadyToDiscard()) {
                          onClose();
                        } else {
                          setToast({
                            type: TOAST_TYPE.ERROR,
                            title: "Error!",
                            message: "Editor is still processing changes. Please wait before proceeding.",
                          });
                        }
                      }}
                    >
                      {t("discard")}
                    </Button>
                  </div>
                  <div tabIndex={isDraft ? getIndex("submit_button") : getIndex("draft_button")}>
                    <Button
                      variant={moveToIssue ? "secondary" : "primary"}
                      size="lg"
                      type="submit"
                      ref={submitBtnRef}
                      loading={isSubmitting}
                      disabled={isSubmitting || descriptionLoadState !== "ready"}
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
                      disabled={isMoving || descriptionLoadState !== "ready"}
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
