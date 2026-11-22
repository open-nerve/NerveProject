/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { Link } from "react-router";
import { observer } from "mobx-react";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useActorName } from "@/hooks/use-actor-name";

type IssueUserProps = { workspaceSlug: string; issueId: string; activityId: string };
/** An activity's maker, as useActorName shows him (P4 spec R22): a member links to his profile. */
export const IssueUser = observer(function IssueUser({ workspaceSlug, issueId, activityId }: IssueUserProps) {
  const { activity } = useIssueDetail();
  const row = activity.getActivityById(issueId, activityId);
  const actor = useActorName(row?.actor_id ?? null);
  if (!row) return null;
  if (!actor.member) return <span className="font-medium text-primary">{actor.name}</span>;
  return (
    <Link to={`/${workspaceSlug}/profile/${row.actor_id}`} className="font-medium text-primary hover:underline">
      {actor.name}
    </Link>
  );
});
