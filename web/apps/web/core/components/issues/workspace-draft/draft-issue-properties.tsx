/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Draft } from "@nerve/api-client";
import { ISSUE_PRIORITIES } from "@nerve/constants";
import type { DraftEdits } from "@/services/issue/draft-input";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { useDetailScope } from "../issue-detail-widgets/use-detail-scope";
import { observer } from "mobx-react";
// icons
import { DueDateOutline, StartDateOutline } from "@makeplane/propel/icons";
// types
import { getDate, renderFormattedPayloadDate, shouldHighlightIssueDueDate } from "@nerve/utils";
// components
import { DateDropdown } from "@/components/dropdowns/date";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
// hooks
import { useLabel } from "@/hooks/store/use-label";
import { useProjectState } from "@/hooks/store/use-project-state";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { IssuePropertyLabels } from "../issue-layouts/properties";

type Props = {
  issue: Draft;
  workspaceSlug: string;
  updateIssue: (edits: DraftEdits) => Promise<void>;
  className: string;
};
export const DraftIssueProperties = observer(function DraftIssueProperties({
  issue: draft,
  workspaceSlug,
  updateIssue,
  className,
}: Props) {
  const issue = draft.payload;
  const { labelMap } = useLabel();
  const { getStateById } = useProjectState();
  const { isMobile } = usePlatformOS();
  const scope = useDetailScope(workspaceSlug, undefined, draft.id, true);
  const refuse = useRefusalToast();
  const stateDetails = issue.state_id ? getStateById(issue.state_id) : undefined;
  const change = async (edits: DraftEdits) => {
    const current = scope();
    if (!current()) return;
    try {
      await updateIssue(edits);
    } catch (error) {
      if (current()) refuse(error);
    }
  };
  const handleState = (state_id: string) => change({ state_id });
  const handlePriority = (priority: string) => change({ priority });
  const handleLabel = (label_ids: string[]) => change({ label_ids });
  const handleAssignee = (assignee_ids: string[]) => change({ assignee_ids });
  const handleStartDate = (date: Date | null) => change({ start_date: date ? renderFormattedPayloadDate(date) : null });
  const handleTargetDate = (date: Date | null) =>
    change({ target_date: date ? renderFormattedPayloadDate(date) : null });

  const defaultLabelOptions =
    issue?.label_ids?.flatMap((id) => {
      const label = labelMap[id];
      return label ? [label] : [];
    }) || [];

  const minDate = getDate(issue.start_date);
  minDate?.setDate(minDate.getDate());

  const maxDate = getDate(issue.target_date);
  maxDate?.setDate(maxDate.getDate());

  return (
    <div className={className} data-draft-properties>
      {/* basic properties */}
      {/* state */}
      {draft.project_id && (
        <div className="h-5">
          <StateDropdown
            buttonContainerClassName="truncate max-w-40"
            value={issue.state_id}
            onChange={handleState}
            projectId={draft.project_id ?? undefined}
            buttonVariant="border-with-text"
            renderByDefault={isMobile}
            showTooltip
          />
        </div>
      )}

      {/* priority */}
      <div className="h-5">
        <PriorityDropdown
          value={ISSUE_PRIORITIES.find((row) => row.key === issue.priority)?.key}
          onChange={handlePriority}
          buttonVariant="border-without-text"
          buttonClassName="border"
          renderByDefault={isMobile}
          showTooltip
        />
      </div>

      {/* label */}

      {draft.project_id && (
        <IssuePropertyLabels
          projectId={draft.project_id}
          value={issue.label_ids ?? []}
          defaultOptions={defaultLabelOptions}
          onChange={handleLabel}
          renderByDefault={isMobile}
          hideDropdownArrow
        />
      )}

      {/* start date */}
      <div className="h-5">
        <DateDropdown
          value={issue.start_date ?? null}
          onChange={handleStartDate}
          maxDate={maxDate}
          placeholder="Start date"
          icon={<StartDateOutline className="h-3 w-3 flex-shrink-0" />}
          buttonVariant={issue.start_date ? "border-with-text" : "border-without-text"}
          optionsClassName="z-10"
          renderByDefault={isMobile}
          showTooltip
        />
      </div>

      {/* target/due date */}
      <div className="h-5">
        <DateDropdown
          value={issue?.target_date ?? null}
          onChange={handleTargetDate}
          minDate={minDate}
          placeholder="Due date"
          icon={<DueDateOutline className="h-3 w-3 flex-shrink-0" />}
          buttonVariant={issue.target_date ? "border-with-text" : "border-without-text"}
          buttonClassName={
            shouldHighlightIssueDueDate(issue?.target_date || null, stateDetails?.group) ? "text-danger-primary" : ""
          }
          clearIconClassName="!text-primary"
          optionsClassName="z-10"
          renderByDefault={isMobile}
          showTooltip
        />
      </div>

      {/* assignee */}
      {draft.project_id && (
        <div className="h-5">
          <MemberDropdown
            projectId={draft.project_id ?? undefined}
            value={issue.assignee_ids ?? []}
            onChange={handleAssignee}
            multiple
            buttonVariant={(issue.assignee_ids?.length ?? 0) > 0 ? "transparent-without-text" : "border-without-text"}
            buttonClassName={(issue.assignee_ids?.length ?? 0) > 0 ? "hover:bg-transparent px-0" : ""}
            showTooltip={issue?.assignee_ids?.length === 0}
            placeholder="Assignees"
            optionsClassName="z-10"
            tooltipContent=""
            renderByDefault={isMobile}
          />
        </div>
      )}
    </div>
  );
});
