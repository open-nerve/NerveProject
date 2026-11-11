/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Disclosure } from "@headlessui/react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { Loader, Row } from "@nerve/ui";
import { cycleCounts } from "@nerve/utils";
// components
import { CycleListGroupHeader } from "@/components/cycles/list/cycle-list-group-header";
import { CyclesListItem } from "@/components/cycles/list/cycles-list-item";
import { CyclesReadFailure } from "@/components/cycles/read-failure";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// local imports
import { ActiveCycleStats } from "./cycle-stats";
import { ActiveCycleProductivity } from "./productivity";
import { ActiveCycleProgress } from "./progress";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The card of the project's cycle in progress on its cycles list (P5a spec 2.5), of the route's project, open by
 * default: it waits for the project's cycles; says that none is in progress; or shows the cycle in progress
 * (ActiveCycle).
 */
export const ActiveCycleRoot = observer(function ActiveCycleRoot(props: Props) {
  const { workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getActiveCycleId } = useCycles();
  // derived values
  const cycleId = getActiveCycleId(projectId);
  const title = t("project_cycles.active_cycle.label");

  return (
    <Disclosure as="section" aria-label={title} className="flex flex-shrink-0 flex-col" defaultOpen>
      {({ open }) => (
        <>
          <Disclosure.Button className="sticky top-0 z-[2] w-full flex-shrink-0 cursor-pointer border-b border-subtle bg-layer-1">
            <CycleListGroupHeader title={title} type="current" isExpanded={open} />
          </Disclosure.Button>
          <Disclosure.Panel>
            {cycleId === undefined ? (
              <Loader className="p-5">
                <Loader.Item height="17rem" />
              </Loader>
            ) : cycleId === null ? (
              <EmptyStateDetailed
                assetKey="cycle"
                title={t("project_cycles.empty_state.active.title")}
                description={t("project_cycles.empty_state.active.description")}
                rootClassName="py-10 h-auto"
              />
            ) : (
              <ActiveCycle key={cycleId} workspaceSlug={workspaceSlug} projectId={projectId} cycleId={cycleId} />
            )}
          </Disclosure.Panel>
        </>
      )}
    </Disclosure>
  );
});

type ActiveCycleProps = Props & { cycleId: string };

/**
 * The cycle in progress: its row, then its progress, its burndown and its breakdowns. It reads the cycle's progress by
 * the key the cycle's page and its preview read it by (SWR shares the one read); the list page shows the card to the
 * project's members alone, of a project not archived, as it reads the project's cycles. The cards show nerve's
 * progress of one choice (cycleCounts, R7): a cycle in progress has its live one. It waits for nerve's answer, or says
 * why nerve could not give it, with a retry.
 */
const ActiveCycle = observer(function ActiveCycle(props: ActiveCycleProps) {
  const { workspaceSlug, projectId, cycleId } = props;
  // store hooks
  const { getCycleProgress, fetchProgress } = useCycles();
  const read = useSessionSWR(["CYCLE_PROGRESS", cycleId], (id) => fetchProgress(id));
  // derived values
  const progress = getCycleProgress(cycleId);
  const shown = progress && cycleCounts(progress.status, progress.live, progress.progress_snapshot);

  return (
    <div className="flex flex-col border-b border-subtle">
      <CyclesListItem
        cycleId={cycleId}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        className="!border-b-transparent"
      />
      <Row className="bg-surface-1 pt-3 pb-6">
        {read.error !== undefined ? (
          <CyclesReadFailure error={read.error} onRetry={() => void read.mutate()} />
        ) : (
          <div className="grid grid-cols-1 gap-3 bg-surface-1 lg:grid-cols-2 xl:grid-cols-3">
            {shown === undefined ? (
              <>
                <CardLoader />
                <CardLoader />
                <CardLoader />
              </>
            ) : (
              <>
                <ActiveCycleProgress progress={shown} />
                <ActiveCycleProductivity
                  workspaceSlug={workspaceSlug}
                  projectId={projectId}
                  cycleId={cycleId}
                  progress={shown}
                />
                <ActiveCycleStats projectId={projectId} progress={shown} />
              </>
            )}
          </div>
        )}
      </Row>
    </div>
  );
});

/** A card waiting for the cycle's progress. */
function CardLoader() {
  return (
    <Loader className="flex min-h-[17rem] flex-col gap-5 rounded-lg border border-subtle bg-surface-1">
      <Loader.Item width="100%" height="100%" />
    </Loader>
  );
}
