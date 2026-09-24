/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useMemo } from "react";
import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
import { ChevronDownOutline, ChevronUpOutline, WarningCircleOutline } from "@makeplane/propel/icons";
import { Disclosure, Transition } from "@headlessui/react";
import { useTranslation } from "@nerve/i18n";
import { EIssuesStoreType } from "@nerve/types";
// components
// constants
// helpers
import { getDate } from "@nerve/utils";
import ProgressChart from "@/components/core/sidebar/progress-chart";
import { ModuleProgressStats } from "@/components/modules";
// hooks
import { useModule } from "@/hooks/store/use-module";
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
// plane web constants
type TModuleProgressProps = {
  moduleId: string;
};

export const ModuleProgress = observer(function ModuleProgress(props: TModuleProgressProps) {
  // props
  const { moduleId } = props;
  // router
  const [searchParams] = useSearchParams();
  const peekModule = searchParams.get("peekModule") || undefined;
  // plane hooks
  const { t } = useTranslation();
  // hooks
  const { getModuleById } = useModule();
  const { getFilter, updateFilterValueFromSidebar } = useWorkItemFilters();
  // derived values
  const moduleFilter = getFilter(EIssuesStoreType.MODULE, moduleId);
  const selectedAssignees = moduleFilter?.findFirstConditionByPropertyAndOperator("assignee_id", "in");
  const selectedLabels = moduleFilter?.findFirstConditionByPropertyAndOperator("label_id", "in");
  const selectedStateGroups = moduleFilter?.findFirstConditionByPropertyAndOperator("state_group", "in");
  const moduleDetails = getModuleById(moduleId);
  const completedIssues = moduleDetails?.completed_issues || 0;
  const totalIssues = moduleDetails?.total_issues || 0;
  const progressHeaderPercentage =
    completedIssues != 0 && totalIssues != 0 ? Math.round((completedIssues / totalIssues) * 100) : 0;
  const chartDistributionData = moduleDetails?.distribution || undefined;
  const completionChartDistributionData = chartDistributionData?.completion_chart || undefined;
  const groupedIssues = useMemo(
    () => ({
      backlog: moduleDetails?.backlog_issues || 0,
      unstarted: moduleDetails?.unstarted_issues || 0,
      started: moduleDetails?.started_issues || 0,
      completed: moduleDetails?.completed_issues || 0,
      cancelled: moduleDetails?.cancelled_issues || 0,
    }),
    [moduleDetails]
  );
  const moduleStartDate = getDate(moduleDetails?.start_date);
  const moduleEndDate = getDate(moduleDetails?.target_date);
  const isModuleStartDateValid = moduleStartDate && moduleStartDate <= new Date();
  const isModuleEndDateValid = moduleStartDate && moduleEndDate && moduleEndDate >= moduleStartDate;
  const isModuleDateValid = isModuleStartDateValid && isModuleEndDateValid;

  if (!moduleDetails) return <></>;
  return (
    <div className="space-y-4 border-t border-subtle px-3 py-4">
      <Disclosure defaultOpen={isModuleDateValid ? true : false}>
        {({ open }) => (
          <div className="space-y-6">
            {/* progress bar header */}
            {isModuleDateValid ? (
              <div className="relative flex w-full items-center justify-between gap-2">
                <Disclosure.Button className="relative flex w-full items-center gap-2">
                  <div className="text-13 font-medium text-secondary">{t("common.progress")}</div>
                  {progressHeaderPercentage > 0 && (
                    <div className="bg-amber-500/20 text-amber-500 flex h-5 w-9 items-center justify-center rounded-sm text-11 font-medium">{`${progressHeaderPercentage}%`}</div>
                  )}
                </Disclosure.Button>
                <Disclosure.Button className="ml-auto">
                  {open ? (
                    <ChevronUpOutline className="h-3.5 w-3.5" aria-hidden="true" />
                  ) : (
                    <ChevronDownOutline className="h-3.5 w-3.5" aria-hidden="true" />
                  )}
                </Disclosure.Button>
              </div>
            ) : (
              <div className="relative flex w-full items-center justify-between gap-2">
                <div className="text-13 font-medium text-secondary">Progress</div>
                <div className="flex items-center gap-1">
                  <WarningCircleOutline height={14} width={14} className="text-secondary" />
                  <span className="text-11 text-secondary italic">
                    {moduleDetails?.start_date && moduleDetails?.target_date
                      ? t("project_module.empty_state.sidebar.in_active")
                      : t("project_module.empty_state.sidebar.invalid_date")}
                  </span>
                </div>
              </div>
            )}

            <Transition as="div" show={open}>
              <Disclosure.Panel className="space-y-4">
                {/* progress burndown chart */}
                <div>
                  {moduleStartDate && moduleEndDate && completionChartDistributionData && (
                    <Fragment>
                      <ProgressChart
                        distribution={completionChartDistributionData}
                        totalIssues={totalIssues}
                        plotTitle={"work items"}
                      />
                    </Fragment>
                  )}
                </div>

                {/* progress detailed view */}
                {chartDistributionData && (
                  <div className="w-full border-t border-subtle pt-5">
                    <ModuleProgressStats
                      distribution={chartDistributionData}
                      groupedIssues={groupedIssues}
                      handleFiltersUpdate={updateFilterValueFromSidebar.bind(
                        updateFilterValueFromSidebar,
                        EIssuesStoreType.MODULE,
                        moduleId
                      )}
                      isEditable={Boolean(!peekModule) && moduleFilter !== undefined}
                      moduleId={moduleId}
                      noBackground={false}
                      roundedTab={false}
                      selectedFilters={{
                        assignees: selectedAssignees,
                        labels: selectedLabels,
                        stateGroups: selectedStateGroups,
                      }}
                      size="xs"
                      totalIssuesCount={totalIssues}
                    />
                  </div>
                )}
              </Disclosure.Panel>
            </Transition>
          </div>
        )}
      </Disclosure>
    </div>
  );
});
