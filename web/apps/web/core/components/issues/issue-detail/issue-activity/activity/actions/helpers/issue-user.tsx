/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { Link } from "react-router";
import { observer } from "mobx-react";
import { SITE_NAME } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useMember } from "@/hooks/store/use-member";

type IssueUserProps = { workspaceSlug: string; issueId: string; activityId: string };
export const IssueUser = observer(function IssueUser({ workspaceSlug, issueId, activityId }: IssueUserProps) {
  const { activity } = useIssueDetail();
  const { getUserDetails } = useMember();
  const { t } = useTranslation();
  const row = activity.getActivityById(issueId, activityId);
  if (!row) return null;
  const user = row.actor_id ? getUserDetails(row.actor_id) : undefined;
  if (row.actor_id === null || !user)
    return (
      <span className="font-medium text-primary">
        {row.actor_id === null ? SITE_NAME : t("common.deactivated_user")}
      </span>
    );
  return (
    <Link to={`/${workspaceSlug}/profile/${row.actor_id}`} className="font-medium text-primary hover:underline">
      {user.display_name}
    </Link>
  );
});
