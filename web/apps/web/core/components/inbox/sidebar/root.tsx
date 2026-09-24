/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import type { TInboxIssueCurrentTab } from "@nerve/types";
import { EInboxIssueCurrentTab } from "@nerve/types";
// nerve imports
import { Header, Loader, EHeaderVariant } from "@nerve/ui";
import { cn } from "@nerve/utils";
// components
import { InboxSidebarLoader } from "@/components/ui/loader/layouts/project-inbox/inbox-sidebar-loader";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useProjectInbox } from "@/hooks/store/use-project-inbox";
import { useNavigate } from "react-router";
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
// local imports
import { FiltersRoot } from "../inbox-filter";
import { InboxIssueAppliedFilters } from "../inbox-filter/applied-filters/root";
import { InboxIssueList } from "./inbox-list";

type IInboxSidebarProps = {
  workspaceSlug: string;
  projectId: string;
  inboxIssueId: string | undefined;
  setIsMobileSidebar: (value: boolean) => void;
};

const tabNavigationOptions: { key: TInboxIssueCurrentTab; i18n_label: string }[] = [
  {
    key: EInboxIssueCurrentTab.OPEN,
    i18n_label: "inbox_issue.tabs.open",
  },
  {
    key: EInboxIssueCurrentTab.CLOSED,
    i18n_label: "inbox_issue.tabs.closed",
  },
];

export const InboxSidebar = observer(function InboxSidebar(props: IInboxSidebarProps) {
  const { workspaceSlug, projectId, inboxIssueId, setIsMobileSidebar } = props;
  // router
  const navigate = useNavigate();
  // ref
  const containerRef = useRef<HTMLDivElement>(null);
  const [elementRef, setElementRef] = useState<HTMLDivElement | null>(null);
  // nerve hooks
  const { t } = useTranslation();
  // store
  const { currentProjectDetails } = useProject();
  const {
    currentTab,
    handleCurrentTab,
    loader,
    filteredInboxIssueIds,
    inboxIssuePaginationInfo,
    fetchInboxPaginationIssues,
    getAppliedFiltersCount,
  } = useProjectInbox();
  // derived values
  const fetchNextPages = useCallback(() => {
    if (!workspaceSlug || !projectId) return;
    fetchInboxPaginationIssues(workspaceSlug, projectId);
  }, [workspaceSlug, projectId, fetchInboxPaginationIssues]);

  // page observer
  useIntersectionObserver(containerRef, elementRef, fetchNextPages, "20%");

  useEffect(() => {
    if (workspaceSlug && projectId && currentTab && filteredInboxIssueIds.length > 0) {
      if (inboxIssueId === undefined) {
        navigate(
          `/${workspaceSlug}/projects/${projectId}/intake?currentTab=${currentTab}&inboxIssueId=${filteredInboxIssueIds[0]}`,
          { replace: true }
        );
      }
    }
  }, [currentTab, filteredInboxIssueIds, inboxIssueId, projectId, navigate, workspaceSlug]);

  return (
    <div className="h-full w-full flex-shrink-0 border-r border-strong bg-surface-1">
      <div className="relative flex h-full w-full flex-col overflow-hidden">
        <Header variant={EHeaderVariant.SECONDARY}>
          {tabNavigationOptions.map((option) => (
            <div
              key={option?.key}
              className={cn(
                `relative flex h-full cursor-pointer items-center gap-1 px-3 text-13 font-medium transition-all`,
                currentTab === option?.key ? `text-accent-primary` : `hover:text-secondary`
              )}
              onClick={() => {
                if (currentTab != option?.key) {
                  handleCurrentTab(workspaceSlug, projectId, option?.key);
                  navigate(`/${workspaceSlug}/projects/${projectId}/intake?currentTab=${option?.key}`);
                }
              }}
            >
              <div>{t(option?.i18n_label)}</div>
              {option?.key === "open" && currentTab === option?.key && (
                <div className="rounded-full bg-accent-primary/20 p-1.5 py-0.5 text-11 font-semibold text-accent-primary">
                  {inboxIssuePaginationInfo?.total_results || 0}
                </div>
              )}
              <div
                className={cn(
                  `absolute right-0 bottom-0 left-0 rounded-t-md border`,
                  currentTab === option?.key ? `border-accent-strong` : `border-transparent`
                )}
              />
            </div>
          ))}
          <div className="m-auto mr-0">
            <FiltersRoot />
          </div>
        </Header>
        <InboxIssueAppliedFilters />

        {loader != undefined && loader === "filter-loading" && !inboxIssuePaginationInfo?.next_page_results ? (
          <InboxSidebarLoader />
        ) : (
          <div
            className="vertical-scrollbar scrollbar-md h-full w-full overflow-hidden overflow-y-auto"
            ref={containerRef}
          >
            {filteredInboxIssueIds.length > 0 ? (
              <InboxIssueList
                setIsMobileSidebar={setIsMobileSidebar}
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                projectIdentifier={currentProjectDetails?.identifier}
                inboxIssueIds={filteredInboxIssueIds}
              />
            ) : (
              <div className="flex h-full w-full items-center justify-center">
                {getAppliedFiltersCount > 0 ? (
                  <EmptyStateDetailed
                    assetKey="search"
                    title={t("common_empty_state.search.title")}
                    description={t("common_empty_state.search.description")}
                    assetClassName="size-20"
                    rootClassName="px-page-x"
                  />
                ) : currentTab === EInboxIssueCurrentTab.OPEN ? (
                  <EmptyStateDetailed
                    assetKey="inbox"
                    title={t("project_empty_state.intake_sidebar.title")}
                    description={t("project_empty_state.intake_sidebar.description")}
                    assetClassName="size-20"
                    actions={[
                      {
                        label: t("project_empty_state.intake_sidebar.cta_primary"),
                        onClick: () => navigate(`/${workspaceSlug}/projects/${projectId}/intake`),
                        variant: "primary",
                      },
                    ]}
                    rootClassName="px-page-x"
                  />
                ) : (
                  // TODO: Add translation
                  <EmptyStateDetailed
                    assetKey="inbox"
                    title="No request closed yet"
                    description="All the work items whether accepted or declined can be found here."
                    assetClassName="size-20"
                    className="px-10"
                  />
                )}
              </div>
            )}
            <div ref={setElementRef}>
              {inboxIssuePaginationInfo?.next_page_results && (
                <Loader className="mx-auto w-full space-y-4 px-2 py-4">
                  <Loader.Item height="64px" width="w-100" />
                  <Loader.Item height="64px" width="w-100" />
                </Loader>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
});
