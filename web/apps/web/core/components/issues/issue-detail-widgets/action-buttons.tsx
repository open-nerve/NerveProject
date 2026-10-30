/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
import { LinkOutline, RelationsOutline, ViewsOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
// nerve imports
// local imports
import { IssueLinksActionButton } from "./links";
import { RelationActionButton } from "./relations";
import { SubIssuesActionButton } from "./sub-issues";
import { IssueDetailWidgetButton } from "./widget-button";

type Props = {
  issueId: string;
  disabled: boolean;
};

export function IssueDetailWidgetActionButtons(props: Props) {
  const { issueId, disabled } = props;
  // translation
  const { t } = useTranslation();

  return (
    <div className="flex flex-wrap items-center gap-2">
      <SubIssuesActionButton
        issueId={issueId}
        customButton={
          <IssueDetailWidgetButton
            title={t("issue.add.sub_issue")}
            icon={<ViewsOutline className="h-3.5 w-3.5 flex-shrink-0" />}
          />
        }
        disabled={disabled}
      />
      <RelationActionButton
        issueId={issueId}
        customButton={
          <IssueDetailWidgetButton
            title={t("issue.add.relation")}
            icon={<RelationsOutline className="h-3.5 w-3.5 flex-shrink-0" />}
          />
        }
        disabled={disabled}
      />
      <IssueLinksActionButton
        customButton={
          <IssueDetailWidgetButton
            title={t("issue.add.link")}
            icon={<LinkOutline className="h-3.5 w-3.5 flex-shrink-0" />}
          />
        }
        disabled={disabled}
      />
    </div>
  );
}
