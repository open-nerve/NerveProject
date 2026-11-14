/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import type { IssueChange } from "@nerve/types";
import { CycleDropdown } from "@/components/dropdowns/cycle/dropdown";
import { usePlanningNames } from "@/hooks/use-planning-names";

type Props = {
  issue: Issue;
  onClose: () => void;
  onChange: (issue: Issue, data: IssueChange) => void;
  disabled: boolean;
};

/**
 * A work item's cycle as a table's cell (P7a spec 2.5, R14, R20): by its name and status, an ended or archived one too
 * (captioned), the project's lists read once for all its rows while the planning store cannot name it; the control
 * named by its row; a pick is the cycle itself. As its list closes, the cell takes the focus back, as the table's other
 * cells do.
 */
export const SpreadsheetCycleColumn = observer(function SpreadsheetCycleColumn(props: Props) {
  const { issue, onChange, disabled, onClose } = props;
  const { workspaceSlug } = useParams();
  usePlanningNames(workspaceSlug, issue.project_id, "cycles", issue.cycle_id ? [issue.cycle_id] : []);
  return (
    <div className="h-11 border-b-[0.5px] border-subtle">
      <CycleDropdown
        issueName={issue.name}
        projectId={issue.project_id}
        value={issue.cycle_id}
        onChange={(cycleId) => onChange(issue, { cycle_id: cycleId })}
        disabled={disabled}
        onClose={onClose}
        buttonVariant="transparent-with-text"
        buttonClassName="text-left rounded-none px-page-x"
        buttonContainerClassName="w-full"
        showTooltip
      />
    </div>
  );
});
