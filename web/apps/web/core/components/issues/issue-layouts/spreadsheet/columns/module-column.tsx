/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import type { IssueChange } from "@nerve/types";
import { ModuleDropdown } from "@/components/dropdowns/module/dropdown";
import { usePlanningNames } from "@/hooks/use-planning-names";

type Props = {
  issue: Issue;
  onClose: () => void;
  onChange: (issue: Issue, data: IssueChange) => void;
  disabled: boolean;
};

/**
 * A work item's modules as a table's cell (P7a spec 2.5, R3, R14, R20): by their names, an archived one captioned, the
 * project's lists read once for all its rows while the planning store cannot name one; the control named by its row; a
 * pick is the one module added or taken out, applied in the work item's turn to the set it then holds. As its list
 * closes, the cell takes the focus back, as the table's other cells do.
 */
export const SpreadsheetModuleColumn = observer(function SpreadsheetModuleColumn(props: Props) {
  const { issue, onChange, disabled, onClose } = props;
  const { workspaceSlug } = useParams();
  usePlanningNames(workspaceSlug, issue.project_id, "modules", issue.module_ids);
  return (
    <div className="h-11 border-b-[0.5px] border-subtle">
      <ModuleDropdown
        issueName={issue.name}
        projectId={issue.project_id}
        value={issue.module_ids}
        onChange={(modules) => onChange(issue, { modules })}
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
