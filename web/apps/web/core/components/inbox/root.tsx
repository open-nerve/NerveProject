/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useState } from "react";
import { observer } from "mobx-react";
import { useSearchParams } from "react-router";
import { LeftSidePaneOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { EmptyStateCompact } from "@nerve/propel/empty-state";
import { cn } from "@nerve/utils";
// components
import { InboxLayoutLoader } from "@/components/ui/loader/layouts/project-inbox/inbox-layout-loader";
// hooks
import useSize from "@/hooks/use-window-size";
// local imports
import { InboxContentRoot } from "./content";
import { useIntakeArrowKeys } from "./content/header/navigation";
import { useIntakeLanding } from "./landing";
import { useIntakePageState } from "./page-state";
import { IntakeSidebar } from "./sidebar/root";
import { useIntakeList } from "./use-intake-list";
import { useLeavingRow } from "./use-leaving-row";

/** The window's width from which the list stands beside the detail: Tailwind's lg (64rem), the layout's classes'. */
const BESIDE = 1024;

type TInboxIssueRoot = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The intake page's body (P5a spec 2.6, 2.7, R5, R14): the list of the tab the address names beside the detail of the
 * work item it selects (?inboxIssueId), the address's alone, so that Back and Forward go through the selections. A
 * row's press is the user's (a push); with nothing selected and the list read with rows, the page selects its first
 * row by a replace, its own doing, as after an action cleared the selection. The whole layout waits for the page's
 * first read (InboxLayoutLoader); a later one, of another tab or other filters, waits in the list alone, the tabs kept.
 * ↑ and ↓ step through the list (useIntakeArrowKeys). On a narrow screen, the list lies over the detail until a row is
 * pressed; with nothing selected, a named button shows or hides it (the detail's header has its own). The landing of
 * the focus as an action moves the selection (useIntakeLanding) is told whether the list is on the screen, and the
 * list's first row, so that the focus never lands on what the screen does not show (P5a R35); a row that has the focus
 * and leaves the list gives it to the next row, else the previous, else the list (useLeavingRow).
 */
export const InboxIssueRoot = observer(function InboxIssueRoot(props: TInboxIssueRoot) {
  const { workspaceSlug, projectId } = props;
  // states
  const [isMobileSidebar, setIsMobileSidebar] = useState(true);
  // whether the page's first read has answered, after which the list waits for its reads alone
  const [opened, setOpened] = useState(false);
  // nerve hooks
  const { t } = useTranslation();
  // router
  const [params, setParams] = useSearchParams();
  const selectedId = params.get("inboxIssueId") ?? undefined;
  // hooks
  const { filters } = useIntakePageState();
  const list = useIntakeList(projectId, filters);
  useIntakeArrowKeys(projectId, selectedId);
  const [width] = useSize();
  // derived values
  const answered = list.page !== undefined || list.read.error !== undefined;
  const first = list.page?.data[0]?.id;
  const landing = useIntakeLanding({ list: isMobileSidebar || width >= BESIDE, first });
  const leaving = useLeavingRow(list.page?.data.map((row) => row.id) ?? [], landing.land);

  useEffect(() => {
    if (answered) setOpened(true);
  }, [answered]);

  useEffect(() => {
    if (selectedId !== undefined || first === undefined) return;
    setParams(
      (current) => {
        const next = new URLSearchParams(current);
        next.set("currentTab", filters.tab);
        next.set("inboxIssueId", first);
        return next;
      },
      { replace: true }
    );
  }, [selectedId, first, filters.tab, setParams]);

  if (!opened && !answered)
    return (
      <div className="relative flex h-full w-full flex-col">
        <InboxLayoutLoader />
      </div>
    );

  return (
    <>
      {!selectedId && (
        <div className="flex h-12 w-full items-center border-b border-subtle px-4 lg:hidden">
          <button
            ref={landing.toggle}
            type="button"
            aria-label={t("inbox_issue.a11y.list_toggle")}
            aria-expanded={isMobileSidebar}
            onClick={() => setIsMobileSidebar(!isMobileSidebar)}
          >
            <LeftSidePaneOutline
              className={cn("h-4 w-4", isMobileSidebar ? "text-accent-primary" : "text-secondary")}
            />
          </button>
        </div>
      )}
      <div className="flex h-full w-full overflow-hidden bg-surface-1">
        <div
          onFocus={leaving.onFocus}
          onBlur={leaving.onBlur}
          className={cn(
            "absolute top-[50px] bottom-0 z-10 w-full flex-shrink-0 bg-surface-1 transition-all lg:!relative lg:!top-0 lg:w-2/6",
            // translated only while hidden on a narrow screen: an element translated at all, even by 0, is the
            // containing block of its fixed descendants, and the popovers of the list's filters and order, fixed and
            // placed by the viewport, would open away from their buttons
            isMobileSidebar ? "translate-none" : "-translate-x-full lg:!translate-none"
          )}
        >
          <IntakeSidebar
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            list={list}
            listRef={landing.list}
            selectedId={selectedId}
            onOpen={() => setIsMobileSidebar(false)}
          />
        </div>

        {selectedId ? (
          <InboxContentRoot
            setIsMobileSidebar={setIsMobileSidebar}
            isMobileSidebar={isMobileSidebar}
            workspaceSlug={workspaceSlug}
            projectId={projectId}
            inboxIssueId={selectedId}
            onLeave={landing.leave}
            region={landing.detail}
          />
        ) : (
          <EmptyStateCompact
            assetKey="intake"
            title={t("project_empty_state.intake_main.title")}
            assetClassName="size-20"
          />
        )}
      </div>
    </>
  );
});
