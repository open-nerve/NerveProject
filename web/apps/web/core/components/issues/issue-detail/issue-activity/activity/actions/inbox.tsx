/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { IntakeOutline } from "@makeplane/propel/icons";
// nerve imports
import type { IntakeStatus } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
// components
import { IssueActivityBlockComponent } from "./";

type TIssueInboxActivity = {
  activityId: string;
  issueId: string;
  workspaceSlug: string;
  ends: "top" | "bottom" | undefined;
};

/**
 * What a triage's row says, by the status the work item was moved to (P5a spec 2.7, R18): nerve writes it on the field
 * intake, its verb that status's name (pending ends a snooze), each key whole.
 */
const TRIAGED: ReadonlyMap<string, string> = new Map<IntakeStatus, string>([
  ["pending", "inbox_issue.activity.pending"],
  ["snoozed", "inbox_issue.activity.snoozed"],
  ["accepted", "inbox_issue.activity.accepted"],
  ["declined", "inbox_issue.activity.declined"],
  ["duplicate", "inbox_issue.activity.duplicate"],
]);

/**
 * A triage's row in a work item's activity: the status it was moved to, by its name. The contract leaves the verb an
 * open string (re-94 O1), so a status it does not name says that the intake status changed.
 */
export const IssueInboxActivity = observer(function IssueInboxActivity(props: TIssueInboxActivity) {
  const { activityId, issueId, workspaceSlug, ends } = props;
  const { t } = useTranslation();
  // hooks
  const {
    activity: { getActivityById },
  } = useIssueDetail();

  const activity = getActivityById(issueId, activityId);

  if (!activity) return <></>;
  return (
    <IssueActivityBlockComponent
      icon={<IntakeOutline className="h-4 w-4 flex-shrink-0 text-secondary" />}
      activityId={activityId}
      issueId={issueId}
      workspaceSlug={workspaceSlug}
      ends={ends}
    >
      <>{t(TRIAGED.get(activity.verb) ?? "inbox_issue.activity.updated")}</>
    </IssueActivityBlockComponent>
  );
});
