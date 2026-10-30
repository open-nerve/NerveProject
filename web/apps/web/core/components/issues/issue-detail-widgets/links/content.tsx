/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { DetailResourceError } from "../resource-error";
// components
import { LinkList } from "../../issue-detail/links";
// helper
import { useLinkOperations } from "./helper";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
};

export const IssueLinksCollapsibleContent = observer(function IssueLinksCollapsibleContent(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled } = props;

  // helper
  const handleLinkOperations = useLinkOperations(workspaceSlug, projectId, issueId);

  const { link } = useIssueDetail();
  return (
    <>
      <DetailResourceError
        error={link.getError(issueId)}
        retry={() => link.retry(workspaceSlug, projectId, issueId)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
      />
      <LinkList issueId={issueId} linkOperations={handleLinkOperations} disabled={disabled} />
    </>
  );
});
