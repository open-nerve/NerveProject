/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, useState } from "react";
import { observer } from "mobx-react";
import useSWR from "swr";
import { useTranslation } from "@nerve/i18n";
// nerve types
import { ProjectsOutline, WorkItemsOutline } from "@makeplane/propel/icons";
import type { TActivityEntityData, TRecentActivityFilterKeys } from "@nerve/types";
// components
import { ContentOverflowWrapper } from "@/components/core/content-overflow-HOC";
// services
import { WorkspaceService } from "@/services/workspace.service";
import { RecentsEmptyState } from "../empty-states";
import { RecentActivityWidgetLoader } from "../loaders";
import { FiltersDropdown } from "./filters";
import { RecentIssue } from "./issue";
import { RecentProject } from "./project";

const workspaceService = new WorkspaceService();
const filters: { name: TRecentActivityFilterKeys; icon?: React.ReactNode; i18n_key: string }[] = [
  { name: "all item", i18n_key: "home.recents.filters.all" },
  { name: "issue", icon: <WorkItemsOutline className="h-4 w-4" />, i18n_key: "home.recents.filters.issues" },
  { name: "project", icon: <ProjectsOutline height={16} width={16} />, i18n_key: "home.recents.filters.projects" },
];

type TRecentWidgetProps = {
  workspaceSlug: string;
  presetFilter?: TRecentActivityFilterKeys;
  showFilterSelect?: boolean;
};

export const RecentActivityWidget = observer(function RecentActivityWidget(props: TRecentWidgetProps) {
  const { presetFilter, showFilterSelect = true, workspaceSlug } = props;
  // states
  const [filter, setFilter] = useState<TRecentActivityFilterKeys>(presetFilter ?? filters[0].name);
  const { t } = useTranslation();
  // ref
  const ref = useRef<HTMLDivElement>(null);

  const { data: recents, isLoading } = useSWR(
    workspaceSlug ? `WORKSPACE_RECENT_ACTIVITY_${workspaceSlug}_${filter}` : null,
    workspaceSlug
      ? () => workspaceService.fetchWorkspaceRecents(workspaceSlug, filter === filters[0].name ? undefined : filter)
      : null,
    {
      revalidateIfStale: false,
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
    }
  );

  const resolveRecent = (activity: TActivityEntityData) => {
    switch (activity.entity_name) {
      case "project":
        return <RecentProject activity={activity} ref={ref} workspaceSlug={workspaceSlug} />;
      case "issue":
        return <RecentIssue activity={activity} ref={ref} workspaceSlug={workspaceSlug} />;
      default:
        return <></>;
    }
  };

  if (!isLoading && recents?.length === 0)
    return (
      <div ref={ref} className="max-h-[500px] overflow-y-scroll">
        <div className="mb-4 flex items-center justify-between">
          <div className="text-14 font-semibold text-tertiary">{t("home.recents.title")}</div>
          {showFilterSelect && <FiltersDropdown filters={filters} activeFilter={filter} setActiveFilter={setFilter} />}
        </div>
        <div className="flex flex-col items-center justify-center">
          <RecentsEmptyState type={filter} />
        </div>
      </div>
    );

  return (
    <ContentOverflowWrapper
      maxHeight={415}
      containerClassName="box-border min-h-[250px]"
      fallback={<></>}
      buttonClassName="bg-surface-2/20"
    >
      <div className="mb-2 flex items-center justify-between">
        <div className="text-14 font-semibold text-tertiary">{t("home.recents.title")}</div>
        {showFilterSelect && <FiltersDropdown filters={filters} activeFilter={filter} setActiveFilter={setFilter} />}
      </div>
      <div className="flex min-h-[250px] flex-col">
        {isLoading && <RecentActivityWidgetLoader />}
        {!isLoading &&
          recents
            ?.filter((recent) => recent.entity_data)
            .map((activity) => <div key={activity.id}>{resolveRecent(activity)}</div>)}
      </div>
    </ContentOverflowWrapper>
  );
});
