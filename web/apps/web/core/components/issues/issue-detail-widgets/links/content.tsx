/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { DetailResourceError } from "../resource-error";
// components
import { LinkList } from "../../issue-detail/links";
// helper
import { useLinkOperations } from "./helper";
import { useLinkMetadataRefresh } from "./use-link-metadata-refresh";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
  active?: boolean;
};

export const IssueLinksCollapsibleContent = observer(function IssueLinksCollapsibleContent(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled, active = true } = props;
  const { t } = useTranslation();
  const refresh = useLinkMetadataRefresh({ workspaceSlug, projectId, issueId, active });

  // helper
  const handleLinkOperations = useLinkOperations(workspaceSlug, projectId, issueId);

  const { link } = useIssueDetail();
  return (
    <>
      <DetailResourceError
        error={link.getError(issueId)}
        retry={refresh.retry}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
      />
      {refresh.expired && !link.getError(issueId) && (
        <div role="status">
          {t("links.metadata.expired")}{" "}
          <button type="button" disabled={refresh.busy} onClick={() => void refresh.retry()}>
            {t("common.retry")}
          </button>
        </div>
      )}
      <LinkList issueId={issueId} linkOperations={handleLinkOperations} disabled={disabled} />
    </>
  );
});
