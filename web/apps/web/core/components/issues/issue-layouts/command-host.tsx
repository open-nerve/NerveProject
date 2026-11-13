/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { pick } from "lodash-es";
import { observer } from "mobx-react";
import { useLocation, useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import type { EIssuesStoreType } from "@nerve/types";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { ArchiveIssueModal } from "../archive-issue-modal";
import { DeleteIssueModal } from "../delete-issue-modal";
import { useDetailCommand } from "../issue-detail/use-issue-operations";
import { CreateUpdateIssueModal } from "../issue-modal/modal";
import type { IssueFormData } from "../issue-modal/types";

type RequestConfirmation = (issue: Issue, submit: () => Promise<void>) => void;
type RequestCommand = (issue: Issue, submit: () => Promise<void>) => Promise<void>;
/**
 * The page whose list a row's edit and copy dialogs write through: the project's, a cycle's (re-66 L4) or a module's
 * (P6b spec 2.7).
 */
type DialogStore = EIssuesStoreType.PROJECT | EIssuesStoreType.CYCLE | EIssuesStoreType.MODULE;
type RequestDialog = (kind: "edit" | "copy", issue: Issue, storeType: DialogStore) => void;
const CommandsContext = createContext<
  | {
      archive: RequestConfirmation;
      remove: RequestConfirmation;
      restore: RequestCommand;
      removeFromCycle: RequestCommand;
      openDialog: RequestDialog;
    }
  | undefined
>(undefined);
export const useArchiveRequest = () => useContext(CommandsContext)?.archive;
export const useDeleteRequest = () => useContext(CommandsContext)?.remove;
export const useRestoreRequest = () => useContext(CommandsContext)?.restore;
/** The removal of a row's work item from the page's cycle (P5b spec 2.6, R6), whose confirmation takes the row off. */
export const useCycleRemovalRequest = () => useContext(CommandsContext)?.removeFromCycle;
/**
 * A row's edit or copy dialog. The dialog's own read of the work item ends in the work item's check against nerve
 * (its sub-issues verify their parent, M4 P12 D22), during which the list takes the row off the page: the layout holds
 * the dialog so that it stays open with what it was opened with (M6 P5b T13).
 */
export const useWorkItemDialogRequest = () => useContext(CommandsContext)?.openDialog;

/** A copy of issue, as a row's copy dialog starts it: its fields, named as its copy. */
const copyOf = (issue: Issue): IssueFormData =>
  pick({ ...issue, name: `${issue.name} (copy)` }, [
    "name",
    "project_id",
    "state_id",
    "description_html",
    "parent_id",
    "priority",
    "assignee_ids",
    "label_ids",
    "start_date",
    "target_date",
  ]);

/**
 * The layout owns row command feedback and row dialogs even when a confirmation, or a check of the work item, removes
 * the initiating row.
 */
export const IssueListCommandHost = observer(function IssueListCommandHost({ children }: { children: ReactNode }) {
  const { rootIssueStore: root } = useIssueDetail();
  const { workspaceSlug, projectId } = useParams();
  const location = useLocation();
  const runCommand = useDetailCommand(workspaceSlug ?? "", projectId, location.key);
  const { t } = useTranslation();
  const owner = useMemo(() => ({ root, location: location.key, pending: new Set<string>() }), [root, location.key]);
  const [request, setRequest] = useState<{
    owner: typeof owner;
    issue: Issue;
    submit: () => Promise<void>;
    permission: string;
    kind: "archive" | "delete";
  }>();
  const [dialog, setDialog] = useState<{
    owner: typeof owner;
    data: IssueFormData;
    sourceIssueId: string | undefined;
    storeType: DialogStore;
    permission: string;
  }>();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug ?? "");
  const permission = workspace && root.permissionReads.identity(workspace.id, projectId ? [projectId] : [], true);
  const requestConfirmation = useCallback(
    (kind: "archive" | "delete", issue: Issue, submit: () => Promise<void>) => {
      if (!root.active || !workspace || issue.workspace_id !== workspace.id) return;
      setRequest({
        owner,
        issue,
        submit,
        kind,
        permission: root.permissionReads.identity(workspace.id, [issue.project_id], true),
      });
    },
    [root, workspace, owner]
  );
  const requestDialog = useCallback<RequestDialog>(
    (kind, issue, storeType) => {
      if (!root.active || !workspace || issue.workspace_id !== workspace.id) return;
      setDialog({
        owner,
        data: kind === "edit" ? issue : copyOf(issue),
        sourceIssueId: kind === "copy" ? issue.id : undefined,
        storeType,
        permission: root.permissionReads.identity(workspace.id, [issue.project_id], true),
      });
    },
    [root, workspace, owner]
  );
  /**
   * Runs a row's command whose confirmation may take the row off the list, one at a time for each command and work
   * item, saying why nerve refused it; true once nerve confirmed it while the page is still the one that sent it.
   */
  const runRowCommand = useCallback(
    async (kind: "restore" | "remove-from-cycle", issue: Issue, submit: () => Promise<void>) => {
      if (!workspace || issue.workspace_id !== workspace.id || issue.project_id !== projectId)
        throw new SessionChangedError();
      const key = JSON.stringify([kind, permission, issue.id]);
      if (owner.pending.has(key)) return false;
      owner.pending.add(key);
      try {
        await runCommand(submit);
        return true;
      } finally {
        owner.pending.delete(key);
      }
    },
    [workspace, projectId, runCommand, owner, permission]
  );
  const requestRestore = useCallback<RequestCommand>(
    async (issue, submit) => {
      if (await runRowCommand("restore", issue, submit))
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("issue.restore.success.title"),
          message: t("issue.restore.success.message"),
        });
    },
    [runRowCommand, t]
  );
  const requestCycleRemoval = useCallback<RequestCommand>(
    async (issue, submit) => {
      if (await runRowCommand("remove-from-cycle", issue, submit))
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("common.success"), message: t("issue.remove.cycle.success") });
    },
    [runRowCommand, t]
  );
  // a confirmation or a dialog of another route, or of a permission since changed, has ended
  const holds = (held: { owner: typeof owner; permission: string }, projectID: string | null | undefined) =>
    held.owner === owner &&
    root.active &&
    !!workspace &&
    !!projectID &&
    held.permission === root.permissionReads.identity(workspace.id, [projectID], true);
  const current = request && holds(request, request.issue.project_id);
  const commands = useMemo(
    () => ({
      archive: (issue: Issue, submit: () => Promise<void>) => requestConfirmation("archive", issue, submit),
      remove: (issue: Issue, submit: () => Promise<void>) => requestConfirmation("delete", issue, submit),
      restore: requestRestore,
      removeFromCycle: requestCycleRemoval,
      openDialog: requestDialog,
    }),
    [requestConfirmation, requestRestore, requestCycleRemoval, requestDialog]
  );
  return (
    <CommandsContext.Provider value={commands}>
      {children}
      {current && request.kind === "archive" && (
        <ArchiveIssueModal
          data={request.issue}
          isOpen
          handleClose={() => setRequest(undefined)}
          onSubmit={request.submit}
        />
      )}
      {current && request.kind === "delete" && (
        <DeleteIssueModal
          data={request.issue}
          isOpen
          handleClose={() => setRequest(undefined)}
          onSubmit={request.submit}
        />
      )}
      {dialog && holds(dialog, dialog.data.project_id) && (
        <CreateUpdateIssueModal
          isOpen
          onClose={() => setDialog(undefined)}
          data={dialog.data}
          sourceIssueId={dialog.sourceIssueId}
          storeType={dialog.storeType}
        />
      )}
    </CommandsContext.Provider>
  );
});
