/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { ListPlus } from "lucide-react";
import { useTranslation } from "@nerve/i18n";
import { isPlanningPageStoreType, type PlanningPageStoreType } from "@/hooks/planning-page-domains";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";
import { useAddExistingRequest } from "./command-host";
import { PLANNING_PAGE_LAYOUTS } from "./planning-page-layouts";

type Props = {
  /** The group's name as its header's controls are named by it. */
  label: string;
  className: string;
};

/**
 * A group header's "add existing" (P7b spec 2.9, R5, R20), beside its "+" where the header offers that (its own gate:
 * the page's list not read-only, the caller a project's admin or member, a column nerve takes work items in): on a
 * cycle's or a module's page only, while the page's cycle or module takes work items (its list state's joinable).
 * Named by the group, as the "+" is (M3 handoff 3). The work items chosen join the page, not the group, as the empty
 * state's do: the request carries nothing of the group's (dragging one into the group is what gives it the group's
 * value). The layout's command host holds the dialog, which outlives the header while the list is checked (R11).
 */
export function GroupAddExisting(props: Props) {
  const storeType = useIssueStoreType();
  return isPlanningPageStoreType(storeType) ? <PlanningGroupAddExisting storeType={storeType} {...props} /> : null;
}

const PlanningGroupAddExisting = observer(function PlanningGroupAddExisting(
  props: Props & { storeType: PlanningPageStoreType }
) {
  const { storeType, label, className } = props;
  const { t } = useTranslation();
  const requestAddExisting = useAddExistingRequest();
  const { joinable } = PLANNING_PAGE_LAYOUTS[storeType].useListState();
  if (!joinable) return null;
  return (
    <button
      type="button"
      aria-label={`${t("issue.add.existing")}: ${label}`}
      className={className}
      onClick={() => {
        if (!requestAddExisting) throw new Error("Adding existing work items requires an IssueListCommandHost");
        requestAddExisting({ storeType });
      }}
    >
      <ListPlus width={14} height={14} aria-hidden="true" />
    </button>
  );
});
