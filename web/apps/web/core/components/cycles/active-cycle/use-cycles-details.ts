/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import useSWR from "swr";
// hooks
import { useCycle } from "@/hooks/store/use-cycle";

interface IActiveCycleDetails {
  workspaceSlug: string;
  projectId: string;
  cycleId: string | null | undefined;
}

const useCyclesDetails = (props: IActiveCycleDetails) => {
  // props
  const { workspaceSlug, projectId, cycleId } = props;
  // store hooks
  const { fetchActiveCycleProgress, getCycleById } = useCycle();
  // derived values
  const cycle = cycleId ? getCycleById(cycleId) : null;

  useSWR(
    workspaceSlug && projectId && cycle?.id ? `PROJECT_ACTIVE_CYCLE_${projectId}_PROGRESS_${cycle.id}` : null,
    workspaceSlug && projectId && cycle?.id ? () => fetchActiveCycleProgress(workspaceSlug, projectId, cycle.id) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
  return {
    cycle,
    cycleId,
  };
};
export default useCyclesDetails;
