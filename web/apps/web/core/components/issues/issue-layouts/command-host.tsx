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
type RequestRestore = (issue: Issue, submit: () => Promise<void>) => Promise<void>;
const CommandsContext = createContext<
  | {
      archive: RequestConfirmation;
      remove: RequestConfirmation;
      restore: RequestRestore;
    }
  | undefined
>(undefined);
export const useArchiveRequest = () => useContext(CommandsContext)?.archive;
export const useDeleteRequest = () => useContext(CommandsContext)?.remove;
export const useRestoreRequest = () => useContext(CommandsContext)?.restore;

/** The layout owns row command feedback even when a confirmation removes the initiating row. */
export const IssueListCommandHost = observer(function IssueListCommandHost({ children }: { children: ReactNode }) {
  const { rootIssueStore: root } = useIssueDetail();
  const { workspaceSlug, projectId } = useParams();
  const location = useLocation();
  const runRestore = useDetailCommand(workspaceSlug ?? "", projectId, location.key);
  const { t } = useTranslation();
  const owner = useMemo(
    () => ({ root, location: location.key, pendingRestores: new Set<string>() }),
    [root, location.key]
  );
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
  const requestRestore = useCallback<RequestRestore>(
    async (issue, submit) => {
      if (!workspace || issue.workspace_id !== workspace.id || issue.project_id !== projectId)
        throw new SessionChangedError();
      const restoreKey = JSON.stringify([permission, issue.id]);
      if (owner.pendingRestores.has(restoreKey)) return;
      owner.pendingRestores.add(restoreKey);
      try {
        await runRestore(submit);
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("issue.restore.success.title"),
          message: t("issue.restore.success.message"),
        });
      } finally {
        owner.pendingRestores.delete(restoreKey);
      }
    },
    [workspace, projectId, runRestore, owner, permission, t]
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
    }),
    [requestConfirmation, requestRestore]
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
