/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
import { useSessionSWR } from "@/lib/use-session-swr";
import { DetailResourceError } from "../issue-detail-widgets/resource-error";
import { DraftIssueBlock } from "./draft-issue-block";
import { WorkspaceDraftEmptyState } from "./empty-state";
import { WorkspaceDraftIssuesLoader } from "./loader";

export const WorkspaceDraftIssuesRoot = observer(function WorkspaceDraftIssuesRoot({
  workspaceSlug: ws,
}: {
  workspaceSlug: string;
}) {
  const { t } = useTranslation();
  const drafts = useWorkspaceDraftIssues();
  const { rootIssueStore: root } = useIssueDetail();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(ws);
  const identity = root.permissionReads.identity(workspace?.id, [], true);
  const { error, isLoading, mutate } = useSessionSWR(
    drafts.canRead(ws) ? ["WORKSPACE_DRAFTS", identity, ws] : null,
    (_identity, slug) => drafts.fetchIssues(slug),
    { dedupingInterval: 0 }
  );
  if (!drafts.canRead(ws)) return null;
  const rows = drafts.getDrafts(ws);
  const failure = drafts.getError(ws) ?? error;
  if (!rows && isLoading) return <WorkspaceDraftIssuesLoader items={14} />;
  return (
    <div className="relative" key={identity}>
      <DetailResourceError
        error={failure}
        retry={async () => {
          await drafts.retry(ws);
          await mutate(drafts.getDrafts(ws), { revalidate: false });
        }}
        workspaceSlug={ws}
        projectId={undefined}
        issueId="drafts"
      />
      {rows?.map((row) => (
        <DraftIssueBlock key={row.id} workspaceSlug={ws} issueId={row.id} />
      ))}
      {rows?.length === 0 && !failure && <WorkspaceDraftEmptyState />}
      {drafts.hasMore(ws) && (
        <button
          type="button"
          disabled={!!drafts.getLoader(ws)}
          className="h-11 w-full border-b border-subtle bg-surface-1 p-3 text-left text-accent-primary"
          onClick={() => void drafts.fetchIssues(ws, true).catch(() => undefined)}
        >
          {t("common.load_more")} &darr;
        </button>
      )}
    </div>
  );
});
