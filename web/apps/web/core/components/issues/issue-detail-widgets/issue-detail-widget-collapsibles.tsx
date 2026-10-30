/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { observer } from "mobx-react";
// nerve imports
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { ISSUE_RELATION_OPTIONS } from "@/components/relations";
// local imports
import { LinksCollapsible } from "./links";
import { RelationsCollapsible } from "./relations";
import { SubIssuesCollapsible } from "./sub-issues";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
};

export const IssueDetailWidgetCollapsibles = observer(function IssueDetailWidgetCollapsibles(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled } = props;
  // store hooks
  const {
    issue: { getIssueById },
    subIssues: { subIssuesByIssueId },
    relation: { getRelationCountByIssueId },
  } = useIssueDetail();
  // derived values
  const issue = getIssueById(issueId);
  const subIssues = subIssuesByIssueId(issueId);
  const issueRelationsCount = getRelationCountByIssueId(issueId, ISSUE_RELATION_OPTIONS);
  // render conditions
  const shouldRenderSubIssues = !!subIssues && subIssues.length > 0;
  const shouldRenderRelations = issueRelationsCount > 0;
  const shouldRenderLinks = !!issue?.link_count && issue?.link_count > 0;

  return (
    <div className="flex flex-col">
      {shouldRenderSubIssues && (
        <SubIssuesCollapsible
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={disabled}
        />
      )}
      {shouldRenderRelations && (
        <RelationsCollapsible workspaceSlug={workspaceSlug} issueId={issueId} disabled={disabled} />
      )}
      {shouldRenderLinks && (
        <LinksCollapsible workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} disabled={disabled} />
      )}
    </div>
  );
});
