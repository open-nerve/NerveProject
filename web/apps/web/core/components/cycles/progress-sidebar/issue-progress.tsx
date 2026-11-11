/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Disclosure } from "@headlessui/react";
// nerve imports
import { ChevronDownOutline, ChevronUpOutline } from "@makeplane/propel/icons";
import type { Cycle, Progress } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Loader } from "@nerve/ui";
// components
import { BurndownChart } from "@/components/core/sidebar/burndown-chart";
// local imports
import { CycleProgressStats } from "./progress-stats";

type Props = {
  cycle: Cycle;
  projectId: string;
  /** The progress the sidebar shows (cycleCounts' choice); undefined until nerve has read it. */
  progress: Progress | undefined;
};

/**
 * The progress of a cycle's sidebar (P5a spec 2.5): none yet for a draft, which has no days; else its burndown, when
 * nerve gives one, and its breakdowns by state, assignee and label.
 */
export const CycleProgress = observer(function CycleProgress(props: Props) {
  const { cycle, projectId, progress } = props;
  // nerve hooks
  const { t } = useTranslation();

  return (
    <div className="space-y-4 border-t border-subtle py-5">
      <Disclosure defaultOpen>
        {({ open }) => (
          <div className="flex flex-col">
            <Disclosure.Button className="relative flex w-full items-center justify-between gap-2">
              <span className="text-13 font-medium text-secondary">{t("project_cycles.active_cycle.progress")}</span>
              {open ? (
                <ChevronUpOutline className="h-3.5 w-3.5" aria-hidden="true" />
              ) : (
                <ChevronDownOutline className="h-3.5 w-3.5" aria-hidden="true" />
              )}
            </Disclosure.Button>
            <Disclosure.Panel className="flex flex-col divide-y divide-subtle-1">
              {cycle.status === "draft" ? (
                <div className="my-2 w-full rounded-md bg-surface-2 px-2 py-2 text-13 text-tertiary">
                  {t("no_data_yet")}
                </div>
              ) : progress === undefined ? (
                <Loader className="mt-4 h-[160px] w-full">
                  <Loader.Item width="100%" height="100%" />
                </Loader>
              ) : (
                <>
                  {progress.burndown && (
                    <div className="py-4">
                      <BurndownChart burndown={progress.burndown} />
                    </div>
                  )}
                  <div className="w-full py-4">
                    <CycleProgressStats cycleId={cycle.id} projectId={projectId} progress={progress} />
                  </div>
                </>
              )}
            </Disclosure.Panel>
          </div>
        )}
      </Disclosure>
    </div>
  );
});
