/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { Loader } from "@nerve/ui";
import { cycleCounts } from "@nerve/utils";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
// local imports
import { CycleProgress } from "./issue-progress";
import { CycleSidebarDetails } from "./sidebar-details";
import { CycleSidebarHeader } from "./sidebar-header";

type Props = {
  handleClose: () => void;
  cycleId: string;
  projectId: string;
  workspaceSlug: string;
};

/**
 * A cycle's progress sidebar (P5a spec 2.5), of the cycle as the store shows it (its page or its preview reads it):
 * its head, its details and its progress. Its numbers are nerve's progress of the cycle, of one choice (cycleCounts,
 * R7): what nerve froze once the cycle has ended, else live; its members are its list's row's (R11).
 */
export const CycleDetailsSidebar = observer(function CycleDetailsSidebar(props: Props) {
  const { handleClose, projectId, workspaceSlug, cycleId } = props;
  // store hooks
  const { getCycleById, getCycleRow, getCycleProgress } = useCycles();
  const cycle = getCycleById(cycleId);
  const progress = getCycleProgress(cycleId);

  if (!cycle)
    return (
      <Loader className="px-5">
        <div className="space-y-2">
          <Loader.Item height="15px" width="50%" />
          <Loader.Item height="15px" width="30%" />
        </div>
        <div className="mt-8 space-y-3">
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
        </div>
      </Loader>
    );

  const shown = progress && cycleCounts(progress.status, progress.live, progress.progress_snapshot);
  return (
    <div className="relative pb-2">
      <div className="flex w-full flex-col gap-5">
        <CycleSidebarHeader
          cycle={cycle}
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          handleClose={handleClose}
        />
        <CycleSidebarDetails
          cycle={cycle}
          projectId={projectId}
          members={getCycleRow(cycleId)?.assignee_ids}
          counts={shown}
        />
      </div>
      <CycleProgress cycle={cycle} projectId={projectId} progress={shown} />
    </div>
  );
});
