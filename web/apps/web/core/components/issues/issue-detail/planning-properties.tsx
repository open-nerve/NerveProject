/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { CyclesOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import type { IssueChange } from "@nerve/types";
import { cn } from "@nerve/utils";
import { SidebarPropertyListItem } from "@/components/common/layout/sidebar/property-list-item";
import { CycleDropdown } from "@/components/dropdowns/cycle/dropdown";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { planningOn } from "@/lib/planning-switch";
import type { TIssueOperations } from "./root";

type Props = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  issueOperations: TIssueOperations;
  /** Whether the work item cannot be changed here: its planning shows read-only. */
  disabled: boolean;
};

/**
 * A work item's planning among the properties of its detail's sidebar and of its preview (P7a spec 2.5, R11): its
 * cycle, while its project's cycles are turned on, by its name, an ended or an archived one too (captioned); the
 * project's lists are read for a name the store does not have (usePlanningNames, R14). A pick is sent through the
 * detail's operations, which say why nerve refused it; the value shown is the one nerve confirmed (v0 design 7.7).
 */
export const IssuePlanningProperties = observer(function IssuePlanningProperties(props: Props) {
  const { workspaceSlug, projectId, issueId, issueOperations, disabled } = props;
  const { t } = useTranslation();
  const {
    issue: { getIssueById },
  } = useIssueDetail();
  const { getProjectById } = useProject();
  const issue = getIssueById(issueId);
  const project = getProjectById(projectId);
  usePlanningNames(workspaceSlug, projectId, "cycles", issue?.cycle_id ? [issue.cycle_id] : []);
  if (!issue || !project) return null;
  // Terminal property events have no success follow-up; the operation reports native refusals.
  const change = (data: IssueChange) =>
    issueOperations.update(workspaceSlug, projectId, issueId, data).catch(() => undefined);
  return (
    <>
      {planningOn(project, "cycles") && (
        <SidebarPropertyListItem icon={CyclesOutline} label={t("common.cycle")}>
          <CycleDropdown
            value={issue.cycle_id}
            onChange={(cycleId) => change({ cycle_id: cycleId })}
            projectId={projectId}
            disabled={disabled}
            buttonVariant="transparent-with-text"
            className="group w-full grow"
            buttonContainerClassName="w-full text-left h-7.5"
            buttonClassName={cn("text-body-xs-regular", { "text-placeholder": !issue.cycle_id })}
            dropdownArrow
            dropdownArrowClassName="h-3.5 w-3.5 hidden group-hover:inline"
          />
        </SidebarPropertyListItem>
      )}
    </>
  );
});
