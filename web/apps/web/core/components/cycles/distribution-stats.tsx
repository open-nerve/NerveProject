/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
// nerve imports
import type { Progress } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
// assets
import userImage from "@/app/assets/user.png?url";
// components
import { ProfileAvatar } from "@/components/assets/profile-image";
import { SingleProgressStats } from "@/components/core/sidebar/single-progress-stats";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";
// local imports
import type { AssigneeRow, LabelRow } from "./distribution-rows";
import { assigneeRows, labelRows } from "./distribution-rows";

// The breakdowns of a cycle's progress by assignee and by label as a page shows them (P5a spec 2.4): its sidebar's and
// the card of the cycle in progress. The rows are distribution-rows.ts's, named by the project's members and labels;
// a row does not set the page's filters (P5b).

/**
 * The rows of a progress's breakdowns by assignee and by label, of the project's members and labels (each by the
 * project: none of a project the caller no longer sees).
 */
export function useDistributionRows(progress: Pick<Progress, "assignees" | "labels">, projectId: string) {
  const { t } = useTranslation();
  const {
    project: { getProjectMemberDetails },
  } = useMember();
  const { getProjectLabels } = useLabel();
  return {
    assignees: assigneeRows(progress.assignees, (id) => getProjectMemberDetails(id, projectId)?.member, t),
    labels: labelRows(progress.labels, getProjectLabels(projectId) ?? [], t),
  };
}

/** The work items by assignee: each row's member by avatar and name (a placeholder for none), and its share. */
export const AssigneeStats = observer(function AssigneeStats({ rows }: { rows: AssigneeRow[] }) {
  const imageWorkspace = useRouteWorkspace();
  return (
    <ul>
      {rows.map((row) => (
        <li key={row.assignee_id ?? "none"}>
          <SingleProgressStats
            title={
              <div className="flex items-center gap-2">
                {row.avatarOf === null ? (
                  <img src={userImage} className="size-4 rounded-full object-cover" alt="" aria-hidden="true" />
                ) : (
                  <ProfileAvatar
                    alt={row.name}
                    fallback={row.name[0]?.toUpperCase()}
                    source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: row.avatarOf }}
                    size="xs"
                  />
                )}
                <span>{row.name}</span>
              </div>
            }
            completed={row.completed_issues}
            total={row.total_issues}
          />
        </li>
      ))}
    </ul>
  );
});

/** The work items by label: each row's label by its color and name, and its share. */
export function LabelStats({ rows }: { rows: LabelRow[] }) {
  return (
    <ul>
      {rows.map((row) => (
        <li key={row.label_id ?? "none"}>
          <SingleProgressStats
            title={
              <div className="flex items-center gap-2 truncate">
                <span
                  className="block h-3 w-3 flex-shrink-0 rounded-full"
                  style={{ backgroundColor: row.color ?? undefined }}
                  aria-hidden="true"
                />
                <span className="truncate text-11 text-ellipsis">{row.name}</span>
              </div>
            }
            completed={row.completed_issues}
            total={row.total_issues}
          />
        </li>
      ))}
    </ul>
  );
}
