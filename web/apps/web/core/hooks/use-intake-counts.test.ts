/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, Workspace } from "@nerve/api-client";
import { fetchHanded, handed } from "@/lib/fake-session-swr";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";
import type { IProjectStore } from "@/store/project/project.store";
import { permissionsOver } from "@/store/user/fake-permissions";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The one fetch of the pending work items of the caller's intakes in the route's workspace (P5a spec 2.5, R7), with
// fake-session-swr.ts's stand-in for useSessionSWR. Its gate: the route names a workspace of his, and he is a member
// (an admin, a member or a guest) of at least one of its projects that is not archived and has its intake turned on,
// as the project store holds them (knownProjectIds); its key names those projects, so that a switch turned on reads
// again. The projects are the project store's own, as nerve lists them in fake time; the store's fetch is a stand-in
// that says which workspace it was asked for.

const held = vi.hoisted(() => ({
  workspace: null as Workspace | null,
  projects: undefined as IProjectStore | undefined,
  fetched: [] as string[],
}));
vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => held.workspace }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => held.projects }));
vi.mock("@/hooks/store/use-intake", () => ({
  useIntake: () => ({
    fetchCounts: ({ id, slug }: Pick<Workspace, "id" | "slug">) =>
      Promise.resolve(held.fetched.push(`the counts of ${slug} (${id})`)),
  }),
}));

const { useIntakeCounts } = await import("./use-intake-counts");

const acme = workspaceOf("acme");
/** A project of acme, its intake on and the caller its member, as fields say otherwise. */
const projectIn = (identifier: string, fields: Partial<Project> = {}) =>
  projectOf(identifier, acme.id, { intake_view: true, ...fields });

beforeEach(() => {
  vi.useFakeTimers();
  handed.length = 0;
  held.fetched.length = 0;
});
afterEach(() => vi.useRealTimers());

/**
 * The caller's acme, with projects nerve lists unarchived and archived ones in its archive; the route names acme
 * unless routed says otherwise. Gives nerve and the project store, for a test that lists them again.
 */
async function callerOf(listed: Project[], archived: Project[] = [], routed: Workspace | null = acme) {
  const { nerve, projects } = await permissionsOver([acme], listed);
  if (archived.length) await loadArchivedProjects(nerve, projects, acme, archived);
  Object.assign(held, { workspace: routed, projects });
  return { nerve, projects };
}
/** The fetch the hook handed useSessionSWR as it ran last. */
const lastKey = () => handed.at(-1)?.[0];

describe("useIntakeCounts", () => {
  it("fetches the route's workspace's counts, keyed by its id and the projects that pass, from its slug", async () => {
    await callerOf([projectIn("WEB")]);
    useIntakeCounts();
    expect(lastKey()).toEqual(["INTAKE_COUNTS", acme.id, acme.slug, "p-web"]);
    await fetchHanded();
    expect(held.fetched).toEqual(["the counts of acme (id-acme)"]);
  });

  it("fetches nothing while the route names no workspace of the caller's", async () => {
    await callerOf([projectIn("WEB")], [], null);
    useIntakeCounts();
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
    await fetchHanded();
    expect(held.fetched).toEqual([]);
  });

  it.each<[when: string, listed: Project[], archived: Project[]]>([
    ["its intake is off", [projectIn("WEB", { intake_view: false })], []],
    ["it is archived, its intake on", [], [projectIn("WEB", { archived_at: "2026-11-01T00:00:00Z" })]],
    ["the caller sees it and is no member of it", [projectIn("WEB", { member_role: null })], []],
  ])("fetches nothing when the caller's one project does not pass: %s", async (_, listed, archived) => {
    const { projects } = await callerOf(listed, archived);
    expect(projects.getProjectById("p-web")).toBeDefined();
    useIntakeCounts();
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
    await fetchHanded();
    expect(held.fetched).toEqual([]);
  });

  it("fetches for a guest of a project whose intake is on", async () => {
    await callerOf([projectIn("WEB", { member_role: 5 })]);
    useIntakeCounts();
    expect(lastKey()).toEqual(["INTAKE_COUNTS", acme.id, acme.slug, "p-web"]);
  });

  it("keys by the ids of the projects that pass alone, in their order, not the list's", async () => {
    await callerOf(
      [
        projectIn("WEB"),
        projectIn("OPS", { intake_view: false }),
        projectIn("API", { member_role: 5 }),
        projectIn("HR", { member_role: null }),
      ],
      [projectIn("DOCS", { archived_at: "2026-11-01T00:00:00Z" })]
    );
    useIntakeCounts();
    expect(lastKey()).toEqual(["INTAKE_COUNTS", acme.id, acme.slug, "p-api,p-web"]);
  });

  it("keys anew, and so fetches again, as another project's intake is turned on", async () => {
    const api = projectIn("API");
    const web = projectIn("WEB", { intake_view: false });
    const { nerve, projects } = await callerOf([web, api]);
    useIntakeCounts();
    const before = lastKey();
    expect(before).toEqual(["INTAKE_COUNTS", acme.id, acme.slug, "p-api"]);

    await loadProjects(nerve, projects, acme, [{ ...web, intake_view: true }, api]);
    useIntakeCounts();
    const after = lastKey();
    expect(after).toEqual(["INTAKE_COUNTS", acme.id, acme.slug, "p-api,p-web"]);
    expect(after).not.toEqual(before);
    handed.splice(0, handed.length - 1);
    await fetchHanded();
    expect(held.fetched).toEqual(["the counts of acme (id-acme)"]);
  });
});
