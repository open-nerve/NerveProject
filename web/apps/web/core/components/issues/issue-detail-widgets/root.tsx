/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
// nerve imports
// local imports
import { IssueDetailWidgetActionButtons } from "./action-buttons";
import { IssueDetailWidgetCollapsibles } from "./issue-detail-widget-collapsibles";
import { IssueDetailWidgetModals } from "./issue-detail-widget-modals";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
};

export function IssueDetailWidgets(props: Props) {
  const { workspaceSlug, projectId, issueId, disabled } = props;

  return (
    <>
      <div className="flex flex-col space-y-4">
        {!disabled && <IssueDetailWidgetActionButtons issueId={issueId} disabled={disabled} />}
        <IssueDetailWidgetCollapsibles
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          disabled={disabled}
        />
      </div>
      <IssueDetailWidgetModals workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} />
    </>
  );
}
