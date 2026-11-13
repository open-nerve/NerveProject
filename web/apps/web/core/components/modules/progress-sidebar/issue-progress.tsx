/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Disclosure } from "@headlessui/react";
// nerve imports
import { ChevronDownOutline, ChevronUpOutline } from "@makeplane/propel/icons";
import type { Module, Progress } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Loader } from "@nerve/ui";
import { progressPercent } from "@nerve/utils";
// components
import { BurndownChart } from "@/components/core/sidebar/burndown-chart";
import { ProgressTabs } from "@/components/core/sidebar/progress-tabs";

type Props = {
  module: Module;
  projectId: string;
  /** nerve's progress of the module; undefined until nerve has read it. */
  progress: Progress | undefined;
};

/**
 * The progress of a module's sidebar (P6a spec 2.6, R8): how far it has come (progressPercent); its burndown, from its
 * start to its target day, when nerve gives one (none without either day, or over too long a span: no data yet); and,
 * either way, its breakdowns by state, assignee and label (ProgressTabs), in plain rows, the tab chosen kept for the
 * module.
 */
export const ModuleProgress = observer(function ModuleProgress(props: Props) {
  const { module, projectId, progress } = props;
  // nerve hooks
  const { t } = useTranslation();
  const percent = progress && progressPercent(progress);

  return (
    <div className="space-y-4 border-t border-subtle py-5">
      <Disclosure defaultOpen>
        {({ open }) => (
          <div className="flex flex-col">
            <Disclosure.Button className="relative flex w-full items-center gap-2">
              <span className="text-13 font-medium text-secondary">{t("common.progress")}</span>
              {percent !== undefined && percent > 0 && (
                <span className="bg-amber-500/20 text-amber-500 flex h-5 w-9 items-center justify-center rounded-sm text-11 font-medium">
                  {`${percent}%`}
                </span>
              )}
              {open ? (
                <ChevronUpOutline className="ml-auto h-3.5 w-3.5" aria-hidden="true" />
              ) : (
                <ChevronDownOutline className="ml-auto h-3.5 w-3.5" aria-hidden="true" />
              )}
            </Disclosure.Button>
            <Disclosure.Panel className="flex flex-col divide-y divide-subtle-1">
              {progress === undefined ? (
                <Loader className="mt-4 h-[160px] w-full">
                  <Loader.Item width="100%" height="100%" />
                </Loader>
              ) : (
                <>
                  {progress.burndown ? (
                    <div className="py-4">
                      <BurndownChart burndown={progress.burndown} />
                    </div>
                  ) : (
                    <div className="my-2 w-full rounded-md bg-surface-2 px-2 py-2 text-13 text-tertiary">
                      {t("no_data_yet")}
                    </div>
                  )}
                  <div className="w-full py-4">
                    <ProgressTabs
                      progress={progress}
                      projectId={projectId}
                      storageKey={`module-progress-tab-${module.id}`}
                    />
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
