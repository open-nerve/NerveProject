/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import useSWR from "swr";
// nerve imports
import { WORKSPACE_CYCLES, WORKSPACE_LABELS, WORKSPACE_MODULES } from "@nerve/constants";
// nerve imports
import { useCycle } from "./store/use-cycle";
import { useLabel } from "./store/use-label";
import { useModule } from "./store/use-module";

export const useWorkspaceIssueProperties = (workspaceSlug: string | undefined) => {
  const { fetchWorkspaceLabels } = useLabel();

  const { fetchWorkspaceModules } = useModule();

  const { fetchWorkspaceCycles } = useCycle();

  // fetch workspace Modules
  useSWR(
    workspaceSlug ? WORKSPACE_MODULES(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceModules(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );

  // fetch workspace Cycles
  useSWR(
    workspaceSlug ? WORKSPACE_CYCLES(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceCycles(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );

  // fetch workspace labels
  useSWR(
    workspaceSlug ? WORKSPACE_LABELS(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceLabels(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
};
