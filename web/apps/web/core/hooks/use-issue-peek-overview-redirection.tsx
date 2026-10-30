/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";
import { openIssuePeek } from "@/components/issues/peek-overview/selection";

import { useNavigate } from "react-router";
// types

// helpers
import { generateWorkItemLink } from "@nerve/utils";
// hooks
import { useIssueDetail } from "./store/use-issue-detail";
import { useProject } from "./store/use-project";

const useIssuePeekOverviewRedirection = () => {
  // router
  const navigate = useNavigate();
  //   store hooks
  const detail = useIssueDetail();
  const { getIsIssuePeeked } = detail;
  const { getProjectIdentifierById } = useProject();

  const handleRedirection = (
    workspaceSlug: string | undefined,
    issue: Issue | undefined,
    isMobile = false,
    nestingLevel?: number,
    trigger?: HTMLElement
  ) => {
    if (!issue) return;
    const { project_id, id, archived_at } = issue;
    const projectIdentifier = getProjectIdentifierById(issue?.project_id);

    const workItemLink = generateWorkItemLink({
      workspaceSlug,
      projectId: project_id,
      issueId: id,
      projectIdentifier,
      sequenceId: issue?.sequence_id,
      isArchived: !!archived_at,
    });
    if (workspaceSlug && project_id && id && !getIsIssuePeeked(id)) {
      if (isMobile) {
        navigate(workItemLink);
      } else {
        openIssuePeek(
          detail,
          { workspaceSlug, projectId: project_id, issueId: id, nestingLevel, isArchived: !!archived_at },
          trigger
        );
      }
    }
  };

  return { handleRedirection };
};

export default useIssuePeekOverviewRedirection;
