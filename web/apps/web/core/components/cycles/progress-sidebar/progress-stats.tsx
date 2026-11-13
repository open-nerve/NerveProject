/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { Progress } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
// components
import { ProgressTabs } from "@/components/core/sidebar/progress-tabs";
import { usePageRowChoices } from "@/components/core/sidebar/use-page-row-choices";

type Props = {
  cycleId: string;
  projectId: string;
  /** The progress the sidebar shows (cycleCounts' choice): its counts, its assignees and its labels. */
  progress: Progress;
};

/**
 * The breakdowns of a cycle's progress (P5a spec 2.4, 2.5), each a tab (ProgressTabs), the tab chosen kept for the
 * cycle (localStorage). While the cycle's page shows its work items by its filters, a row of a state group, an assignee
 * or a label toggles its value in them and says whether it is chosen (usePageRowChoices, P5b spec 2.8, R8); the rows
 * are plain before the page has its filters, and in the list page's preview (?peekCycle), which never has them.
 */
export const CycleProgressStats = observer(function CycleProgressStats(props: Props) {
  const { cycleId, projectId, progress } = props;
  // hooks
  const choices = usePageRowChoices(EIssuesStoreType.CYCLE, cycleId);

  return (
    <ProgressTabs
      progress={progress}
      projectId={projectId}
      storageKey={`cycle-progress-tab-${cycleId}`}
      choices={choices}
    />
  );
});
