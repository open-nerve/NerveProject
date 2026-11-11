/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { MembersOutline, UserAltOutline, WorkItemsOutline } from "@makeplane/propel/icons";
// nerve imports
import type { Cycle, Progress } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Loader } from "@nerve/ui";
// components
import { ProfileAvatar } from "@/components/assets/profile-image";
import { AvatarGroupOverflow } from "@/components/common/avatar-group-overflow";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useRouteWorkspace } from "@/hooks/use-route-workspace";

type Props = {
  cycle: Cycle;
  projectId: string;
  /** The cycle's members, the assignees of its work items: its list's row's; undefined until a list has its row. */
  members: string[] | undefined;
  /** The counts the sidebar shows (cycleCounts' choice); undefined until nerve has read its progress. */
  counts: Pick<Progress, "total_issues" | "completed_issues" | "cancelled_issues"> | undefined;
};

/**
 * The details of a cycle's progress sidebar (P5a spec 2.5): its description, plain text; its lead and its members, by
 * the project's members (one who has left it is named so); and its work items, the completed of those not cancelled
 * (progressPercent's counts, M6 design 3.5).
 */
export const CycleSidebarDetails = observer(function CycleSidebarDetails(props: Props) {
  const { cycle, projectId, members, counts } = props;
  // hooks
  const { t } = useTranslation();
  const imageWorkspace = useRouteWorkspace();
  const {
    project: { getProjectMemberDetails },
  } = useMember();

  const memberOf = (accountId: string) => getProjectMemberDetails(accountId, projectId)?.member;
  const nameOf = (accountId: string) => memberOf(accountId)?.display_name ?? t("distribution.former_member");
  const avatarOf = (accountId: string, size?: "xs") => (
    <ProfileAvatar
      key={accountId}
      alt={nameOf(accountId)}
      fallback={memberOf(accountId)?.display_name[0]?.toUpperCase()}
      source={{ kind: "member", workspaceID: imageWorkspace?.id ?? "", memberID: accountId }}
      size={size}
    />
  );

  return (
    <div className="flex w-full flex-col gap-5">
      {cycle.description && (
        <p className="text-14 break-words whitespace-pre-wrap text-secondary">{cycle.description}</p>
      )}

      <dl className="flex flex-col gap-5 pt-2.5 pb-6">
        <Detail icon={<UserAltOutline className="h-4 w-4" />} title={t("lead")}>
          {cycle.owned_by_id === null ? (
            <span className="text-13 text-tertiary">{t("none")}</span>
          ) : (
            <div className="flex items-center gap-2.5">
              {avatarOf(cycle.owned_by_id, "xs")}
              <span className="text-13 text-secondary">{nameOf(cycle.owned_by_id)}</span>
            </div>
          )}
        </Detail>

        <Detail icon={<MembersOutline className="h-4 w-4" />} title={t("members")}>
          {members === undefined ? (
            <Loader>
              <Loader.Item height="20px" width="60px" />
            </Loader>
          ) : members.length > 0 ? (
            <AvatarGroupOverflow size="xs">{members.map((member) => avatarOf(member))}</AvatarGroupOverflow>
          ) : (
            <span className="px-1.5 text-13 text-tertiary">{t("no_assignee")}</span>
          )}
        </Detail>

        <Detail icon={<WorkItemsOutline className="h-4 w-4" />} title={t("work_items")}>
          {counts === undefined ? (
            <Loader>
              <Loader.Item height="20px" width="40px" />
            </Loader>
          ) : (
            <span className="px-1.5 text-13 text-tertiary">
              {`${counts.completed_issues}/${counts.total_issues - counts.cancelled_issues}`}
            </span>
          )}
        </Detail>
      </dl>
    </div>
  );
});

/** A row of the details: its title, by its icon, and its value. */
function Detail({ icon, title, children }: { icon: ReactNode; title: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-start gap-1">
      <dt className="flex w-2/5 items-center justify-start gap-2 text-tertiary">
        {icon}
        <span className="text-14">{title}</span>
      </dt>
      <dd className="flex w-3/5 items-center rounded-xs">{children}</dd>
    </div>
  );
}
