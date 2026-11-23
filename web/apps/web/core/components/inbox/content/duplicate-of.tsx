/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { DuplicateOfOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { ControlLink } from "@nerve/ui";
import { generateWorkItemLink } from "@nerve/utils";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

type Props = {
  workspaceSlug: string;
  projectId: string;
  /** The work item the selected one duplicates, as its intake names it: none, or one gone or hidden from the caller. */
  duplicateToId: string | null;
};

/**
 * The work item a work item of a project's intake duplicates (P5a spec 2.5, 2.9, R17): read once, keyed by the
 * caller's permissions, by the read a work item detail makes of its parent (fetchIssueCore), never by
 * refreshAuthoritative, a verification that would mark the cells of the project's held lists stale and, failing, give
 * those lists its error (re-94 m3). It is named by the project's identifier, its number and its name, and linked to
 * it. While it is read, or once nerve refuses it (gone, or not the caller's to see), there is no row.
 */
export const IntakeDuplicateOf = observer(function IntakeDuplicateOf(props: Props) {
  const { workspaceSlug, projectId, duplicateToId } = props;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { rootIssueStore: root, issue: detail } = useIssueDetail();
  const { getProjectById } = useProject();
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
  const identity = root.permissionReads.identity(workspace?.id, [projectId], true);
  const read = useSessionSWR(
    duplicateToId ? ["intake-duplicate", identity, workspaceSlug, projectId, duplicateToId] : null,
    async (_identity, ws, project, id) => (await detail.fetchIssueCore(ws, project, id)) ?? null
  );
  const project = getProjectById(projectId);
  const target = duplicateToId && read.data && !read.error ? detail.getIssueById(duplicateToId) : undefined;
  if (!target || !project) return null;
  const link = generateWorkItemLink({
    workspaceSlug,
    projectId,
    issueId: target.id,
    projectIdentifier: project.identifier,
    sequenceId: target.sequence_id,
  });

  return (
    <div className="mt-3 flex min-h-8 items-center gap-2">
      <div className="flex w-2/5 flex-shrink-0 items-center gap-1 text-13 text-tertiary">
        <DuplicateOfOutline className="h-4 w-4 flex-shrink-0" />
        <span>{t("issue.relation.duplicate")}</span>
      </div>
      <ControlLink
        href={link}
        onClick={() => void navigate(link)}
        target="_self"
        className="w-3/5 flex-grow truncate rounded-sm px-2 py-1 text-13 text-secondary hover:bg-layer-1"
      >
        {`${project.identifier}-${target.sequence_id} · ${target.name}`}
      </ControlLink>
    </div>
  );
});
