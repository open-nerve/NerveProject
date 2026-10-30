/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { observer } from "mobx-react";
import { Link, useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import { generateWorkItemLink } from "@nerve/utils";
import { useProject } from "@/hooks/store/use-project";

export const SpreadsheetSubIssueColumn = observer(function SpreadsheetSubIssueColumn({ issue }: { issue: Issue }) {
  const { workspaceSlug } = useParams();
  const { getProjectIdentifierById } = useProject();
  const projectIdentifier = getProjectIdentifierById(issue.project_id);
  const count = issue.sub_issues_count ?? 0;
  const label = `${count} sub-work item${count !== 1 ? "s" : ""}`;
  const className = "flex h-11 w-full items-center border-b-[0.5px] border-subtle py-1 text-11";
  if (
    !count ||
    !workspaceSlug ||
    !projectIdentifier ||
    !Number.isSafeInteger(issue.sequence_id) ||
    issue.sequence_id < 1
  )
    return <span className={className}>{label}</span>;
  const href = generateWorkItemLink({
    workspaceSlug,
    projectIdentifier,
    sequenceId: issue.sequence_id,
    projectId: issue.project_id,
    issueId: issue.id,
  });
  return (
    <Link to={href} className={`${className} clickable hover:bg-surface-2`}>
      {label}
    </Link>
  );
});
