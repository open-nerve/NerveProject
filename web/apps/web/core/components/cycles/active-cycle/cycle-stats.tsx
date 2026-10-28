/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
import { isEmpty } from "lodash-es";
import { observer } from "mobx-react";
import { useTheme } from "next-themes";
// headless ui
import { Tab } from "@headlessui/react";
// nerve imports
import { Avatar } from "@makeplane/propel/components/avatar";
import { useTranslation } from "@nerve/i18n";
import type { ICycle } from "@nerve/types";
// ui
import { Loader } from "@nerve/ui";
import { cn, getFileURL } from "@nerve/utils";
// assets
import darkAssigneeAsset from "@/app/assets/empty-state/active-cycle/assignee-dark.webp?url";
import lightAssigneeAsset from "@/app/assets/empty-state/active-cycle/assignee-light.webp?url";
import darkLabelAsset from "@/app/assets/empty-state/active-cycle/label-dark.webp?url";
import lightLabelAsset from "@/app/assets/empty-state/active-cycle/label-light.webp?url";
import userImage from "@/app/assets/user.png?url";
// components
import { SingleProgressStats } from "@/components/core/sidebar/single-progress-stats";
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";
// hooks
import useLocalStorage from "@/hooks/use-local-storage";
// store

export type ActiveCycleStatsProps = {
  workspaceSlug: string;
  projectId: string;
  cycle: ICycle | null;
  cycleId?: string | null;
};

export const ActiveCycleStats = observer(function ActiveCycleStats(props: ActiveCycleStatsProps) {
  const { cycle, cycleId } = props;
  // local storage
  const { storedValue: tab, setValue: setTab } = useLocalStorage("activeCycleTab", "Assignees");
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  // derived values
  const assigneesResolvedPath = resolvedTheme === "light" ? lightAssigneeAsset : darkAssigneeAsset;
  const labelsResolvedPath = resolvedTheme === "light" ? lightLabelAsset : darkLabelAsset;

  const currentValue = (tab: string | null) => {
    switch (tab) {
      case "Assignees":
        return 0;
      case "Labels":
        return 1;
      default:
        return 0;
    }
  };
  const loaders = (
    <Loader className="space-y-3">
      <Loader.Item height="30px" />
      <Loader.Item height="30px" />
      <Loader.Item height="30px" />
    </Loader>
  );

  return cycleId ? (
    <div className="col-span-1 flex min-h-[17rem] flex-col gap-4 overflow-hidden rounded-lg border border-subtle bg-surface-1 p-4 lg:col-span-2 xl:col-span-1">
      <Tab.Group
        as={Fragment}
        defaultIndex={currentValue(tab)}
        onChange={(i) => {
          switch (i) {
            case 0:
              return setTab("Assignees");
            case 1:
              return setTab("Labels");

            default:
              return setTab("Assignees");
          }
        }}
      >
        <Tab.List
          as="div"
          className="relative grid rounded-sm border-[0.5px] border-subtle bg-layer-1 p-[1px]"
          style={{
            gridTemplateColumns: `repeat(2, 1fr)`,
          }}
        >
          <Tab
            className={({ selected }) =>
              cn(
                "relative z-[1] rounded-[3px] py-1.5 text-11 font-semibold text-placeholder transition duration-500 focus:outline-none",
                {
                  "bg-surface-1 text-tertiary": selected,
                  "hover:text-tertiary": !selected,
                }
              )
            }
          >
            {t("project_cycles.active_cycle.assignees")}
          </Tab>
          <Tab
            className={({ selected }) =>
              cn(
                "relative z-[1] rounded-[3px] py-1.5 text-11 font-semibold text-placeholder transition duration-500 focus:outline-none",
                {
                  "bg-surface-1 text-tertiary": selected,
                  "hover:text-tertiary": !selected,
                }
              )
            }
          >
            {t("project_cycles.active_cycle.labels")}
          </Tab>
        </Tab.List>

        <Tab.Panels as={Fragment}>
          <Tab.Panel
            as="div"
            className="vertical-scrollbar scrollbar-sm flex h-52 w-full flex-col gap-1 overflow-y-auto text-secondary"
          >
            {cycle && !isEmpty(cycle.distribution) ? (
              cycle?.distribution?.assignees && cycle.distribution.assignees.length > 0 ? (
                cycle.distribution?.assignees?.map((assignee, index) => {
                  if (assignee.assignee_id)
                    return (
                      <SingleProgressStats
                        key={assignee.assignee_id}
                        title={
                          <div className="flex items-center gap-2">
                            <Avatar
                              alt={assignee?.display_name ?? undefined}
                              fallback={assignee?.display_name?.[0]?.toUpperCase()}
                              src={getFileURL(assignee?.avatar_url ?? "")}
                              size="xs"
                            />

                            <span>{assignee.display_name}</span>
                          </div>
                        }
                        completed={assignee.completed_issues}
                        total={assignee.total_issues}
                      />
                    );
                  else
                    return (
                      <SingleProgressStats
                        key={`unassigned-${index}`}
                        title={
                          <div className="flex items-center gap-2">
                            <div className="h-5 w-5 rounded-full border-2 border-subtle bg-layer-1">
                              <img src={userImage} height="100%" width="100%" className="rounded-full" alt="User" />
                            </div>
                            <span>{t("no_assignee")}</span>
                          </div>
                        }
                        completed={assignee.completed_issues}
                        total={assignee.total_issues}
                      />
                    );
                })
              ) : (
                <div className="flex h-full w-full items-center justify-center">
                  <SimpleEmptyState
                    title={t("active_cycle.empty_state.assignee.title")}
                    assetPath={assigneesResolvedPath}
                  />
                </div>
              )
            ) : (
              loaders
            )}
          </Tab.Panel>

          <Tab.Panel
            as="div"
            className="vertical-scrollbar scrollbar-sm flex h-52 w-full flex-col gap-1 overflow-y-auto text-secondary"
          >
            {cycle && !isEmpty(cycle.distribution) ? (
              cycle?.distribution?.labels && cycle.distribution.labels.length > 0 ? (
                cycle.distribution.labels?.map((label, index) => (
                  <SingleProgressStats
                    key={label.label_id ?? `no-label-${index}`}
                    title={
                      <div className="flex items-center gap-2 truncate">
                        <span
                          className="block h-3 w-3 flex-shrink-0 rounded-full"
                          style={{
                            backgroundColor: label.color ?? "#000000",
                          }}
                        />
                        <span className="truncate text-11 text-ellipsis">{label.label_name ?? "No labels"}</span>
                      </div>
                    }
                    completed={label.completed_issues}
                    total={label.total_issues}
                  />
                ))
              ) : (
                <div className="flex h-full w-full items-center justify-center">
                  <SimpleEmptyState title={t("active_cycle.empty_state.label.title")} assetPath={labelsResolvedPath} />
                </div>
              )
            ) : (
              loaders
            )}
          </Tab.Panel>
        </Tab.Panels>
      </Tab.Group>
    </div>
  ) : (
    <Loader className="col-span-1 flex min-h-[17rem] flex-col gap-4 overflow-hidden bg-surface-1 lg:col-span-2 xl:col-span-1">
      <Loader.Item width="100%" height="17rem" />
    </Loader>
  );
});
