/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueFormData, IssueFormSubmitOptions } from "./types";
import { useContext, useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// Nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sessionGuard } from "@/lib/in-session";
import { draftFormData } from "./draft-form";
import { TransferPendingError, useDraftContent } from "./draft-files";
import { ContentCopy, CopyNotice } from "./content-copy";
import { StoreContext } from "@/lib/store-context";
import { joinUrlPath } from "@nerve/utils";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";
import { EIssuesStoreType } from "@nerve/types";
import type { EditorRefApi } from "@nerve/editor";
// hooks
import { useIssueModal } from "@/hooks/context/use-issue-modal";
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
import { useProject } from "@/hooks/store/use-project";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useIssuesActions } from "@/hooks/use-issues-actions";
// services
// local imports
import { DraftIssueLayout } from "./draft-issue-layout";
import { useDescriptionRead } from "./description-read";
import { useCreateCommand, useCreateStart } from "./create-command";
import { useUpdateCommand } from "./update-command";
import { usePublishCommand } from "./publish-command";
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
  const editorRef = useRef<EditorRefApi>(null);
  const submission = useRef<Promise<void> | undefined>(undefined);
  // states
  const [changesMade, setChangesMade] = useState<IssueFormData | null>(null);
  const [createMore, setCreateMore] = useState(false);
  const [activeProjectId, setActiveProjectId] = useState<string | null>(null);
  const [isPending, setIsPending] = useState(false);
  // store hooks
  const { t } = useTranslation();
  const refuse = useRefusalToast();
  const { workspaceSlug, projectId: routerProjectId, workItem } = useParams();
  const draftIssues = useWorkspaceDraftIssues();
  const root = useContext(StoreContext);
  // Only a copy can carry another host's images; it then goes through one real target draft.
  const [copy] = useState(() =>
    workspaceSlug && (sourceIssueId !== undefined || (props.isDraft && props.draftSeed !== undefined))
      ? new ContentCopy(root, workspaceSlug)
      : undefined
  );
  const { allowedProjectIds } = useIssueModal();
  const { getProjectByIdentifier } = useProject();
  const content = useDraftContent(workspaceSlug ?? "", isDraft ? data?.id : undefined);
  // current store details
  const { createIssueAsGiven, updateIssue } = useIssuesActions(
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

  const { description, setDescription, descriptionLoadState, retryDescription, retireDescription } = useDescriptionRead(
    { isOpen, workspaceSlug, projectId, data, sourceIssueId, fetchIssueDetails, isDraft }
  );

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

  /** Whether saving the stored draft changes its project; its content lifetime must end before that request. */
  const movesProject = (payload: IssueFormData) => {
    const held = isDraft && data?.id ? draftIssues.getIssueById(data.id, workspaceSlug) : undefined;
    return !!held && Object.hasOwn(payload, "project_id") && (payload.project_id || null) !== held.project_id;
  };
  /** A move or publish waits until the user finishes or explicitly cancels the transfers on the wire. */
  const assertSettled = () => {
    if (content.settled()) return;
    setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("content_files.draft.transfer_busy") });
    throw new TransferPendingError();
  };

  const handleCreateMoreToggleChange = (value: boolean) => {
    setCreateMore(value);
  };

  const finishClose = () => {
    retireDescription();
    // Closing says where the copy stands: a publish whose answer was lost may already be a work item.
    if (copy?.target && workspaceSlug)
      setToast(
        copy.published
          ? { type: TOAST_TYPE.INFO, title: t("info"), message: t("content_files.copy.published") }
          : {
              type: TOAST_TYPE.INFO,
              title: t("info"),
              message:
                copy.unknown === "publish" && !copy.checked
                  ? t("content_files.copy.unknown_publish")
                  : t("content_files.copy.kept_closed"),
              actionItems: <a href={joinUrlPath(workspaceSlug, "drafts")}>{t("content_files.copy.open_drafts")}</a>,
            }
      );

    setActiveProjectId(null);
    setChangesMade(null);
    onClose();
  };

  const start = useCreateStart({ workspaceSlug, storeType, isDraft, projectId: activeProjectId, data });
  const handleCreateIssue = useCreateCommand({
    workspaceSlug,
    draftSeed: props.isDraft ? props.draftSeed : undefined,
    copy,
    formScope,
    createIssue: createIssueAsGiven,
    descriptionLoadState,
  });
  const handleUpdateIssue = useUpdateCommand({
    workspaceSlug,
    isDraft,
    data,
    formScope,
    movesProject,
    content,
    updateIssue,
    showActionItemsOnUpdate,
  });
  const { executePublish, publishBlocked } = usePublishCommand({
    workspaceSlug,
    isDraft,
    data,
    descriptionLoadState,
    formScope,
    assertSettled,
    content,
    finishClose,
  });

  const executeFormSubmit = async (
    payload: IssueFormData,
    is_draft_issue: boolean,
    options?: IssueFormSubmitOptions
  ) => {
    if (!workspaceSlug || (!is_draft_issue && !payload.project_id) || !storeType || descriptionLoadState !== "ready")
      return;

    const session = sessionGuard();
    const current = formScope();
    const inSession = () => session() && current();
    if (data?.id && movesProject(payload)) assertSettled();
    if (beforeFormSubmit) await beforeFormSubmit();
    if (!inSession()) throw new SessionChangedError();
    const response = !data?.id
      ? await handleCreateIssue(payload, is_draft_issue, options)
      : await handleUpdateIssue(payload, options);
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
    if (data?.id || !createMore || options?.closeAfterSave) {
      // This is the confirmed command's close, not a user dismissal of a pending form.
      finishClose();
    } else issueTitleRef.current?.focus();
    if (!data?.id) {
      setDescription("<p></p>");
      setChangesMade(null);
    }
  };
  const withPending = (command: () => Promise<void>) => {
    if (submission.current) return submission.current;
    const pending = Promise.resolve()
      .then(command)
      .finally(() => {
        if (submission.current === pending) {
          submission.current = undefined;
          setIsPending(false);
        }
      });
    submission.current = pending;
    setIsPending(true);
    return pending;
  };
  const handleFormSubmit = (payload: IssueFormData, saveDraft = false, options?: IssueFormSubmitOptions) =>
    withPending(() => executeFormSubmit(payload, saveDraft, options));
  const handlePublish = (payload: IssueFormData) => withPending(() => executePublish(payload));

  const handleFormChange = (formData: IssueFormData | null) => setChangesMade(formData);

  // don't open the modal outside a workspace or if there are no projects
  if (
    !workspaceSlug ||
    (isDraft &&
      (!draftIssues.canRead(workspaceSlug) || (data?.id && !draftIssues.getIssueById(data.id, workspaceSlug)))) ||
    (!isDraft && (!allowedProjectIds?.length || !activeProjectId))
  )
    return null;

  const commonIssueModalProps: IssueFormProps = {
    editorRef,
    issueTitleRef: issueTitleRef,
    sourceIssueId,
    hasDraftSeed: props.isDraft && props.draftSeed !== undefined,
    canSaveDraft: draftIssues.canRead(workspaceSlug),
    descriptionLoadState,
    onRetryDescription: retryDescription,
    data: {
      ...start.data,
      description_html: description,
    },
    onClose: finishClose,
    onSubmit: (payload, saveDraft = isDraft, options) => handleFormSubmit(payload, saveDraft, options),
    isPending,
    onPublish: handlePublish,
    workspaceSlug,
    projectId: activeProjectId,
    isCreateMoreToggleEnabled: createMore,
    onCreateMoreToggleChange: handleCreateMoreToggleChange,
    isDraft: isDraft,
    moveToIssue: moveToIssue,
    modalTitle: modalTitle,
    primaryButtonText: primaryButtonText,
    isProjectSelectionDisabled: isProjectSelectionDisabled,
    content,
    publishBlocked,
    notice: copy && workspaceSlug ? <CopyNotice copy={copy} slug={workspaceSlug} /> : undefined,
    planningLoading: start.planningLoading,
  };

  return (
    <DraftIssueLayout
      {...commonIssueModalProps}
      isOpen={isOpen}
      promptToSaveDraft={withDraftIssueWrapper && draftIssues.canRead(workspaceSlug)}
      isBusy={() => !!submission.current}
      changesMade={changesMade}
      onChange={handleFormChange}
    />
  );
});
