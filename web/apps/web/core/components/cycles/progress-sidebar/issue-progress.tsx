/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { isEmpty } from "lodash-es";
import { observer } from "mobx-react";
import { useSearchParams } from "next/navigation";
import { Disclosure, Transition } from "@headlessui/react";
// plane imports
import { useTranslation } from "@plane/i18n";
import { ChevronDownOutline, ChevronUpOutline } from "@makeplane/propel/icons";
import type { ICycle, TProgressSnapshot } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { getDate } from "@plane/utils";
// hooks
import { useCycle } from "@/hooks/store/use-cycle";
// plane web components
import { useWorkItemFilters } from "@/hooks/store/work-item-filters/use-work-item-filters";
// local imports
import { CycleProgressStats } from "./progress-stats";
import { SidebarChart } from "./sidebar-chart";

type TCycleProgressProps = {
  cycleId: string;
};

export const validateCycleSnapshot = (cycleDetails: ICycle | null): ICycle | null => {
  if (!cycleDetails || cycleDetails === null) return cycleDetails;

  const updatedCycleDetails: any = { ...cycleDetails };
  if (!isEmpty(cycleDetails.progress_snapshot)) {
    Object.keys(cycleDetails.progress_snapshot || {}).forEach((key) => {
      const currentKey = key as keyof TProgressSnapshot;
      if (!isEmpty(cycleDetails.progress_snapshot) && !isEmpty(updatedCycleDetails)) {
        updatedCycleDetails[currentKey as keyof ICycle] = cycleDetails?.progress_snapshot?.[currentKey];
      }
    });
  }
  return updatedCycleDetails;
};

export const CycleProgress = observer(function CycleProgress(props: TCycleProgressProps) {
  // props
  const { cycleId } = props;
  // router
  const searchParams = useSearchParams();
  const peekCycle = searchParams.get("peekCycle") || undefined;
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { getCycleById } = useCycle();
  const { getFilter, updateFilterValueFromSidebar } = useWorkItemFilters();
  // derived values
  const cycleFilter = getFilter(EIssuesStoreType.CYCLE, cycleId);
  const selectedAssignees = cycleFilter?.findFirstConditionByPropertyAndOperator("assignee_id", "in");
  const selectedLabels = cycleFilter?.findFirstConditionByPropertyAndOperator("label_id", "in");
  const selectedStateGroups = cycleFilter?.findFirstConditionByPropertyAndOperator("state_group", "in");
  const cycleDetails = validateCycleSnapshot(getCycleById(cycleId));
  const totalIssues = cycleDetails?.total_issues || 0;
  const chartDistributionData = cycleDetails?.distribution || undefined;
  const groupedIssues = useMemo(
    () => ({
      backlog: cycleDetails?.backlog_issues || 0,
      unstarted: cycleDetails?.unstarted_issues || 0,
      started: cycleDetails?.started_issues || 0,
      completed: cycleDetails?.completed_issues || 0,
      cancelled: cycleDetails?.cancelled_issues || 0,
    }),
    [cycleDetails]
  );
  const cycleStartDate = getDate(cycleDetails?.start_date);
  const cycleEndDate = getDate(cycleDetails?.end_date);
  const isCycleStartDateValid = cycleStartDate && cycleStartDate <= new Date();
  const isCycleEndDateValid = cycleStartDate && cycleEndDate && cycleEndDate >= cycleStartDate;
  const isCycleDateValid = isCycleStartDateValid && isCycleEndDateValid;

  if (!cycleDetails) return <></>;
  return (
    <div className="space-y-4 border-t border-subtle py-5">
      <Disclosure defaultOpen>
        {({ open }) => (
          <div className="flex flex-col">
            {/* progress bar header */}
            {isCycleDateValid ? (
              <div className="relative flex w-full items-center justify-between gap-2">
                <Disclosure.Button className="relative flex w-full items-center gap-2">
                  <div className="text-13 font-medium text-secondary">{t("project_cycles.active_cycle.progress")}</div>
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
                <div className="text-13 font-medium text-secondary">{t("project_cycles.active_cycle.progress")}</div>
              </div>
            )}
            <Transition as="div" show={open}>
              <Disclosure.Panel className="flex flex-col divide-y divide-subtle-1">
                {cycleStartDate && cycleEndDate ? (
                  <>
                    {isCycleDateValid && <SidebarChart cycleId={cycleId} />}
                    {/* progress detailed view */}
                    {chartDistributionData && (
                      <div className="w-full py-4">
                        <CycleProgressStats
                          cycleId={cycleId}
                          distribution={chartDistributionData}
                          groupedIssues={groupedIssues}
                          handleFiltersUpdate={updateFilterValueFromSidebar.bind(
                            updateFilterValueFromSidebar,
                            EIssuesStoreType.CYCLE,
                            cycleId
                          )}
                          isEditable={Boolean(!peekCycle) && cycleFilter !== undefined}
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
                  </>
                ) : (
                  <div className="my-2 w-full rounded-md bg-surface-2 px-2 py-2 text-13 text-tertiary">
                    {t("no_data_yet")}
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
