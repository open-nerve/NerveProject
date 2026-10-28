/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// nerve imports
import { EIssuesStoreType } from "@nerve/types";
// components
import { ProjectArchivedEmptyState } from "./archived-issues";
import { ProfileViewEmptyState } from "./profile-view";
import { ProjectEmptyState } from "./project-issues";

interface Props {
  storeType: EIssuesStoreType;
}

export function IssueLayoutEmptyState(props: Props) {
  switch (props.storeType) {
    case EIssuesStoreType.PROJECT:
      return <ProjectEmptyState />;
    case EIssuesStoreType.ARCHIVED:
      return <ProjectArchivedEmptyState />;
    case EIssuesStoreType.PROFILE:
      return <ProfileViewEmptyState />;
    default:
      return null;
  }
}
