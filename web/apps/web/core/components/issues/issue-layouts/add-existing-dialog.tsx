/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useParams } from "react-router";
import type { Issue } from "@nerve/api-client";
import type { EIssuesStoreType, IssueChange } from "@nerve/types";
import {
  ExistingIssuesListModal,
  type SelectionSubmitResult,
} from "@/components/core/modals/existing-issues-list-modal";
import {
  isPlanningPageStoreType,
  PLANNING_PAGE_DOMAINS,
  type PlanningPageStoreType,
} from "@/hooks/planning-page-domains";
import type { IssueSearchQuery } from "@/services/issue/issue-search.service";

/**
 * What an entry of "add existing" asks of the layout's command host (P7b spec 2.9, R11): the page whose actions add the
 * work items (the project's, a cycle's or a module's), what each change carries beside the page's own (a calendar's
 * day), and the entry's own search and hiding.
 */
export type AddExistingRequest = {
  storeType: EIssuesStoreType.PROJECT | PlanningPageStoreType;
  extra?: IssueChange;
  searchParams?: Omit<IssueSearchQuery, "project_id" | "q">;
  hide?: (issue: Issue) => boolean;
};

/** A planning page's search: nerve's most at once, there being no further page of it (P7b spec 2.9, R2). */
const PLANNING_SEARCH_LIMIT = 100;

/**
 * The dialog of an "add existing" request the host holds (P7b spec 2.9, R2, R11): it searches the route's project,
 * starts the batch the host sends and, while it stays open, shows each chosen one's outcome. A cycle's or a module's
 * page searches nerve's most at once and hides the work items already in its cycle or module, nerve's search excluding
 * none; one in another cycle is listed, and adding it moves it.
 */
export const AddExistingDialog = observer(function AddExistingDialog(props: {
  request: AddExistingRequest;
  /** Sends the batch of the chosen ones by the host's actions, which outlive the dialog. */
  submit: (ids: readonly string[]) => Promise<SelectionSubmitResult>;
  onClose: () => void;
}) {
  const { request, submit, onClose } = props;
  const params = useParams();
  const { workspaceSlug, projectId } = params;
  const domain = isPlanningPageStoreType(request.storeType) ? PLANNING_PAGE_DOMAINS[request.storeType] : undefined;
  const entityId = domain && params[domain.param];
  const page = projectId && entityId ? { projectID: projectId, entityID: entityId } : undefined;
  if (!workspaceSlug || !projectId) return null;

  return (
    <ExistingIssuesListModal
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      isOpen
      handleClose={onClose}
      searchParams={domain ? { ...request.searchParams, limit: PLANNING_SEARCH_LIMIT } : (request.searchParams ?? {})}
      shouldHideIssue={(issue) => !!request.hide?.(issue) || (!!page && !!domain?.holds(issue, page))}
      handleOnSubmit={submit}
    />
  );
});
