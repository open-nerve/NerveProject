/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { SyntheticEvent } from "react";
import { observer } from "mobx-react";
import type { Issue, Project } from "@nerve/api-client";
import type { IIssueDisplayProperties, IssueChange } from "@nerve/types";
import { CycleDropdown } from "@/components/dropdowns/cycle/dropdown";
import { ModuleDropdown } from "@/components/dropdowns/module/dropdown";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { planningOn, type PlanningDomain } from "@/lib/planning-switch";
import { WithDisplayPropertiesHOC } from "./with-display-properties-HOC";

type Props = {
  workspaceSlug: string | undefined;
  issue: Issue;
  /** The row's own project, whose switches turn its cycle and its modules on. */
  project: Project | undefined;
  displayProperties: IIssueDisplayProperties;
  isReadOnly: boolean;
  /** Sends a change of the row's work item through the page's list actions. */
  onChange: (change: IssueChange) => void;
};

/** Keeps a pick in a property from reaching the row, which would open the work item. */
function keepInProperty(event: SyntheticEvent<HTMLDivElement>) {
  event.stopPropagation();
  event.preventDefault();
}

/** The row's modules, by their names (an archived one captioned); a pick adds the work item to one or takes it out. */
const RowModules = observer(function RowModules({ workspaceSlug, issue, isReadOnly, onChange }: Props) {
  usePlanningNames(workspaceSlug, issue.project_id, "modules", issue.module_ids);
  return (
    // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
    <div className="h-5" onFocus={keepInProperty} onClick={keepInProperty}>
      <ModuleDropdown
        issueName={issue.name}
        value={issue.module_ids}
        onChange={(modules) => onChange({ modules })}
        projectId={issue.project_id}
        disabled={isReadOnly}
        buttonVariant="border-with-text"
        buttonContainerClassName="truncate max-w-40"
        showTooltip
      />
    </div>
  );
});

/** The row's cycle, by its name and status (an ended or archived one too, captioned); a pick is the cycle itself. */
const RowCycle = observer(function RowCycle({ workspaceSlug, issue, isReadOnly, onChange }: Props) {
  usePlanningNames(workspaceSlug, issue.project_id, "cycles", issue.cycle_id ? [issue.cycle_id] : []);
  return (
    // oxlint-disable-next-line jsx_a11y/click-events-have-key-events oxlint-disable-next-line jsx_a11y/no-static-element-interactions
    <div className="h-5" onFocus={keepInProperty} onClick={keepInProperty}>
      <CycleDropdown
        issueName={issue.name}
        value={issue.cycle_id}
        onChange={(cycleId) => onChange({ cycle_id: cycleId })}
        projectId={issue.project_id}
        disabled={isReadOnly}
        buttonVariant="border-with-text"
        buttonContainerClassName="truncate max-w-40"
        showTooltip
      />
    </div>
  );
});

/**
 * A work item's modules, then its cycle, among a list's or a board's row properties (P7a spec 2.5, R11, R14, R20): each
 * while its display property is on and the row's own project has the domain turned on (a page of many projects, such
 * as the member view, has rows of each), read-only while the row cannot be edited. Each control is named by its row; a
 * value the planning store cannot name has its project's lists read, once a project for all its rows (R5, R14). A pick
 * goes through the page's list actions, which follow it as the page's other writes (a cycle's or a module's page reads
 * its progress again): a cycle as itself, a module as the one added or taken out, applied in the work item's turn to
 * the set it then holds (R3). The values shown are the ones nerve confirmed (v0 design 7.7).
 */
export const IssueRowPlanningProperties = observer(function IssueRowPlanningProperties(props: Props) {
  const { displayProperties, project } = props;
  const on = (domain: PlanningDomain) => () => project !== undefined && planningOn(project, domain);
  return (
    <>
      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey="modules"
        shouldRenderProperty={on("modules")}
      >
        <RowModules {...props} />
      </WithDisplayPropertiesHOC>
      <WithDisplayPropertiesHOC
        displayProperties={displayProperties}
        displayPropertyKey="cycle"
        shouldRenderProperty={on("cycles")}
      >
        <RowCycle {...props} />
      </WithDisplayPropertiesHOC>
    </>
  );
});
