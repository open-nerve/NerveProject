/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { sortBy } from "lodash-es";
import { Avatar } from "@makeplane/propel/components/avatar";
import { EIconSize, ISSUE_PRIORITIES, STATE_GROUPS } from "@nerve/constants";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import { PriorityIcon, StateGroupIcon } from "@nerve/propel/icons";
import type { GroupByColumnTypes, IGroupByColumn } from "@nerve/types";
import { getFileURL } from "@nerve/utils";
import { rootStore } from "@/lib/store-context";

type ColumnScope = {
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  isWorkspaceLevel: boolean;
  groupIds?: Iterable<string>;
};

/** Synchronous directory projection. The host supplies the address and owns any directory reads/errors. */
export function getGroupByColumns({
  groupBy,
  includeNone,
  ...scope
}: ColumnScope & {
  groupBy: GroupByColumnTypes | null;
  includeNone: boolean;
}): IGroupByColumn[] | undefined {
  const workspace = scope.workspaceSlug ? rootStore.workspaceRoot.getWorkspaceBySlug(scope.workspaceSlug) : undefined;
  if (!workspace) return undefined;
  if (!scope.isWorkspaceLevel) {
    const project = rootStore.projectRoot.project.getProjectById(scope.projectId);
    if (
      !project ||
      project.workspace_id !== workspace.id ||
      rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, project.id) === undefined
    )
      return undefined;
  }
  if (!groupBy) return includeNone ? [{ id: "", name: "All work items", payload: {} }] : undefined;
  const columns: Record<GroupByColumnTypes, (scope: ColumnScope) => IGroupByColumn[] | undefined> = {
    project: projectColumns,
    state: stateColumns,
    "state_detail.group": stateGroupColumns,
    priority: priorityColumns,
    labels: labelColumns,
    assignees: assigneeColumns,
    created_by: creatorColumns,
  };
  return columns[groupBy](scope);
}

function projectColumns(scope: ColumnScope): IGroupByColumn[] {
  const workspace = rootStore.workspaceRoot.getWorkspaceBySlug(scope.workspaceSlug!);
  if (!workspace) return [];
  const directory = rootStore.projectRoot.project;
  const ids = scope.isWorkspaceLevel
    ? directory.knownProjectIds(workspace.id)
    : scope.projectId
      ? [scope.projectId]
      : [];
  const projects = ids.flatMap((id) => {
    const project = directory.getProjectById(id);
    return project && project.workspace_id === workspace.id && project.member_role !== null && !project.archived_at
      ? [project]
      : [];
  });
  return sortBy(projects, "sort_order").map((project) => ({
    id: project.id,
    name: project.name,
    icon: (
      <div className="grid h-6 w-6 shrink-0 place-items-center">
        <Logo logo={project.logo_props} />
      </div>
    ),
    payload: { project_id: project.id },
  }));
}
function stateColumns({ projectId, isWorkspaceLevel }: ColumnScope): IGroupByColumn[] | undefined {
  if (isWorkspaceLevel || !projectId) return undefined;
  return rootStore.state.getProjectStates(projectId)?.map((state) => ({
    id: state.id,
    name: state.name,
    icon: (
      <div className="size-4 rounded-full">
        <StateGroupIcon
          stateGroup={state.group}
          color={state.color}
          size={EIconSize.LG}
          percentage={rootStore.state.getStatePercentageInGroup(state.id)}
        />
      </div>
    ),
    payload: { state_id: state.id },
  }));
}
function stateGroupColumns(): IGroupByColumn[] {
  return Object.values(STATE_GROUPS).map((group) => ({
    id: group.key,
    name: group.label,
    icon: (
      <div className="size-4 rounded-full">
        <StateGroupIcon stateGroup={group.key} size={EIconSize.LG} />
      </div>
    ),
    payload: {},
  }));
}
function priorityColumns(): IGroupByColumn[] {
  return ISSUE_PRIORITIES.map((priority) => ({
    id: priority.key,
    name: priority.title,
    icon: <PriorityIcon priority={priority.key} />,
    payload: { priority: priority.key },
  }));
}
function labelColumns({ isWorkspaceLevel, projectId, workspaceSlug, groupIds }: ColumnScope): IGroupByColumn[] {
  const workspace = workspaceSlug ? rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
  const labels = isWorkspaceLevel
    ? Object.values(rootStore.label.labelMap).filter((label) => {
        const project = rootStore.projectRoot.project.getProjectById(label.project_id);
        return (
          project && project.workspace_id === workspace?.id && project.member_role !== null && !project.archived_at
        );
      })
    : (rootStore.label.getProjectLabels(projectId) ?? []);
  const held = new Map(labels.map((label) => [label.id, label]));
  // The server owns group membership/counts. A cold catalog may withhold a name, never a returned group.
  const ids = new Set([...held.keys(), ...(isWorkspaceLevel ? (groupIds ?? []) : []), "none"]);
  return [...ids].map((id) => ({
    id,
    name: id === "none" ? "None" : (held.get(id)?.name ?? "—"),
    icon: <div className="h-3 w-3 rounded-full" style={{ backgroundColor: held.get(id)?.color || "#666" }} />,
    payload: id === "none" ? {} : { label_ids: [id] },
  }));
}
function memberIds(scope: ColumnScope): string[] {
  return scope.isWorkspaceLevel
    ? scope.workspaceSlug
      ? rootStore.memberRoot.workspace.getWorkspaceMemberIds(scope.workspaceSlug)
      : []
    : scope.projectId
      ? (rootStore.memberRoot.project.getProjectMemberIds(scope.projectId, false) ?? [])
      : [];
}
function memberColumns(scope: ColumnScope, assignee: boolean): IGroupByColumn[] {
  return memberIds(scope).flatMap((id) => {
    const member = rootStore.memberRoot.getUserDetails(id);
    return member
      ? [
          {
            id,
            name: member.display_name,
            icon: (
              <Avatar
                alt={member.display_name}
                fallback={member.display_name?.[0]?.toUpperCase()}
                src={getFileURL(member.avatar_url ?? "")}
                size="xs"
              />
            ),
            payload: assignee ? { assignee_ids: [id] } : {},
          },
        ]
      : [];
  });
}
function assigneeColumns(scope: ColumnScope): IGroupByColumn[] {
  return [...memberColumns(scope, true), { id: "none", name: "None", icon: <Avatar size="xs" />, payload: {} }];
}
function creatorColumns(scope: ColumnScope): IGroupByColumn[] {
  return memberColumns(scope, false);
}
