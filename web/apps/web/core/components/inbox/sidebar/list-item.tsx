/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useId } from "react";
import { observer } from "mobx-react";
import { Link } from "react-router";
// nerve imports
import type { IntakeIssue } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { PriorityIcon } from "@nerve/propel/icons";
import { Row } from "@nerve/ui";
import { cn, renderFormattedDate } from "@nerve/utils";
// components
import { ButtonAvatars } from "@/components/dropdowns/member/avatar";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useProject } from "@/hooks/store/use-project";
// local imports
import { IntakeStatusChip } from "../content/header/status-chip";
import { useIntakePageState } from "../page-state";

/** A row shows each of its labels up to this many; past it, how many it has. */
const SHOWN_LABELS = 3;

type Props = {
  workspaceSlug: string;
  projectId: string;
  /** The work item as nerve listed it. */
  row: IntakeIssue;
  /** Whether it is the work item the page shows. */
  selected: boolean;
  /** The user pressed it (the narrow page then hides the list). */
  onOpen: () => void;
};

/**
 * A row of the intake page's list (P5a spec 2.7): a link to the work item's address in the tab shown, the user's own
 * move (a push), named by its identifier, in the route's project, and its name; the one the page shows is current, and
 * pressing it again goes nowhere. It shows its identifier, its status by its name (a snooze the days it has to go,
 * IntakeStatusChip), which describes the link too, its name, when it was submitted, its priority, its labels (how many,
 * past three) and who submitted it, all as nerve listed it (IntakeIssue).
 */
export const IntakeListItem = observer(function IntakeListItem(props: Props) {
  const { workspaceSlug, projectId, row, selected, onOpen } = props;
  const { t } = useTranslation();
  const { filters } = useIntakePageState();
  const { getProjectById } = useProject();
  const { getProjectLabels } = useLabel();
  const status = useId();
  // derived values
  const identifier = `${getProjectById(projectId)?.identifier ?? ""}-${row.sequence_id}`;
  const labels = getProjectLabels(projectId)?.filter((label) => row.label_ids.includes(label.id)) ?? [];
  return (
    <Link
      to={`/${workspaceSlug}/projects/${projectId}/intake?currentTab=${filters.tab}&inboxIssueId=${row.id}`}
      aria-label={`${identifier} ${row.name}`}
      aria-describedby={status}
      aria-current={selected ? "true" : undefined}
      onClick={(event) => {
        if (selected) event.preventDefault();
        onOpen();
      }}
      className="block focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong"
    >
      <Row
        className={cn(
          "relative flex flex-col gap-2 border border-t-transparent border-r-transparent border-b-subtle-1 border-l-transparent py-4 transition-all hover:bg-accent-primary/5",
          { "border-accent-strong": selected }
        )}
      >
        <div className="space-y-1">
          <div className="relative flex items-center justify-between gap-2">
            <div className="flex-shrink-0 text-11 font-medium text-tertiary">{identifier}</div>
            <div id={status}>
              <IntakeStatusChip intake={row.intake} />
            </div>
          </div>
          <h3 className="w-full truncate text-13">{row.name}</h3>
        </div>
        <div className="flex items-center justify-between">
          <div className="flex flex-wrap items-center gap-2">
            <div className="text-11 text-secondary">{renderFormattedDate(row.created_at)}</div>
            <div className="rounded-full border-2 border-strong-1" />
            <PriorityIcon priority={row.priority} withContainer className="h-3 w-3" />
            {labels.length > SHOWN_LABELS ? (
              <div className="relative flex !h-[17.5px] items-center gap-1 rounded-sm border border-strong px-1 text-11">
                <span className="bg-orange-400 h-2 w-2 rounded-full" />
                <span className="max-w-28 truncate normal-case">
                  {t("inbox_issue.list.labels", { count: labels.length })}
                </span>
              </div>
            ) : (
              labels.map((label) => (
                <div
                  key={label.id}
                  className="relative flex !h-[17.5px] items-center gap-1 rounded-sm border border-strong px-1 text-11"
                >
                  <span className="h-2 w-2 rounded-full" style={{ backgroundColor: label.color }} />
                  <span className="max-w-28 truncate normal-case">{label.name}</span>
                </div>
              ))
            )}
          </div>
          {row.created_by_id !== null && <ButtonAvatars userIds={row.created_by_id} />}
        </div>
      </Row>
    </Link>
  );
});
