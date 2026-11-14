/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Draft, IssueDraftPayload, Project } from "@nerve/api-client";
import { ISSUE_PRIORITIES } from "@nerve/constants";
import type { DraftEdits } from "@/services/issue/draft-input";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { modulesAfter } from "@/lib/planning-directory";
import { planningOn } from "@/lib/planning-switch";
import { useDetailScope } from "../issue-detail-widgets/use-detail-scope";
import { observer } from "mobx-react";
// icons
import { DueDateOutline, StartDateOutline } from "@makeplane/propel/icons";
// types
import { getDate, renderFormattedPayloadDate, shouldHighlightIssueDueDate } from "@nerve/utils";
// components
import { CycleDropdown } from "@/components/dropdowns/cycle/dropdown";
import { DateDropdown } from "@/components/dropdowns/date";
import { MemberDropdown } from "@/components/dropdowns/member/dropdown";
import { ModuleDropdown } from "@/components/dropdowns/module/dropdown";
import { PriorityDropdown } from "@/components/dropdowns/priority";
import { StateDropdown } from "@/components/dropdowns/state/dropdown";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { IssuePropertyLabels } from "../issue-layouts/properties";

/** A change of the draft, made in its turn of its queue of the payload it then holds. */
type DraftChange = (held: IssueDraftPayload) => DraftEdits;

type Props = {
  issue: Draft;
  /** The row's name, which its cycle's and modules' controls take (R20). */
  name: string;
  workspaceSlug: string;
  updateIssue: (change: DraftChange) => Promise<void>;
  className: string;
};

type PlanningProps = {
  workspaceSlug: string;
  /** The draft's project, of which its cycle and modules are. */
  project: Project;
  payload: IssueDraftPayload;
  name: string;
  change: (change: DraftChange) => Promise<void>;
};

/** The draft's modules, by their names (an archived one captioned): a pick adds one or takes one out. */
const DraftModules = observer(function DraftModules({ workspaceSlug, project, payload, name, change }: PlanningProps) {
  const value = payload.module_ids ?? [];
  usePlanningNames(workspaceSlug, project.id, "modules", value);
  return (
    <div className="h-5">
      <ModuleDropdown
        issueName={name}
        value={value}
        onChange={(modules) => change((held) => ({ module_ids: modulesAfter(held.module_ids ?? [], modules) }))}
        projectId={project.id}
        buttonVariant="border-with-text"
        buttonContainerClassName="truncate max-w-40"
        showTooltip
      />
    </div>
  );
});

/** The draft's cycle, by its name and status (an ended or archived one too, captioned): a pick is the cycle itself. */
const DraftCycle = observer(function DraftCycle({ workspaceSlug, project, payload, name, change }: PlanningProps) {
  usePlanningNames(workspaceSlug, project.id, "cycles", payload.cycle_id ? [payload.cycle_id] : []);
  return (
    <div className="h-5">
      <CycleDropdown
        issueName={name}
        value={payload.cycle_id ?? null}
        onChange={(cycle_id) => change(() => ({ cycle_id }))}
        projectId={project.id}
        buttonVariant="border-with-text"
        buttonContainerClassName="truncate max-w-40"
        showTooltip
      />
    </div>
  );
});

/**
 * A draft's properties on its row of the workspace's drafts. Its modules and its cycle (P7a spec 2.8, R5, R6, R14,
 * R20) show while it has a project with the domain on, named by the row; a value the planning store cannot name has
 * its project's lists read, once a project for all its drafts. Each change is made in the draft's turn of the payload
 * it then holds (v0 design 7.7), the rest of which it keeps: a cycle as itself, none as no key; a module as the one
 * added or taken out of the set it then holds. nerve's refusal is said.
 */
export const DraftIssueProperties = observer(function DraftIssueProperties({
  issue: draft,
  name,
  workspaceSlug,
  updateIssue,
  className,
}: Props) {
  const issue = draft.payload;
  const { getStateById } = useProjectState();
  const { getProjectById } = useProject();
  const { isMobile } = usePlatformOS();
  const scope = useDetailScope(workspaceSlug, undefined, draft.id, true);
  const refuse = useRefusalToast();
  const stateDetails = issue.state_id ? getStateById(issue.state_id) : undefined;
  const project = getProjectById(draft.project_id ?? undefined);
  const change = async (edits: DraftChange) => {
    const current = scope();
    if (!current()) return;
    try {
      await updateIssue(edits);
    } catch (error) {
      if (current()) refuse(error);
    }
  };
  const handleState = (state_id: string) => change(() => ({ state_id }));
  const handlePriority = (priority: string) => change(() => ({ priority }));
  const handleLabel = (label_ids: string[]) => change(() => ({ label_ids }));
  const handleAssignee = (assignee_ids: string[]) => change(() => ({ assignee_ids }));
  const handleStartDate = (date: Date | null) =>
    change(() => ({ start_date: date ? renderFormattedPayloadDate(date) : null }));
  const handleTargetDate = (date: Date | null) =>
    change(() => ({ target_date: date ? renderFormattedPayloadDate(date) : null }));
  const planning = project && { workspaceSlug, project, payload: issue, name, change };

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
          issueName={issue.name}
          projectId={draft.project_id}
          value={issue.label_ids ?? []}
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

      {planning && planningOn(planning.project, "modules") && <DraftModules {...planning} />}
      {planning && planningOn(planning.project, "cycles") && <DraftCycle {...planning} />}
    </div>
  );
});
