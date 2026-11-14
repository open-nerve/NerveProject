/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { pick } from "lodash-es";
import { observer } from "mobx-react";
import { useLocation, useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { EIssuesStoreType } from "@nerve/types";
import { isPlanningPageStoreType } from "@/hooks/planning-page-domains";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { ArchiveIssueModal } from "../archive-issue-modal";
import { DeleteIssueModal } from "../delete-issue-modal";
import { useDetailCommand } from "../issue-detail/use-issue-operations";
import { CreateUpdateIssueModal } from "../issue-modal/modal";
import type { IssueFormData } from "../issue-modal/types";

type RequestConfirmation = (issue: Issue, submit: () => Promise<void>) => void;
type RequestCommand = (issue: Issue, submit: () => Promise<void>) => Promise<void>;
/** A removal from the page's cycle or module, with what the page says once nerve confirms it (an i18n key). */
type RequestRemoval = (issue: Issue, submit: () => Promise<void>, success: string) => Promise<void>;
/**
 * The pages whose list a dialog the host holds writes through: a row's edit and copy, the project's, a cycle's (re-66
 * L4) or a module's (P6b spec 2.7); a group's new work item, each page whose groups offer one (its store's
 * viewFlags.enableIssueCreation): those and a person's page (P7a spec 2.7, re-72 F4).
 */
const DIALOG_STORES = [
  EIssuesStoreType.PROJECT,
  EIssuesStoreType.CYCLE,
  EIssuesStoreType.MODULE,
  EIssuesStoreType.PROFILE,
] as const;
type DialogStore = (typeof DIALOG_STORES)[number];
/** Whether the page of the store type is one whose list's dialogs the host holds. */
export const isDialogStore = (type: EIssuesStoreType): type is DialogStore =>
  (DIALOG_STORES as readonly EIssuesStoreType[]).includes(type);
type RequestDialog = (kind: "edit" | "copy", issue: Issue, storeType: DialogStore) => void;
/** A group's new work item, with what the group gives it (its payload). */
type RequestCreate = (data: IssueFormData, storeType: DialogStore) => void;
const CommandsContext = createContext<
  | {
      archive: RequestConfirmation;
      remove: RequestConfirmation;
      restore: RequestCommand;
      removeFromView: RequestRemoval;
      openDialog: RequestDialog;
      create: RequestCreate;
    }
  | undefined
>(undefined);
export const useArchiveRequest = () => useContext(CommandsContext)?.archive;
export const useDeleteRequest = () => useContext(CommandsContext)?.remove;
export const useRestoreRequest = () => useContext(CommandsContext)?.restore;
/**
 * The removal of a row's work item from the page's cycle or module (P5b spec 2.6, R6; P6b spec 2.7), whose
 * confirmation takes the row off; the page's domain names what the page then says.
 */
export const useRemovalFromViewRequest = () => useContext(CommandsContext)?.removeFromView;
/**
 * A row's edit or copy dialog. The dialog's own read of the work item ends in the work item's check against nerve
 * (its sub-issues verify their parent, M4 P12 D22), during which the list takes the row off the page: the layout holds
 * the dialog so that it stays open with what it was opened with (M6 P5b T13).
 */
export const useWorkItemDialogRequest = () => useContext(CommandsContext)?.openDialog;
/**
 * A group's new work item's dialog (P7a spec 2.7, R16): while the list is checked the layout shows its loader, its
 * groups and their headers gone, so the layout holds the dialog, as a row's.
 */
export const useCreateDialogRequest = () => useContext(CommandsContext)?.create;

/**
 * A copy of issue, as a row's copy dialog starts it: its fields, named as its copy, its cycle and modules with them,
 * of which its form keeps those it may join (P7a spec 2.7, R15).
 */
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
    "cycle_id",
    "module_ids",
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
    /** The projects of the permission that holds it: its work item's; or a new one's (requestCreate). */
    projects: string[];
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
        projects: [issue.project_id],
        permission: root.permissionReads.identity(workspace.id, [issue.project_id], true),
      });
    },
    [root, workspace, owner]
  );
  const requestCreate = useCallback<RequestCreate>(
    (data, storeType) => {
      if (!root.active || !workspace) return;
      // the group's project when grouped by project, else the route's; on a page of no project (a person's), none:
      // the caller's workspace and its projects hold it
      const project = data.project_id ?? projectId;
      const projects = project ? [project] : [];
      setDialog({
        owner,
        data,
        sourceIssueId: undefined,
        storeType,
        projects,
        permission: root.permissionReads.identity(workspace.id, projects, true),
      });
    },
    [root, workspace, owner, projectId]
  );
  /**
   * Runs a row's command whose confirmation may take the row off the list, one at a time for each command and work
   * item, saying why nerve refused it; true once nerve confirmed it while the page is still the one that sent it.
   */
  const runRowCommand = useCallback(
    async (kind: "restore" | "remove-from-view", issue: Issue, submit: () => Promise<void>) => {
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
  const requestRemovalFromView = useCallback<RequestRemoval>(
    async (issue, submit, success) => {
      if (await runRowCommand("remove-from-view", issue, submit))
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("common.success"), message: t(success) });
    },
    [runRowCommand, t]
  );
  // a confirmation or a dialog of another route, or of a permission since changed, has ended
  const holds = (held: { owner: typeof owner; permission: string }, projects: readonly string[]) =>
    held.owner === owner &&
    root.active &&
    !!workspace &&
    held.permission === root.permissionReads.identity(workspace.id, projects, true);
  const current = request && holds(request, [request.issue.project_id]);
  const commands = useMemo(
    () => ({
      archive: (issue: Issue, submit: () => Promise<void>) => requestConfirmation("archive", issue, submit),
      remove: (issue: Issue, submit: () => Promise<void>) => requestConfirmation("delete", issue, submit),
      restore: requestRestore,
      removeFromView: requestRemovalFromView,
      openDialog: requestDialog,
      create: requestCreate,
    }),
    [requestConfirmation, requestRestore, requestRemovalFromView, requestDialog, requestCreate]
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
      {dialog && holds(dialog, dialog.projects) && (
        <CreateUpdateIssueModal
          isOpen
          onClose={() => setDialog(undefined)}
          data={dialog.data}
          sourceIssueId={dialog.sourceIssueId}
          storeType={dialog.storeType}
          // a new work item of a cycle's or a module's page joins the page's cycle or module, so it stays in its
          // project (P5b spec 2.5, P6b spec 2.6); an edit's and a copy's form keep theirs anyway
          isProjectSelectionDisabled={isPlanningPageStoreType(dialog.storeType)}
        />
      )}
    </CommandsContext.Provider>
  );
});
