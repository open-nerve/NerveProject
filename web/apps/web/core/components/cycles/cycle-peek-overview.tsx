/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
import { ChevronRightOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// local imports
import { CycleDetailsSidebar } from "./progress-sidebar";
import { CyclesReadFailure } from "./read-failure";

const PEEK = "peekCycle";

/**
 * The cycle the page previews (?peekCycle), if any, and how to preview another, or none: the address keeps the rest of
 * its query.
 */
export function usePeekCycle(): [string | null, (cycleId: string | null) => void] {
  const [searchParams, setSearchParams] = useSearchParams();
  const peek = (cycleId: string | null) =>
    setSearchParams((params) => {
      const next = new URLSearchParams(params);
      if (cycleId === null) next.delete(PEEK);
      else next.set(PEEK, cycleId);
      return next;
    });
  return [searchParams.get(PEEK), peek];
}

type Props = {
  projectId: string;
  workspaceSlug: string;
};

/**
 * The preview of a cycle on its project's list and archive (P5a spec 2.5, R11): the cycle ?peekCycle names, in a panel
 * at the page's right, its progress sidebar. It reads the cycle and its progress by the session's store, the reads of
 * a cycle's own page (an archived cycle's alike: nerve reads it), and says why nerve could not read either, with a
 * retry.
 */
export const CyclePeekOverview = observer(function CyclePeekOverview(props: Props) {
  const { projectId, workspaceSlug } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { fetchCycle, fetchProgress } = useCycles();
  const [peekCycle, peek] = usePeekCycle();
  const read = useSessionSWR(peekCycle ? ["CYCLE", peekCycle] : null, (id) => fetchCycle(id));
  const progress = useSessionSWR(peekCycle ? ["CYCLE_PROGRESS", peekCycle] : null, (id) => fetchProgress(id));

  if (!peekCycle) return null;

  const failed = [read, progress].filter((fetch) => fetch.error !== undefined);
  const handleClose = () => peek(null);
  return (
    <div
      className="fixed right-0 z-[9] flex h-full w-full max-w-[21.5rem] flex-shrink-0 flex-col gap-3.5 overflow-y-auto border-l border-subtle bg-surface-1 px-4 duration-300 md:relative"
      style={{
        boxShadow:
          "0px 1px 4px 0px rgba(0, 0, 0, 0.06), 0px 2px 4px 0px rgba(16, 24, 40, 0.06), 0px 1px 8px -1px rgba(16, 24, 40, 0.06)",
      }}
    >
      {failed.length > 0 ? (
        <>
          {/* the sidebar's close, which the panel has without the sidebar */}
          <button
            type="button"
            aria-label={t("close")}
            className="mt-2 flex size-6 flex-shrink-0 items-center justify-center rounded-full bg-layer-3 hover:bg-layer-3-hover"
            onClick={handleClose}
          >
            <ChevronRightOutline className="size-4 text-secondary" aria-hidden="true" />
          </button>
          <CyclesReadFailure error={failed[0]?.error} onRetry={() => failed.forEach((fetch) => void fetch.mutate())} />
        </>
      ) : (
        // Keyed by the cycle previewed (v0 design 7.7), as a cycle's own page is: the preview of another mounts its
        // sidebar afresh, so nothing of the cycle before (days picked, a change of days still out, its progress' tab)
        // carries into it.
        <CycleDetailsSidebar
          key={peekCycle}
          handleClose={handleClose}
          projectId={projectId}
          workspaceSlug={workspaceSlug}
          cycleId={peekCycle}
        />
      )}
    </div>
  );
});
