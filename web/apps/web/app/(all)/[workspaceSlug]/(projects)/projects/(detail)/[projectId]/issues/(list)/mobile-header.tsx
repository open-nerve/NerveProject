/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { EIssuesStoreType } from "@nerve/types";
// components
import { MobileHeaderFilters } from "@/components/issues/mobile-header-filters";
// hooks
import { useIssues } from "@/hooks/store/use-issues";

type Props = {
  /** The page's project, as the route gives it. */
  projectId: string;
};

/**
 * The small screen's header of a project's work items page (P7c spec 2.7, X2): the page's layouts and display
 * (MobileHeaderFilters) once nerve has given the page's preferences, as the header at a wider width gives its controls
 * (IssuesHeader): not before nerve's first answer, nor while that first read failed. An observer, so that an answer
 * after the page opened shows them: the layout that passes it is not one (re-76 F1).
 */
export const ProjectIssuesMobileHeader = observer(function ProjectIssuesMobileHeader({ projectId }: Props) {
  const { issuesFilter } = useIssues(EIssuesStoreType.PROJECT);
  const preferencesShown = !!projectId && issuesFilter.getIssueFilters(projectId) !== undefined;

  if (!preferencesShown) return null;

  return <MobileHeaderFilters storeType={EIssuesStoreType.PROJECT} entityId={projectId} />;
});
