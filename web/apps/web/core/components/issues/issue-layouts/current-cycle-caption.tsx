/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import { CyclesOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
import { useCycles } from "@/hooks/store/use-cycles";
import { usePlanningNames } from "@/hooks/use-planning-names";

/**
 * A work item's cycle as a cycle's page's "add existing" lists the work item (P7b spec 2.9, R2, the controller's A′):
 * the cycle it is in, which adding it to the page's leaves, so that the move shows before it is sent. By its name as
 * the planning store has it, an archived one captioned so, read through usePlanningNames (the project's lists read once
 * for all the options, whoever asks); nothing for a work item in no cycle, nor while its cycle has no name.
 */
export const CurrentCycleCaption = observer(function CurrentCycleCaption({ issue }: { issue: Issue }) {
  const { workspaceSlug } = useParams();
  const { t } = useTranslation();
  const { getCycleById } = useCycles();
  usePlanningNames(workspaceSlug, issue.project_id, "cycles", issue.cycle_id ? [issue.cycle_id] : []);
  const cycle = issue.cycle_id ? getCycleById(issue.cycle_id) : undefined;
  if (!cycle) return null;
  return (
    <span className="ml-auto flex max-w-[40%] shrink-0 items-center gap-1 text-11 text-secondary">
      <CyclesOutline className="size-3 shrink-0" aria-hidden="true" />
      <span className="sr-only">, {t("common.cycle")}: </span>
      <span className="truncate">{cycle.name}</span>
      {cycle.archived_at !== null && (
        <span className="shrink-0 text-placeholder">
          <span className="sr-only">, </span>
          {t("common.archived")}
        </span>
      )}
    </span>
  );
});
