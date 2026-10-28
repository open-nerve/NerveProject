/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import type { IssueFormData, IssueDescriptionLoadState } from "./types";
import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// Nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { TWorkspaceDraftIssue } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useIssueModal } from "@/hooks/context/use-issue-modal";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useIssuesActions } from "@/hooks/use-issues-actions";
// services
import { FileService } from "@/services/file.service";
const fileService = new FileService();
// local imports
import { CreateIssueToastActionItems } from "../create-issue-toast-action-items";
import { DraftIssueLayout } from "./draft-issue-layout";
import { IssueFormRoot } from "./form";
import type { IssueFormProps } from "./form";
import type { IssuesModalProps } from "./modal";

export const CreateUpdateIssueModalBase = observer(function CreateUpdateIssueModalBase(props: IssuesModalProps) {
  const {
    data,
    sourceIssueId,
    isOpen,
    onClose,
    beforeFormSubmit,
    withDraftIssueWrapper = true,
    storeType: issueStoreFromProps,
    isDraft = false,
    fetchIssueDetails = true,
    moveToIssue = false,
    modalTitle,
    primaryButtonText,
    isProjectSelectionDisabled = false,
    showActionItemsOnUpdate = false,
  } = props;
  const issueStoreType = useIssueStoreType();

  const storeType = issueStoreFromProps ?? issueStoreType;
  // ref
  const issueTitleRef = useRef<HTMLInputElement>(null);
  const descriptionRequest = useRef(0);
  // states
  const [changesMade, setChangesMade] = useState<IssueFormData | null>(null);
  const [createMore, setCreateMore] = useState(false);
  const [activeProjectId, setActiveProjectId] = useState<string | null>(null);
  const [description, setDescription] = useState<string | undefined>(undefined);
  const [descriptionLoadState, setDescriptionLoadState] = useState<IssueDescriptionLoadState>("loading");
  const [descriptionAttempt, setDescriptionAttempt] = useState(0);
  const [uploadedAssetIds, setUploadedAssetIds] = useState<string[]>([]);
  const [isDuplicateModalOpen, setIsDuplicateModalOpen] = useState(false);
  // store hooks
  const { t } = useTranslation();
  const { workspaceSlug, projectId: routerProjectId, workItem } = useParams();
  const { issues: draftIssues } = useIssues(EIssuesStoreType.WORKSPACE_DRAFT);
  const { fetchIssue } = useIssueDetail();
  const { allowedProjectIds } = useIssueModal();
  const { getProjectByIdentifier } = useProject();
  // current store details
  const { createIssue, updateIssue } = useIssuesActions(storeType);
  // derived values
  const routerProjectIdentifier = workItem?.toString().split("-")[0];
  const projectIdFromRouter = routerProjectIdentifier ? getProjectByIdentifier(routerProjectIdentifier)?.id : undefined;
  const projectId = data?.project_id ?? routerProjectId ?? projectIdFromRouter;

  useEffect(() => {
    const request = ++descriptionRequest.current;
    let active = true;
    const isCurrentRequest = () => active && descriptionRequest.current === request;
    setDescription(undefined);
    setDescriptionLoadState("loading");
    if (!isOpen || !workspaceSlug) return;

    const loadDescription = async () => {
      try {
        let html = data?.description_html ?? "<p></p>";
        const issueId = data?.id ?? sourceIssueId;
        if (issueId !== undefined && fetchIssueDetails) {
          if (!projectId) throw new Error("Missing source project");
          const response = await fetchIssue(workspaceSlug, projectId, issueId);
          if (typeof response?.description_html !== "string") throw new Error("Missing source description");
          html = response.description_html;
        }
        if (isCurrentRequest()) {
          setDescription(html);
          setDescriptionLoadState("ready");
        }
      } catch {
        if (isCurrentRequest()) setDescriptionLoadState("error");
      }
    };
    // This effect owns both success and failure, including requests retired by close/source changes.
    void loadDescription();
    return () => {
      active = false;
    };
  }, [
    isOpen,
    workspaceSlug,
    projectId,
    data?.id,
    data?.description_html,
    sourceIssueId,
    fetchIssueDetails,
    fetchIssue,
    descriptionAttempt,
  ]);

  useEffect(() => {
    // if modal is closed, reset active project to null
    // and return to avoid activeProjectId being set to some other project
    if (!isOpen) {
      setActiveProjectId(null);
      return;
    }

    // if data is present, set active project to the project of the
    // issue. This has more priority than the project in the url.
    if (data && data.project_id) {
      setActiveProjectId(data.project_id);
      return;
    }

    // if data is not present, set active project to the first project in the allowedProjectIds array
    if (allowedProjectIds && allowedProjectIds.length > 0 && !activeProjectId)
      setActiveProjectId(projectId ?? allowedProjectIds?.[0]);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data?.project_id, projectId, isOpen, activeProjectId]);

  const handleCreateMoreToggleChange = (value: boolean) => {
    setCreateMore(value);
  };

  const handleClose = (saveAsDraft?: boolean) => {
    descriptionRequest.current++;
    if (changesMade && saveAsDraft && !data) {
      handleCreateIssue(changesMade, true);
    }

    setActiveProjectId(null);
    setChangesMade(null);
    onClose();
    handleDuplicateIssueModal(false);
  };

  const handleCreateIssue = async (
    payload: IssueFormData,
    is_draft_issue: boolean = false
  ): Promise<Issue | TWorkspaceDraftIssue | undefined> => {
    if (!workspaceSlug || !payload.project_id || descriptionLoadState !== "ready") return;

    try {
      let response: Issue | TWorkspaceDraftIssue | undefined;
      // if draft issue, use draft issue store to create issue
      if (is_draft_issue) {
        response = await draftIssues.createIssue(workspaceSlug, payload);
      } else if (createIssue) {
        response = await createIssue(payload.project_id, payload);
      }

      // update uploaded assets' status
      if (uploadedAssetIds.length > 0) {
        await fileService.updateBulkProjectAssetsUploadStatus(
          workspaceSlug,
          response?.project_id ?? "",
          response?.id ?? "",
          {
            asset_ids: uploadedAssetIds,
          }
        );
        setUploadedAssetIds([]);
      }

      if (!response) throw new Error();

      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("success"),
        message: `${is_draft_issue ? t("draft_created") : t("issue_created_successfully")} `,
        actionItems: !is_draft_issue && response?.project_id && (
          <CreateIssueToastActionItems
            workspaceSlug={workspaceSlug}
            projectId={response?.project_id}
            issueId={response.id}
          />
        ),
      });
      if (!createMore) handleClose();
      if (createMore && issueTitleRef) issueTitleRef?.current?.focus();
      setDescription("<p></p>");
      setChangesMade(null);
      return response;
    } catch (error: any) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("error"),
        message: error?.error ?? t(is_draft_issue ? "draft_creation_failed" : "issue_creation_failed"),
      });
      throw error;
    }
  };

  const handleUpdateIssue = async (payload: IssueFormData): Promise<Issue | undefined> => {
    if (!workspaceSlug || !payload.project_id || !data?.id) return;

    try {
      if (isDraft) await draftIssues.updateIssue(workspaceSlug, data.id, payload);
      else if (updateIssue) await updateIssue(payload.project_id, data.id, payload);

      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("success"),
        message: t("issue_updated_successfully"),
        actionItems:
          showActionItemsOnUpdate && payload.project_id ? (
            <CreateIssueToastActionItems
              workspaceSlug={workspaceSlug}
              projectId={payload.project_id}
              issueId={data.id}
            />
          ) : undefined,
      });
      handleClose();
    } catch (error: any) {
      console.error(error);
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("error"),
        message: error?.error ?? t("issue_could_not_be_updated"),
      });
    }
  };

  const handleFormSubmit = async (payload: IssueFormData, is_draft_issue: boolean = false) => {
    if (!workspaceSlug || !payload.project_id || !storeType || descriptionLoadState !== "ready") return;

    let response: Issue | TWorkspaceDraftIssue | undefined = undefined;

    try {
      if (beforeFormSubmit) await beforeFormSubmit();
      if (!data?.id) response = await handleCreateIssue(payload, is_draft_issue);
      else response = await handleUpdateIssue(payload);
    } finally {
      if (response && props.onSubmit) {
        if (props.isDraft) {
          if (!("sequence_id" in response)) await props.onSubmit(response);
        } else if ("sequence_id" in response) await props.onSubmit(response);
      }
    }
  };

  const handleFormChange = (formData: IssueFormData | null) => setChangesMade(formData);

  const handleUpdateUploadedAssetIds = (assetId: string) => setUploadedAssetIds((prev) => [...prev, assetId]);

  const handleDuplicateIssueModal = (value: boolean) => setIsDuplicateModalOpen(value);

  // don't open the modal outside a workspace or if there are no projects
  if (!workspaceSlug || !allowedProjectIds || allowedProjectIds.length === 0 || !activeProjectId) return null;

  const commonIssueModalProps: IssueFormProps = {
    issueTitleRef: issueTitleRef,
    sourceIssueId,
    descriptionLoadState,
    onRetryDescription: () => setDescriptionAttempt((attempt) => attempt + 1),
    data: {
      ...data,
      description_html: description,
    },
    onAssetUpload: handleUpdateUploadedAssetIds,
    onClose: handleClose,
    onSubmit: (payload) => handleFormSubmit(payload, isDraft),
    workspaceSlug,
    projectId: activeProjectId,
    isCreateMoreToggleEnabled: createMore,
    onCreateMoreToggleChange: handleCreateMoreToggleChange,
    isDraft: isDraft,
    moveToIssue: moveToIssue,
    modalTitle: modalTitle,
    primaryButtonText: primaryButtonText,
    isDuplicateModalOpen: isDuplicateModalOpen,
    handleDuplicateIssueModal: handleDuplicateIssueModal,
    isProjectSelectionDisabled: isProjectSelectionDisabled,
  };

  return (
    <ModalCore
      isOpen={isOpen}
      position={EModalPosition.TOP}
      width={isDuplicateModalOpen ? EModalWidth.VIXL : EModalWidth.XXXXL}
      className="rounded-lg !bg-transparent shadow-none transition-[width] ease-linear"
    >
      {withDraftIssueWrapper ? (
        <DraftIssueLayout {...commonIssueModalProps} changesMade={changesMade} onChange={handleFormChange} />
      ) : (
        <IssueFormRoot {...commonIssueModalProps} />
      )}
    </ModalCore>
  );
});
