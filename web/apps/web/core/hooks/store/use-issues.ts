/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useContext } from "react";
import { merge } from "lodash-es";
import type { TIssueMap } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { StoreContext } from "@/lib/store-context";
import type { IArchivedIssues, IArchivedIssuesFilter } from "@/store/issue/archived";
import type { IProfileIssues, IProfileIssuesFilter } from "@/store/issue/profile";
import type { IProjectIssues, IProjectIssuesFilter } from "@/store/issue/project";
import type { IWorkspaceDraftIssues, IWorkspaceDraftIssuesFilter } from "@/store/issue/workspace-draft";

type defaultIssueStore = {
  issueMap: TIssueMap;
};

export type TStoreIssues = {
  [EIssuesStoreType.WORKSPACE_DRAFT]: defaultIssueStore & {
    issues: IWorkspaceDraftIssues;
    issuesFilter: IWorkspaceDraftIssuesFilter;
  };
  [EIssuesStoreType.PROFILE]: defaultIssueStore & {
    issues: IProfileIssues;
    issuesFilter: IProfileIssuesFilter;
  };
  [EIssuesStoreType.PROJECT]: defaultIssueStore & {
    issues: IProjectIssues;
    issuesFilter: IProjectIssuesFilter;
  };
  [EIssuesStoreType.ARCHIVED]: defaultIssueStore & {
    issues: IArchivedIssues;
    issuesFilter: IArchivedIssuesFilter;
  };
  [EIssuesStoreType.DEFAULT]: defaultIssueStore & {
    issues: IProjectIssues;
    issuesFilter: IProjectIssuesFilter;
  };
};

export const useIssues = <T extends EIssuesStoreType>(storeType?: T): TStoreIssues[T] => {
  const context = useContext(StoreContext);
  if (context === undefined) throw new Error("useIssues must be used within StoreProvider");

  const defaultStore: defaultIssueStore = {
    issueMap: context.issue.issues.issuesMap,
  };

  switch (storeType) {
    case EIssuesStoreType.WORKSPACE_DRAFT:
      return merge(defaultStore, {
        issues: context.issue.workspaceDraftIssues,
        issuesFilter: context.issue.workspaceDraftIssuesFilter,
      }) as TStoreIssues[T];
    case EIssuesStoreType.PROFILE:
      return merge(defaultStore, {
        issues: context.issue.profileIssues,
        issuesFilter: context.issue.profileIssuesFilter,
      }) as TStoreIssues[T];
    case EIssuesStoreType.PROJECT:
      return merge(defaultStore, {
        issues: context.issue.projectIssues,
        issuesFilter: context.issue.projectIssuesFilter,
      }) as TStoreIssues[T];
    case EIssuesStoreType.ARCHIVED:
      return merge(defaultStore, {
        issues: context.issue.archivedIssues,
        issuesFilter: context.issue.archivedIssuesFilter,
      }) as TStoreIssues[T];
    case undefined:
    case EIssuesStoreType.DEFAULT:
      return merge(defaultStore, {
        issues: context.issue.projectIssues,
        issuesFilter: context.issue.projectIssuesFilter,
      }) as TStoreIssues[T];
    default:
      throw new Error(`Unsupported issue store: ${storeType}`);
  }
};
