/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext } from "react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useIssues } from "./store/use-issues";

export const IssuesStoreContext = createContext<EIssuesStoreType | undefined>(undefined);

export const useIssueStoreType = () => {
  const storeType = useContext(IssuesStoreContext);
  const { globalViewId, viewId, projectId, cycleId, moduleId, userId } = useParams();

  // If store type exists in context, use that store type
  if (storeType) return storeType;

  // A cycle's page lists its cycle's work items (P5b spec 2.5).
  if (cycleId) return EIssuesStoreType.CYCLE;

  // These domain routes no longer own a work item list. Shared creation and peek commands
  // still use DEFAULT with the selected work item/project.
  if (globalViewId || viewId || moduleId) return EIssuesStoreType.DEFAULT;

  if (userId) return EIssuesStoreType.PROFILE;

  if (projectId) return EIssuesStoreType.PROJECT;

  return EIssuesStoreType.DEFAULT;
};

export const useIssuesStore = () => {
  const storeType = useIssueStoreType();

  return useIssues(storeType);
};
