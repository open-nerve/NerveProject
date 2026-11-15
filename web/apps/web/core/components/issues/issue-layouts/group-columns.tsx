/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ReactElement } from "react";
import { sortBy } from "lodash-es";
import type { IssueDetail } from "@nerve/api-client";
import { Avatar } from "@makeplane/propel/components/avatar";
import { CyclesOutline, ModuleOutline } from "@makeplane/propel/icons";
import { EIconSize, ISSUE_PRIORITIES, STATE_GROUPS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import { CycleGroupIcon, ModuleStatusIcon, PriorityIcon, StateGroupIcon } from "@nerve/propel/icons";
import type { GroupByColumnTypes, GroupJoinRefusal, IGroupByColumn, TIssueGroupByOptions } from "@nerve/types";
import { ProfileAvatar } from "@/components/assets/profile-image";
import { usePlanningNames } from "@/hooks/use-planning-names";
import { planningOn, type PlanningDomain } from "@/lib/planning-switch";
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
    cycle: (columnScope) => planningColumns("cycles", columnScope),
    module: (columnScope) => planningColumns("modules", columnScope),
  };
  return columns[groupBy](scope);
}

/** The groupings by a domain of the project's planning: by its cycle, by its modules. */
const PLANNING_GROUPINGS: Partial<Record<NonNullable<TIssueGroupByOptions>, PlanningDomain>> = {
  cycle: "cycles",
  module: "modules",
};

/**
 * Reads the names of the groups nerve answered of a grouping by the project's cycles or modules (P7b spec 2.4, 2.12):
 * usePlanningNames on their ids, which reads the project's archive for an archived one, once a project whoever asks,
 * nothing while the domain is off. On no ids for any other grouping, for a hook is called whatever the grouping.
 */
export function usePlanningGroupNames(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  groupBy: TIssueGroupByOptions | undefined,
  groupIds: Iterable<string> | undefined
): void {
  const domain = groupBy ? PLANNING_GROUPINGS[groupBy] : undefined;
  const ids = domain ? [...(groupIds ?? [])].filter((id) => id !== "none") : [];
  usePlanningNames(workspaceSlug, projectId, domain ?? "cycles", ids);
}

/**
 * A column's name as its header shows it (P7b spec 2.4, R4): an archived cycle's or module's with the caption after it,
 * as the cycle dropdown names one (dropdowns/cycle/base.tsx); the label names the header's controls.
 */
export function useGroupName(name: string, archived: boolean | undefined) {
  const { t } = useTranslation();
  const caption = archived ? t("common.archived") : undefined;
  return {
    label: caption ? `${name}, ${caption}` : name,
    caption: caption && (
      <span className="flex-shrink-0 text-caption-sm-regular text-placeholder">
        <span className="sr-only">, </span>
        {caption}
      </span>
    ),
  };
}

/** A value of a domain as its column shows it: its name, icon, whether it is archived and nerve's refusal to join it. */
type PlanningValue = { name: string; icon: ReactElement; archived: boolean; refusal: GroupJoinRefusal | undefined };

/** What a domain's columns read of the planning store, and the columns' payloads and icons (P7b spec 2.4). */
type PlanningGroups = {
  /** The project's values that are not archived, in nerve's order, and its archive; undefined until read. */
  listed: (projectId: string) => string[] | undefined;
  archive: (projectId: string) => string[] | undefined;
  /** A value as the store has it, undefined for one it cannot name. */
  value: (id: string) => PlanningValue | undefined;
  /** The icon of None and of a group the store cannot name. */
  blank: ReactElement;
  payload: (id: string) => Partial<IssueDetail>;
  /** nerve's refusal of a work item's joining any value while the project has the domain turned off. */
  off: GroupJoinRefusal;
};
const PLANNING_GROUPS: Record<PlanningDomain, () => PlanningGroups> = {
  cycles: () => {
    const { cycles } = rootStore.planning;
    return {
      listed: cycles.getProjectCycleIds,
      archive: cycles.getArchivedCycleIds,
      value: (id) => {
        const cycle = cycles.getCycleById(id);
        if (!cycle) return undefined;
        const archived = cycle.archived_at !== null;
        return {
          name: cycle.name,
          icon: <CycleGroupIcon cycleGroup={cycle.status} className="size-3.5 shrink-0" />,
          archived,
          // As nerve checks a cycle joined: archived first, ended last (updateIssue, api/modules/issue.yaml).
          refusal: archived
            ? "planning.archived"
            : cycle.status === "completed"
              ? "planning.cycle_completed"
              : undefined,
        };
      },
      blank: <CyclesOutline className="size-3.5 shrink-0" />,
      payload: (id) => ({ cycle_id: id }),
      off: "planning.cycles_disabled",
    };
  },
  modules: () => {
    const { modules } = rootStore.planning;
    return {
      listed: modules.getProjectModuleIds,
      archive: modules.getArchivedModuleIds,
      value: (id) => {
        const item = modules.getModuleById(id);
        if (!item) return undefined;
        const archived = item.archived_at !== null;
        return {
          name: item.name,
          icon: <ModuleStatusIcon status={item.status} className="size-3.5 shrink-0" />,
          archived,
          refusal: archived ? "planning.archived" : undefined,
        };
      },
      blank: <ModuleOutline className="size-3.5 shrink-0" />,
      payload: (id) => ({ module_ids: [id] }),
      off: "planning.modules_disabled",
    };
  },
};

/**
 * A grouping's columns by the project's cycles or modules (P7b spec 2.4, R4), in the web's order, not nerve's (the text
 * of its keys): the project's values that are not archived, in nerve's order, ended ones too; then the archived ones
 * the answer has, in its archive's order once read, by name and caption; then those the answer has that the store
 * cannot place (in the answer's order, unnamed, until the archive names them; deleted meanwhile after it); None last.
 * A column nerve refuses a work item's joining says why: an archived or ended value, or any value while the project
 * has the domain turned off; None never. A list of several projects has no such columns (nerve refuses the grouping).
 */
function planningColumns(
  domain: PlanningDomain,
  { projectId, isWorkspaceLevel, groupIds }: ColumnScope
): IGroupByColumn[] | undefined {
  const project = isWorkspaceLevel ? undefined : rootStore.projectRoot.project.getProjectById(projectId);
  if (!project) return undefined;
  const groups = PLANNING_GROUPS[domain]();
  const listed = groups.listed(project.id) ?? [];
  const answered = [...(groupIds ?? [])].filter((id) => id !== "none" && !listed.includes(id));
  const archived = (groups.archive(project.id) ?? []).filter((id) => answered.includes(id));
  const on = planningOn(project, domain);
  return [
    ...[...listed, ...archived, ...answered.filter((id) => !archived.includes(id))].map((id) => {
      const value = groups.value(id);
      const column: IGroupByColumn = {
        id,
        name: value?.name ?? "—",
        icon: value?.icon ?? groups.blank,
        payload: groups.payload(id),
      };
      if (value?.archived) column.archived = true;
      const refusal = on ? value?.refusal : groups.off;
      if (refusal) column.refusal = refusal;
      return column;
    }),
    { id: "none", name: "None", icon: groups.blank, payload: {} },
  ];
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
  const workspaceID = scope.workspaceSlug
    ? rootStore.workspaceRoot.getWorkspaceBySlug(scope.workspaceSlug)?.id
    : undefined;
  return memberIds(scope).flatMap((id) => {
    const member = rootStore.memberRoot.getUserDetails(id);
    return member
      ? [
          {
            id,
            name: member.display_name,
            icon: (
              <ProfileAvatar
                alt={member.display_name}
                fallback={member.display_name?.[0]?.toUpperCase()}
                source={{ kind: "member", workspaceID: workspaceID ?? "", memberID: id }}
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
