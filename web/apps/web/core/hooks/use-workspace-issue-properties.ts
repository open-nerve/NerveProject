/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import useSWR from "swr";
// nerve imports
import { WORKSPACE_MODULES } from "@nerve/constants";
// local imports
import { useModule } from "./store/use-module";

export const useWorkspaceIssueProperties = (workspaceSlug: string | undefined) => {
  const { fetchWorkspaceModules } = useModule();

  // fetch workspace Modules
  useSWR(
    workspaceSlug ? WORKSPACE_MODULES(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceModules(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
};
