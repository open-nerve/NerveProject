/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import useSWR from "swr";
// plane web imports
import { WORKSPACE_CYCLES, WORKSPACE_LABELS, WORKSPACE_MODULES } from "@plane/constants";
// plane imports
import { useCycle } from "./store/use-cycle";
import { useLabel } from "./store/use-label";
import { useModule } from "./store/use-module";

export const useWorkspaceIssueProperties = (workspaceSlug: string | undefined) => {
  const { fetchWorkspaceLabels } = useLabel();

  const { fetchWorkspaceModules } = useModule();

  const { fetchWorkspaceCycles } = useCycle();

  // fetch workspace Modules
  useSWR(
    workspaceSlug ? WORKSPACE_MODULES(workspaceSlug.toString()) : null,
    workspaceSlug ? () => fetchWorkspaceModules(workspaceSlug.toString()) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );

  // fetch workspace Cycles
  useSWR(
    workspaceSlug ? WORKSPACE_CYCLES(workspaceSlug.toString()) : null,
    workspaceSlug ? () => fetchWorkspaceCycles(workspaceSlug.toString()) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );

  // fetch workspace labels
  useSWR(
    workspaceSlug ? WORKSPACE_LABELS(workspaceSlug.toString()) : null,
    workspaceSlug ? () => fetchWorkspaceLabels(workspaceSlug.toString()) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
};
