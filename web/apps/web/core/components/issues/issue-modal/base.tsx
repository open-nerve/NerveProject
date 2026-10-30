/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, Draft } from "@nerve/api-client";

import type { IssueFormData, IssueDescriptionLoadState } from "./types";
import { useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// Nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sessionGuard } from "@/lib/in-session";
import { draftInput } from "@/services/issue/draft-input";
import { draftFormData } from "./draft-form";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { EIssuesStoreType } from "@nerve/types";
import { EModalPosition, EModalWidth, ModalCore } from "@nerve/ui";
// hooks
import { useIssueModal } from "@/hooks/context/use-issue-modal";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
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
  const data = props.isDraft && props.draft ? draftFormData(props.draft) : props.data;
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
  const refuse = useRefusalToast();
  const { workspaceSlug, projectId: routerProjectId, workItem } = useParams();
  const draftIssues = useWorkspaceDraftIssues();
  const { fetchIssue } = useIssueDetail();
  const { allowedProjectIds } = useIssueModal();
  const { getProjectByIdentifier } = useProject();
  // current store details
  const { createIssue, updateIssue } = useIssuesActions(
    storeType === EIssuesStoreType.WORKSPACE_DRAFT ? EIssuesStoreType.DEFAULT : storeType
  );
  // derived values
  const routerProjectIdentifier = workItem?.toString().split("-")[0];
  const projectIdFromRouter = routerProjectIdentifier ? getProjectByIdentifier(routerProjectIdentifier)?.id : undefined;
  const projectId = isDraft ? (data?.project_id ?? null) : (data?.project_id ?? routerProjectId ?? projectIdFromRouter);
  const formScope = useDetailScope(
    workspaceSlug ?? "",
    isDraft ? undefined : (activeProjectId ?? projectId ?? undefined),
    JSON.stringify([isOpen, isDraft, data?.id, sourceIssueId]),
    true
  );

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
        if (issueId !== undefined && fetchIssueDetails && !isDraft) {
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
    isDraft,
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

    if (isDraft) {
      setActiveProjectId(data?.project_id ?? null);
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

  const handleClose = () => {
    descriptionRequest.current++;

    setActiveProjectId(null);
    setChangesMade(null);
    onClose();
    handleDuplicateIssueModal(false);
  };

  const handleCreateIssue = async (
    payload: IssueFormData,
    is_draft_issue: boolean = false
  ): Promise<Issue | Draft | undefined> => {
    if (!workspaceSlug || (!is_draft_issue && !payload.project_id) || descriptionLoadState !== "ready") return;

    const session = sessionGuard();
    const current = formScope();
    const inSession = () => session() && current();
    try {
      let response: Issue | Draft | undefined;
      // if draft issue, use draft issue store to create issue
      if (is_draft_issue) {
        response = await draftIssues.createIssue(
          workspaceSlug,
          draftInput(payload, props.isDraft ? props.draftSeed : undefined)
        );
      } else if (createIssue) {
        if (!payload.name?.trim()) return;
        response = await createIssue(payload.project_id!, {
          name: payload.name,
          description_html: payload.description_html,
          state_id: payload.state_id,
          priority: payload.priority,
          assignee_ids: payload.assignee_ids,
          label_ids: payload.label_ids,
          ...(payload.start_date != null ? { start_date: payload.start_date } : {}),
          ...(payload.target_date != null ? { target_date: payload.target_date } : {}),
          ...(payload.parent_id != null ? { parent_id: payload.parent_id } : {}),
        });
      }

      if (!inSession()) throw new SessionChangedError();
      if (!response) throw new Error("Missing confirmed work item");
      // M5 attachment follow-up cannot turn a confirmed creation into a repeatable creation failure.
      if (uploadedAssetIds.length > 0 && response.project_id) {
        try {
          await fileService.updateBulkProjectAssetsUploadStatus(workspaceSlug, response.project_id, response.id, {
            asset_ids: uploadedAssetIds,
          });
        } catch (error) {
          if (inSession()) refuse(error);
        }
        if (!inSession()) throw new SessionChangedError();
        setUploadedAssetIds([]);
      }

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
      return response;
    } catch (error) {
      if (inSession()) refuse(error);
      throw error;
    }
  };

  const handleUpdateIssue = async (payload: IssueFormData): Promise<Draft | undefined> => {
    if (!workspaceSlug || (!isDraft && !payload.project_id) || !data?.id) return;
    const session = sessionGuard();
    const current = formScope();
    const inSession = () => session() && current();
    try {
      let response: Draft | undefined;
      if (isDraft)
        response = await draftIssues.updateIssue(workspaceSlug, data.id, (held) => draftInput(payload, held.payload));
      else if (updateIssue) await updateIssue(payload.project_id!, data.id, payload);
      if (!inSession()) throw new SessionChangedError();
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
      return response;
    } catch (error) {
      if (inSession()) refuse(error);
      throw error;
    }
  };
  const handlePublish = async (payload: IssueFormData) => {
    if (!workspaceSlug || !isDraft || !data?.id || descriptionLoadState !== "ready") return;
    const current = formScope();
    const inSession = sessionGuard();
    try {
      await draftIssues.moveIssue(workspaceSlug, data.id, (held) => draftInput(payload, held.payload));
      if (!inSession() || !current()) return;
      setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: t("issue_created_successfully") });
      handleClose();
    } catch (error) {
      if (inSession() && current()) refuse(error);
      throw error;
    }
  };

  const handleFormSubmit = async (payload: IssueFormData, is_draft_issue: boolean = false) => {
    if (!workspaceSlug || (!is_draft_issue && !payload.project_id) || !storeType || descriptionLoadState !== "ready")
      return;

    const session = sessionGuard();
    const current = formScope();
    const inSession = () => session() && current();
    if (beforeFormSubmit) await beforeFormSubmit();
    if (!inSession()) throw new SessionChangedError();
    const response = !data?.id ? await handleCreateIssue(payload, is_draft_issue) : await handleUpdateIssue(payload);
    if (!inSession()) throw new SessionChangedError();
    // Deliver the confirmed result before our own close retires this form's lifetime.
    if (response && props.onSubmit) {
      try {
        if (props.isDraft) {
          if (!("sequence_id" in response)) await props.onSubmit(response);
        } else if ("sequence_id" in response) await props.onSubmit(response);
      } catch (error) {
        // A consumer's follow-up is not a failure of the confirmed creation; never invite another POST.
        if (inSession()) refuse(error);
      }
    }
    if (!inSession()) return;
    if (data?.id || !createMore) handleClose();
    else issueTitleRef.current?.focus();
    if (!data?.id) {
      setDescription("<p></p>");
      setChangesMade(null);
    }
  };

  const handleFormChange = (formData: IssueFormData | null) => setChangesMade(formData);

  const handleUpdateUploadedAssetIds = (assetId: string) => setUploadedAssetIds((prev) => [...prev, assetId]);

  const handleDuplicateIssueModal = (value: boolean) => setIsDuplicateModalOpen(value);

  // don't open the modal outside a workspace or if there are no projects
  if (
    !workspaceSlug ||
    (isDraft &&
      (!draftIssues.canRead(workspaceSlug) || (data?.id && !draftIssues.getIssueById(data.id, workspaceSlug)))) ||
    (!isDraft && (!allowedProjectIds?.length || !activeProjectId))
  )
    return null;

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
    onSubmit: (payload, saveDraft = isDraft) => handleFormSubmit(payload, saveDraft),
    onPublish: handlePublish,
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
