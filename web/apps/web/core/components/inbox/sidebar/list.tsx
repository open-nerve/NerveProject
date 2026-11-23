/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useCallback, useRef, useState } from "react";
import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateDetailed } from "@nerve/propel/empty-state";
import { Loader } from "@nerve/ui";
// components
import { PlanningReadFailure } from "@/components/planning/read-failure";
import { InboxSidebarLoader } from "@/components/ui/loader/layouts/project-inbox/inbox-sidebar-loader";
// hooks
import { useIntake } from "@/hooks/store/use-intake";
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";
// store
import { customized } from "@/store/issue/intake/filters";
// local imports
import { useIntakePageState } from "../page-state";
import type { IntakeList as IntakeListRead } from "../use-intake-list";
import { IntakeListItem } from "./list-item";

/**
 * The empty state of each tab, and of filters of the tab's own that select no work item: its picture, and its title's
 * and its description's keys, each whole (tools/i18n-keys.mjs reads them).
 */
const EMPTY = {
  open: {
    asset: "inbox",
    title: "project_empty_state.intake_sidebar.title",
    description: "project_empty_state.intake_sidebar.description",
  },
  closed: {
    asset: "inbox",
    title: "project_empty_state.intake_closed.title",
    description: "project_empty_state.intake_closed.description",
  },
  filtered: {
    asset: "search",
    title: "common_empty_state.search.title",
    description: "common_empty_state.search.description",
  },
} as const;

type Props = {
  workspaceSlug: string;
  projectId: string;
  /** The list of the tab shown, and its read (useIntakeList). */
  list: IntakeListRead;
  /** The work item the page shows. */
  selectedId: string | undefined;
  /** The user pressed a row. */
  onOpen: () => void;
};

/**
 * The list of the intake page's tab (P5a spec 2.5, 2.7, R6): the project's work items its filters select, as nerve
 * gives them 50 at a time. Until nerve answers, a wait (InboxSidebarLoader); why nerve could not read it, with a retry
 * (never a wait that does not end); with no work item, the tab's own empty state, or that none matches filters of the
 * tab's own; else a row for each (IntakeListItem). As its end comes into view, it reads the next page by nerve's
 * cursor, while nerve gives one and no read of it is out (fetchMore, which adds the page only if the list still ends
 * there), a toast saying why nerve could not; a page of no rows with a cursor reads on, its end in view still.
 */
export const IntakeList = observer(function IntakeList(props: Props) {
  const { workspaceSlug, projectId, list, selectedId, onOpen } = props;
  const { t } = useTranslation();
  const { filters } = useIntakePageState();
  const { fetchMore } = useIntake();
  const refusalToast = useRefusalToast();
  const container = useRef<HTMLDivElement>(null);
  const [end, setEnd] = useState<HTMLDivElement | null>(null);
  // whether a next page's read is out: one at a time
  const reading = useRef(false);
  const { page, read } = list;
  const cursor = page?.next_cursor;
  const readMore = useCallback(() => {
    if (!cursor || reading.current) return;
    reading.current = true;
    void followInSession(() => fetchMore(projectId, filters), { failed: refusalToast }).finally(() => {
      reading.current = false;
    });
  }, [cursor, fetchMore, projectId, filters, refusalToast]);
  useIntersectionObserver(container, end, readMore, "20%");

  if (read.error !== undefined) return <PlanningReadFailure error={read.error} onRetry={() => void read.mutate()} />;
  if (page === undefined) return <InboxSidebarLoader />;
  if (page.data.length === 0 && !cursor) {
    const empty = EMPTY[customized(filters) ? "filtered" : filters.tab];
    return (
      <div className="flex h-full w-full items-center justify-center">
        <EmptyStateDetailed
          assetKey={empty.asset}
          title={t(empty.title)}
          description={t(empty.description)}
          assetClassName="size-20"
          rootClassName="px-page-x"
        />
      </div>
    );
  }
  return (
    <div ref={container} className="vertical-scrollbar scrollbar-md h-full w-full overflow-hidden overflow-y-auto">
      <ul>
        {page.data.map((row) => (
          <li key={row.id}>
            <IntakeListItem
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              row={row}
              selected={row.id === selectedId}
              onOpen={onOpen}
            />
          </li>
        ))}
      </ul>
      <div ref={setEnd}>
        {cursor && (
          <Loader className="mx-auto w-full space-y-4 px-2 py-4">
            <Loader.Item height="64px" width="w-100" />
            <Loader.Item height="64px" width="w-100" />
          </Loader>
        )}
      </div>
    </div>
  );
});
