/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { EUserPermissions } from "@nerve/constants";
import { SessionChangedError } from "@/lib/auth/token-manager";
import {
  confirmGroups,
  decodePage,
  pagePartKey,
  type PageAnswer,
  type PagePart,
  type Boundary,
} from "@/lib/paged-groups";
import { issueParts } from "@/lib/issue-membership";
import type { IIssueRootStore } from "../root.store";
import type { IssueListBinding } from "./issue-list-binding";

export type IssueListScope =
  | {
      scope: "project";
      /** Proven empty date intersection; never a server query term or read answer. */
      empty?: boolean;
      workspaceSlug: string;
      workspaceID: string;
      projectID: string;
      query: import("@/services/issue/issue-list.service").ProjectIssueQuery;
    }
  | {
      scope: "member";
      workspaceSlug: string;
      workspaceID: string;
      userID: string;
      query: import("@/services/issue/issue-list.service").MemberIssueQuery;
    }
  | {
      /** The workspace's list (M7/P6b spec 2.3, R1): the work items of the projects the caller is a member of; flat. */
      scope: "workspace";
      workspaceSlug: string;
      workspaceID: string;
      query: import("@/services/issue/issue-list.service").WorkspaceIssueQuery;
    };

export type ReadRequest = Readonly<{
  part: PagePart | null;
  first: boolean;
  cursor: string | null;
  lower: Boundary | null;
}>;

/** The user's role in the scope now; undefined without one, or when the slug no longer names the scope's workspace. */
export function scopeRole(root: IIssueRootStore, scope: IssueListScope): number | undefined {
  const stores = root.rootStore;
  if (stores.workspaceRoot.getWorkspaceBySlug(scope.workspaceSlug)?.id !== scope.workspaceID) return undefined;
  const permission = stores.user.permission;
  return scope.scope === "project"
    ? permission.getProjectRoleByWorkspaceSlugAndProjectId(scope.workspaceSlug, scope.projectID)
    : permission.getWorkspaceRoleByWorkspaceSlug(scope.workspaceSlug);
}

/**
 * Archived and member lists need a member or an admin; any role reads a project's open list, and the workspace's list
 * (its guests too, M7/P6b spec 2.3).
 */
export function allowsRole(scope: IssueListScope, role: number | undefined): boolean {
  return scope.scope === "member" || (scope.scope === "project" && scope.query.archived === "true")
    ? role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER
    : role !== undefined;
}

/**
 * One page of the list, checked against its scope, the session and the directories before any row is published.
 * nextSource numbers the page when it is decoded, so a later answer gets a later source.
 */
export async function readPage(
  list: IssueListBinding,
  request: ReadRequest,
  nextSource: () => number
): Promise<PageAnswer> {
  const role = scopeRole(list.root, list.scope);
  if (!allowsRole(list.scope, role)) throw new SessionChangedError();
  const currentScope = () => scopeRole(list.root, list.scope) === role && allowsRole(list.scope, role);
  const { part, first, cursor, lower } = request;
  const token = list.root.captureRead(list.scope.scope === "member");
  const query = {
    ...list.scope.query,
    ...(part
      ? {
          ...(list.scope.query.group_by ? { group_key: part.group } : {}),
          ...(part.subgroup === undefined ? {} : { sub_group_key: part.subgroup }),
          ...(!first && cursor ? { cursor } : {}),
        }
      : {}),
  };
  const scope = list.scope;
  const page =
    scope.scope === "project"
      ? await list.root.listApi.listProject(scope.projectID, query)
      : scope.scope === "member"
        ? await list.root.listApi.listMember(scope.workspaceSlug, scope.userID, {
            ...query,
            relation: scope.query.relation,
          })
        : // flat: a later page names its cursor alone
          await list.root.listApi.listWorkspace(scope.workspaceSlug, {
            ...scope.query,
            ...(part && !first && cursor ? { cursor } : {}),
          });
  const projects = list.scope.scope === "project" ? [list.scope.projectID] : (list.scope.query.project_id ?? []);
  if (!currentScope() || !list.root.isReadCurrent(token, projects, list.scope.workspaceID))
    throw new SessionChangedError();
  const issues = page.groups
    ? page.groups.flatMap((group) =>
        group.subgroups ? group.subgroups.flatMap((cell) => cell.data) : (group.data ?? [])
      )
    : (page.data ?? []);
  if (
    issues.some(
      (issue) =>
        issue.workspace_id !== list.scope.workspaceID ||
        (list.scope.scope === "project" && issue.project_id !== list.scope.projectID) ||
        (list.scope.scope !== "project" &&
          !!list.scope.query.project_id?.length &&
          !list.scope.query.project_id.includes(issue.project_id))
    )
  )
    throw new Error("Issue page does not match its scope");
  const returnedProjects = [...new Set(issues.map((issue) => issue.project_id))];
  const returnedCurrent = () =>
    !returnedProjects.length || list.root.isReadCurrent(token, returnedProjects, list.scope.workspaceID);
  if (!returnedCurrent()) throw new SessionChangedError();
  const projectStore = list.root.rootStore.projectRoot.project;
  await Promise.all(
    returnedProjects.map(async (projectID) => {
      if (!projectStore.getProjectById(projectID)) await projectStore.fetchProject(projectID);
    })
  );
  if (!returnedCurrent() || !currentScope() || !list.root.isReadCurrent(token, projects, list.scope.workspaceID))
    throw new SessionChangedError();
  if (issues.some((issue) => !list.root.canSeeIssue(issue)))
    throw new Error("Required issue project permission is unavailable");
  await list.prepareCatalog(issues);
  if (!returnedCurrent() || !currentScope() || !list.root.isReadCurrent(token, projects, list.scope.workspaceID))
    throw new SessionChangedError();
  const known = list.reconciled.value;
  const rowParts: PagePart[] = part
    ? []
    : [...(known?.groups ?? [])].flatMap(([group, data]) =>
        data.subgroups ? [...data.subgroups.keys()].map((subgroup) => ({ group, subgroup })) : [{ group }]
      );
  let answer = decodePage(
    page,
    {
      sourceID: nextSource(),
      part: list.scope.query.group_by ? part : null,
      first,
      rowParts,
      resetCounters: !part
        ? [...(known?.subgroupTotals?.keys() ?? [])].map((subgroup) => ({ kind: "subgroup", subgroup }))
        : [],
      ranges: new Map(part ? [[pagePartKey(part), { lower, upper: null, complete: false }]] : []),
    },
    list.context
  );
  // Decode every row before touching canonical; malformed trailing rows cannot partially publish a page.
  if (list.root.verification.blocksPage(list, token)) throw new SessionChangedError();
  list.root.acceptListIssues(issues, token);
  for (const issue of issues) {
    if (list.root.issues.held(issue.id) === null)
      answer = {
        ...confirmGroups(answer, { id: issue.id, beforeSnapshot: { kind: "known", issue }, after: null }, list.context)
          .value,
        read: answer.read,
      };
    else if (list.root.verification.isUncertain(issue.id))
      for (const affected of issueParts(issue, list.context)) {
        const group = answer.groups.get(affected.group);
        const cell = affected.subgroup === undefined ? group : group?.subgroups?.get(affected.subgroup);
        if (cell) cell.stale = true;
      }
  }
  return answer;
}
