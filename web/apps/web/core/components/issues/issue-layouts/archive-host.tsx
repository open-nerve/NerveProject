/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { observer } from "mobx-react";
import { useLocation, useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { ArchiveIssueModal } from "../archive-issue-modal";

type RequestArchive = (issue: Issue, submit: () => Promise<void>) => void;
const ArchiveContext = createContext<RequestArchive | undefined>(undefined);
export const useArchiveRequest = () => useContext(ArchiveContext);

/** The page owns the confirmation: a successful archive can remove the initiating row immediately. */
export const IssueArchiveHost = observer(function IssueArchiveHost({ children }: { children: ReactNode }) {
  const { rootIssueStore: root } = useIssueDetail();
  const { workspaceSlug } = useParams();
  const location = useLocation();
  const owner = useMemo(() => ({ root, location: location.key }), [root, location.key]);
  const [request, setRequest] = useState<{
    owner: typeof owner;
    issue: Issue;
    submit: () => Promise<void>;
    permission: string;
  }>();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug ?? "");
  const requestArchive = useCallback<RequestArchive>(
    (issue, submit) => {
      if (!root.active || !workspace || issue.workspace_id !== workspace.id) return;
      setRequest({
        owner,
        issue,
        submit,
        permission: root.permissionReads.identity(workspace.id, [issue.project_id], true),
      });
    },
    [root, workspace, owner]
  );
  const current =
    request &&
    request.owner === owner &&
    root.active &&
    workspace &&
    request.permission === root.permissionReads.identity(workspace.id, [request.issue.project_id], true);
  return (
    <ArchiveContext.Provider value={requestArchive}>
      {children}
      {current && (
        <ArchiveIssueModal
          data={request.issue}
          isOpen
          handleClose={() => setRequest(undefined)}
          onSubmit={request.submit}
        />
      )}
    </ArchiveContext.Provider>
  );
});
