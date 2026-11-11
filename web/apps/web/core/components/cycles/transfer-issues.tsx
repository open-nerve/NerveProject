/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { TransferWorkItemOutline, WarningCircleOutline } from "@makeplane/propel/icons";
// nerve imports
import type { CycleListItem } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { useCyclePage } from "./actions-host";

type EntryProps = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The work items a transfer of the cycle of row would move, for the caller (P5a spec 2.5, M6 design 3.5, R8): those
 * unfinished by its live counts (in the backlog, not started, started), never the counts it was frozen with, which a
 * transfer does not change; of an ended cycle not archived alone, frozen or not, as often as some are left; none for a
 * caller who may not write the project's cycles (canWriteCycles, as nerve decides: not its guests, nor one it shows
 * only his own work items, nor anyone while they are turned off).
 */
function useTransferable(row: CycleListItem | undefined, workspaceSlug: string, projectId: string): number {
  const { canWriteCycles } = useUserPermissions();
  if (!canWriteCycles(workspaceSlug, projectId) || row?.status !== "completed" || row.archived_at !== null) return 0;
  return row.backlog_issues + row.unstarted_issues + row.started_issues;
}

/**
 * A cycle's row's way to transfer its unfinished work items, "Transfer N work items" (useTransferable says when it
 * shows), which opens its page's dialog of the transfer (CycleActionsHost): the dialog stays, though the row's counts,
 * read again, take the way away.
 */
export const TransferIssuesButton = observer(function TransferIssuesButton(props: EntryProps & { row: CycleListItem }) {
  const { row, workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { openDialog } = useCyclePage();
  const count = useTransferable(row, workspaceSlug, projectId);

  if (count === 0) return null;

  return (
    <button
      type="button"
      className="flex h-6 flex-shrink-0 items-center gap-1 px-2 text-11 text-accent-secondary"
      onClick={() => openDialog("transfer", row)}
    >
      <TransferWorkItemOutline className="w-4 fill-accent-primary" aria-hidden="true" />
      <span>{t("project_cycles.transfer.entry", { count })}</span>
    </button>
  );
});

/**
 * The banner of an ended cycle's page while its unfinished work items may be transferred (useTransferable, by its
 * list's row): that an ended cycle takes no more work items, and the button that opens its page's dialog of the
 * transfer (CycleActionsHost).
 */
export const TransferIssues = observer(function TransferIssues(props: EntryProps & { cycleId: string }) {
  const { cycleId, workspaceSlug, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getCycleRow } = useCycles();
  const { openDialog } = useCyclePage();
  const row = getCycleRow(cycleId);
  const count = useTransferable(row, workspaceSlug, projectId);

  if (!row || count === 0) return null;

  return (
    <div className="flex flex-wrap items-center gap-4 px-4 pt-4 pb-2">
      <div className="flex items-center gap-2 text-13 text-secondary">
        <WarningCircleOutline className="h-3.5 w-3.5 text-secondary" aria-hidden="true" />
        <span>{t("project_cycles.transfer.completed")}</span>
      </div>
      <Button
        variant="primary"
        size="lg"
        prependIcon={<TransferWorkItemOutline aria-hidden="true" />}
        onClick={() => openDialog("transfer", row)}
      >
        {t("project_cycles.transfer.title")}
      </Button>
    </div>
  );
});
