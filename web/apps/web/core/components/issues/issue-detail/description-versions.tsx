/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { RefObject } from "react";
import { observer } from "mobx-react";
import type { EditorRefApi } from "@nerve/editor";
import { DescriptionVersionsRoot } from "@/components/core/description-versions";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useMember } from "@/hooks/store/use-member";
import { useSessionSWR } from "@/lib/use-session-swr";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useDetailScope } from "@/components/issues/issue-detail-widgets/use-detail-scope";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  isRestoreDisabled: boolean;
  editorRef: RefObject<Pick<EditorRefApi, "setEditorValue"> | null>;
};
export const IssueDescriptionVersions = observer(function IssueDescriptionVersions(props: Props) {
  const { workspaceSlug: ws, projectId: project, issueId: id, editorRef, isRestoreDisabled } = props;
  const { rootIssueStore: root, versions, issue } = useIssueDetail();
  const { getUserDetails } = useMember();
  const scope = useDetailScope(ws, project, id);
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(ws);
  const item = issue.getIssueDetailById(id);
  const canRead = !!item && versions.canRead(ws, project);
  const identity = root.permissionReads.identity(workspace?.id, [project], true);
  const { error, mutate } = useSessionSWR(
    canRead ? ["ISSUE_DESCRIPTION_VERSIONS", identity, ws, project, id] : null,
    (_identity, slug, projectID, issueID) => versions.fetchVersions(slug, projectID, issueID),
    { dedupingInterval: 0 }
  );
  if (!canRead || !item) return null;
  return (
    <DescriptionVersionsRoot
      className="flex-shrink-0"
      scopeKey={identity}
      workspaceSlug={ws}
      projectId={project}
      entityInformation={{
        id,
        createdAt: new Date(item.created_at),
        createdByDisplayName: getUserDetails(item.created_by_id ?? "")?.display_name ?? "",
        isRestoreDisabled,
      }}
      versions={versions
        .getVersions(id)
        ?.map((row) => ({ id: row.id, ownerId: row.owned_by_id, lastSavedAt: row.last_saved_at }))}
      listError={versions.getError(id) ?? error}
      retryList={() => mutate()}
      loadMore={versions.hasMore(id) ? () => versions.fetchVersions(ws, project, id, true) : undefined}
      readDescription={async (version) => (await versions.fetchVersion(ws, project, id, version)).description_html}
      handleRestore={async (html) => {
        const current = scope();
        if (!current()) throw new SessionChangedError();
        const confirmed = await versions.restore(ws, project, id, html);
        if (!current()) throw new SessionChangedError();
        editorRef.current?.setEditorValue(confirmed, false);
      }}
    />
  );
});
