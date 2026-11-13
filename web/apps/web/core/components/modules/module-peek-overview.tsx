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
// components
import { PlanningReadFailure } from "@/components/planning/read-failure";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { usePlanningReads } from "@/hooks/use-planning-reads";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";
// local imports
import { ModuleDetailsSidebar } from "./progress-sidebar";

const PEEK = "peekModule";

/**
 * The module the page previews (?peekModule), if any, and how to preview another, or none: the address keeps the rest
 * of its query.
 */
export function usePeekModule(): [string | null, (moduleId: string | null) => void] {
  const [searchParams, setSearchParams] = useSearchParams();
  const peek = (moduleId: string | null) =>
    setSearchParams((params) => {
      const next = new URLSearchParams(params);
      if (moduleId === null) next.delete(PEEK);
      else next.set(PEEK, moduleId);
      return next;
    });
  return [searchParams.get(PEEK), peek];
}

type Props = {
  projectId: string;
  workspaceSlug: string;
};

/**
 * The preview of a module on its project's list and archive (P6a spec 2.6, R3, R12): the module ?peekModule names, in
 * a panel at the page's right, its progress sidebar. It reads the module, its progress and its links by the session's
 * store, the reads of a module's own page, for the caller who reads the project's modules (usePlanningReads'
 * readable: an archived module's alike, nerve reads it, and whether the modules are turned on or not), and says why
 * nerve could not read one of them, with a retry. Closing it takes it out of the address.
 */
export const ModulePeekOverview = observer(function ModulePeekOverview({ projectId, workspaceSlug }: Props) {
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { fetchModule, fetchProgress, fetchLinks } = useModules();
  const [peekModule, peek] = usePeekModule();
  const { readable } = usePlanningReads(workspaceSlug, projectId, "modules");
  const previewed = readable ? peekModule : null;
  const read = useSessionSWR(previewed ? ["MODULE", previewed] : null, (id) => fetchModule(id));
  const progress = useSessionSWR(previewed ? ["MODULE_PROGRESS", previewed] : null, (id) => fetchProgress(id));
  const links = useSessionSWR(previewed ? ["MODULE_LINKS", previewed] : null, (id) => fetchLinks(id));

  if (!previewed) return null;

  const failed = [read, progress, links].filter((fetch) => fetch.error !== undefined);
  const handleClose = () => peek(null);
  return (
    <div
      className="absolute right-0 z-[9] flex h-full w-full max-w-[24rem] flex-shrink-0 flex-col gap-3.5 overflow-y-auto border-l border-subtle bg-surface-1 px-6 duration-300 md:relative"
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
            className="mt-5 flex size-5 flex-shrink-0 items-center justify-center rounded-full bg-layer-3 hover:bg-layer-3-hover"
            onClick={handleClose}
          >
            <ChevronRightOutline className="size-3 text-secondary" aria-hidden="true" />
          </button>
          <PlanningReadFailure
            error={failed[0]?.error}
            onRetry={() => failed.forEach((fetch) => void fetch.mutate())}
          />
        </>
      ) : (
        // Keyed by the module previewed (v0 design 7.7), as a module's own page is: the preview of another mounts its
        // sidebar afresh, so nothing of the module before (days picked, a change still out, its progress' tab)
        // carries into it.
        <ModuleDetailsSidebar
          key={previewed}
          moduleId={previewed}
          projectId={projectId}
          workspaceSlug={workspaceSlug}
          handleClose={handleClose}
        />
      )}
    </div>
  );
});
