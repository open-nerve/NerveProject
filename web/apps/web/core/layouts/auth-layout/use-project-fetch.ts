/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project } from "@nerve/api-client";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useLabel } from "@/hooks/store/use-label";
import { useMember } from "@/hooks/store/use-member";
import { useModules } from "@/hooks/store/use-modules";
import { useProject } from "@/hooks/store/use-project";
import { useProjectRead } from "@/hooks/use-project-read";
import { useProjectPreferences } from "@/hooks/store/use-project-preferences";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { problemCode } from "@/lib/api-error";
import { planningOn } from "@/lib/planning-switch";
import { useSessionSWR } from "@/lib/use-session-swr";

/**
 * What ProjectAuthWrapper shows for the address's project (M3 design 3.19, 7.6): that nerve cannot be reached, with a
 * retry; a wait for nerve's read of the project; that the project is not found (it does not exist, is deleted, or the
 * caller does not see it); that it is archived; that he sees it and is no member of it, which he may join; or its
 * pages, a member's.
 */
export type ProjectAccess =
  | { kind: "unavailable"; retry: () => void }
  | { kind: "loading" }
  | { kind: "not-found" }
  | { kind: "archived"; project: Project }
  | { kind: "not-member"; project: Project }
  | { kind: "member"; project: Project };

/**
 * The project side of what a page of a project fetches as it mounts (M3 design 3.1, 7.1), and the one decision what
 * the project is to the caller: nerve's read of the project, which decides it; once it says he is a member of a
 * project that is not archived, his tab bar in the project, its labels, its members and its states, which nerve gives
 * its members alone (an archived project's pages do not show, so they need none), and its cycles and its modules that
 * are not archived, each while it is turned on (cycle_view, module_view: their pages show none otherwise; M6 P5a spec
 * 2.2, P6a spec 2.2), by the switch predicate the gate of their pages reads (planningOn, P6a spec 2.3). A project
 * counts in the address's workspace alone, workspaceSlug's (the route's, as useWorkspaceFetch reads it; the stores'
 * current workspace follows the address a render later): one of another of his workspaces is not found here. Gives
 * what the wrapper shows.
 */
export function useProjectFetch(workspaceSlug: string | undefined, projectId: string): ProjectAccess {
  const { getWorkspaceBySlug } = useWorkspace();
  const { getProjectById } = useProject();
  const { fetchNavigation } = useProjectPreferences();
  const { fetchProjectLabels } = useLabel();
  const {
    project: { fetchProjectMembers },
  } = useMember();
  const { fetchProjectStates } = useProjectState();
  const { fetchCycles } = useCycles();
  const { fetchModules } = useModules();
  const read = useProjectRead(projectId);
  const workspace = workspaceSlug === undefined ? null : getWorkspaceBySlug(workspaceSlug);
  const held = getProjectById(projectId);
  const project = held && held.workspace_id === workspace?.id ? held : undefined;
  const access = decide(read, project);
  // the project's own reads, a member's alone, of a project not archived: for anything else they are nothing to fetch
  const member = access.kind === "member" ? access.project.id : undefined;
  useSessionSWR(member ? ["PROJECT_PREFERENCES", member] : null, (id) => fetchNavigation(id));
  useSessionSWR(member ? ["PROJECT_LABELS", member] : null, (id) => fetchProjectLabels(id));
  useSessionSWR(member ? ["PROJECT_MEMBERS", member] : null, (id) => fetchProjectMembers(id));
  useSessionSWR(member ? ["PROJECT_STATES", member] : null, (id) => fetchProjectStates(id));
  const cycles = access.kind === "member" && planningOn(access.project, "cycles") ? access.project.id : undefined;
  useSessionSWR(cycles ? ["PROJECT_CYCLES", cycles] : null, (id) => fetchCycles(id));
  const modules = access.kind === "member" && planningOn(access.project, "modules") ? access.project.id : undefined;
  useSessionSWR(modules ? ["PROJECT_MODULES", modules] : null, (id) => fetchModules(id));
  return access;
}

/** Of the SWR answer to the read of the project, what the decision takes. */
type Read = { data?: unknown; error?: unknown; mutate: () => Promise<unknown> };

/**
 * The decision, in this order: nerve's refusal of the read (the project not found, else nerve not reached, even for a
 * project it gave before); no answer yet; the project as the store now gives it (none once deleted or left, or of
 * another workspace than the address's), archived whatever his role, else member_role says whether he is a member.
 */
function decide(read: Read, project: Project | undefined): ProjectAccess {
  if (read.error) {
    const notFound = problemCode(read.error) === "project.not_found";
    return notFound ? { kind: "not-found" } : { kind: "unavailable", retry: () => void read.mutate() };
  }
  if (read.data === undefined) return { kind: "loading" };
  if (project === undefined) return { kind: "not-found" };
  if (project.archived_at !== null) return { kind: "archived", project };
  return project.member_role === null ? { kind: "not-member", project } : { kind: "member", project };
}
