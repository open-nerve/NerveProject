/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { observer } from "mobx-react";
import { useLocation, useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { setToast, TOAST_TYPE } from "@nerve/propel/toast";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { ArchiveIssueModal } from "../archive-issue-modal";
import { DeleteIssueModal } from "../delete-issue-modal";
import { useDetailCommand } from "../issue-detail/use-issue-operations";

type RequestConfirmation = (issue: Issue, submit: () => Promise<void>) => void;
type RequestCommand = (issue: Issue, submit: () => Promise<void>) => Promise<void>;
const CommandsContext = createContext<
  | {
      archive: RequestConfirmation;
      remove: RequestConfirmation;
      restore: RequestCommand;
      removeFromCycle: RequestCommand;
    }
  | undefined
>(undefined);
export const useArchiveRequest = () => useContext(CommandsContext)?.archive;
export const useDeleteRequest = () => useContext(CommandsContext)?.remove;
export const useRestoreRequest = () => useContext(CommandsContext)?.restore;
/** The removal of a row's work item from the page's cycle (P5b spec 2.6, R6), whose confirmation takes the row off. */
export const useCycleRemovalRequest = () => useContext(CommandsContext)?.removeFromCycle;

/** The layout owns row command feedback even when a confirmation removes the initiating row. */
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
  const current =
    request &&
    request.owner === owner &&
    root.active &&
    workspace &&
    request.permission === root.permissionReads.identity(workspace.id, [request.issue.project_id], true);
  const commands = useMemo(
    () => ({
      archive: (issue: Issue, submit: () => Promise<void>) => requestConfirmation("archive", issue, submit),
      remove: (issue: Issue, submit: () => Promise<void>) => requestConfirmation("delete", issue, submit),
      restore: requestRestore,
      removeFromCycle: requestCycleRemoval,
    }),
    [requestConfirmation, requestRestore, requestCycleRemoval]
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
    </CommandsContext.Provider>
  );
});
