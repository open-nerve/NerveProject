/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useRef, useState } from "react";
import { observer } from "mobx-react";
import { useTheme } from "next-themes";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { CompletedAtOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { PriorityIcon, StateGroupIcon } from "@nerve/propel/icons";
import { Loader } from "@nerve/ui";
import { renderFormattedDate, renderFormattedDateWithoutYear } from "@nerve/utils";
// assets
import darkPriorityAsset from "@/app/assets/empty-state/active-cycle/priority-dark.webp?url";
import lightPriorityAsset from "@/app/assets/empty-state/active-cycle/priority-light.webp?url";
// components
import { CyclesReadFailure } from "@/components/cycles/read-failure";
import { SimpleEmptyState } from "@/components/empty-state/simple-empty-state-root";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useCyclePriorityIssues } from "@/hooks/store/use-cycle-priority-issues";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useUserPermissions } from "@/hooks/store/user";
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

type Props = {
  workspaceSlug: string;
  projectId: string;
  /** The cycle in progress the card shows. */
  cycleId: string;
};

/**
 * The tab of the card of the cycle in progress that lists its urgent and high work items (P5b spec 2.7, R7), the
 * newest first, as nerve gives them: by their own list (useCyclePriorityIssues), never the cycle page's, read by the
 * card's own gates (a member's read, of a project not archived whose cycles are on) once the tab is open. It waits for
 * nerve's answer, or says why nerve could not give it, with a retry; says how to see one when there is none; scrolled
 * to its end, it reads the next page by its cursor, and says why one failed, with a retry. A row is a plain item.
 */
export const ActiveCyclePriorityIssues = observer(function ActiveCyclePriorityIssues(props: Props) {
  const { workspaceSlug, projectId, cycleId } = props;
  // refs
  const containerRef = useRef<HTMLDivElement | null>(null);
  // states
  const [loaderElement, setLoaderElement] = useState<HTMLDivElement | null>(null);
  // theme hook
  const { resolvedTheme } = useTheme();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const store = useCyclePriorityIssues();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  // a member's read, of a project not archived whose cycles are on: as the cycles list reads the project's cycles
  const project = getProjectById(projectId);
  const listed =
    getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) !== undefined &&
    project?.archived_at === null &&
    project.cycle_view;
  const page = { projectID: projectId, cycleID: cycleId };
  const read = useSessionSWR(
    listed
      ? ["CYCLE_PRIORITY_ISSUES", workspaceSlug, projectId, cycleId, store.readIdentity(workspaceSlug, page)]
      : null,
    (slug, projectID, cycleID) => store.fetchIssues(slug, { projectID, cycleID }),
    { dedupingInterval: 0 }
  );
  const shown = store.view(workspaceSlug, page);
  const loadMore = useCallback(() => {
    void store.fetchNextIssues(workspaceSlug, { projectID: projectId, cycleID: cycleId }).catch(() => {});
  }, [store, workspaceSlug, projectId, cycleId]);
  useIntersectionObserver(containerRef, loaderElement, loadMore, "0% 0% 100% 0%");

  if (shown === undefined)
    return read.error !== undefined ? (
      <CyclesReadFailure error={read.error} onRetry={() => void read.mutate()} />
    ) : (
      <Loader className="space-y-3">
        <Loader.Item height="30px" />
        <Loader.Item height="30px" />
        <Loader.Item height="30px" />
      </Loader>
    );

  if (shown.total === 0)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <SimpleEmptyState
          title={t("active_cycle.empty_state.priority_issue.title")}
          assetPath={resolvedTheme === "light" ? lightPriorityAsset : darkPriorityAsset}
        />
      </div>
    );

  return (
    <div
      ref={containerRef}
      className="vertical-scrollbar scrollbar-sm flex h-full w-full flex-col gap-1 overflow-y-auto"
    >
      <ul className="flex flex-col gap-1">
        {shown.issueIds.map((issueId) => (
          <PriorityIssueRow key={issueId} issueId={issueId} projectId={projectId} />
        ))}
      </ul>
      {shown.error !== undefined ? (
        <CyclesReadFailure
          error={shown.error}
          onRetry={() =>
            void store.retryIssues(workspaceSlug, { projectID: projectId, cycleID: cycleId }).catch(() => {})
          }
        />
      ) : (
        shown.more && (
          <div
            ref={setLoaderElement}
            className="relative flex h-11 animate-pulse items-center gap-3 bg-layer-1 p-3 text-13"
          />
        )
      )}
    </div>
  );
});

/** A work item of the tab: its identifier, name, priority, state (read-only) and target date. */
const PriorityIssueRow = observer(function PriorityIssueRow(props: { issueId: string; projectId: string }) {
  const { issueId, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const { getStateById } = useProjectState();
  // derived values
  const issue = getIssueById(issueId);
  const state = getStateById(issue?.state_id);

  if (!issue) return null;

  return (
    <li className="group flex items-center justify-between gap-2 rounded-md p-1 hover:bg-surface-2">
      <div className="flex w-full min-w-24 flex-grow items-center gap-1.5 truncate">
        <IssueIdentifier issueId={issue.id} projectId={projectId} size="xs" variant="secondary" />
        <Tooltip label={issue.name} layout="stacked" align="start">
          <span className="truncate text-13 text-primary">{issue.name}</span>
        </Tooltip>
      </div>
      <PriorityIcon priority={issue.priority} withContainer size={12} />
      <span className="sr-only">{`${t("priority")}: ${t(issue.priority)}`}</span>
      <div className="flex flex-shrink-0 items-center gap-1.5">
        <span className="flex max-w-24 items-center gap-1 truncate rounded-sm bg-layer-1 px-2 py-0.5 text-11">
          <StateGroupIcon
            stateGroup={state?.group ?? "backlog"}
            color={state?.color ?? "var(--text-color-tertiary)"}
            className="size-3 flex-shrink-0"
          />
          <span className="truncate">{state?.name}</span>
        </span>
        {issue.target_date && (
          <Tooltip label={`${t("target_date")}: ${renderFormattedDate(issue.target_date) ?? ""}`}>
            <span className="flex h-full items-center gap-1.5 truncate rounded-sm bg-layer-1 px-2 py-0.5 text-11 group-hover:bg-surface-1">
              <CompletedAtOutline className="h-3 w-3 flex-shrink-0" />
              <span className="truncate text-11">{renderFormattedDateWithoutYear(issue.target_date)}</span>
            </span>
          </Tooltip>
        )}
      </div>
    </li>
  );
});
