/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, Workspace } from "@nerve/api-client";
import { fetchHanded, handed } from "@/lib/fake-session-swr";
import {
  acme,
  intakeIssueOf,
  intakeTab,
  listPath,
  loadList,
  open,
  pageOf,
  snoozedToo,
} from "@/store/issue/intake/fake-intake";
import { filterKey } from "@/store/issue/intake/filters";
import { loadArchivedProjects, loadProjects, projectOf } from "@/store/project/fake-projects";

// The read of a project's intake list the intake page's filters select (P5a spec 2.5), with fake-session-swr.ts's
// stand-in for useSessionSWR: keyed by the caller's permissions (identity), the route's workspace by its id and slug,
// the project and the filters' key, so that a change of filters reads the list they name; its fetcher given the key's
// strings alone, so that it fetches the filters the key names (filtersOf), never those of the render that made it.
// Its gate: the route names a workspace of the caller's, and the project is one whose intake he is offered
// (intakeEntryOn: turned on, and a role of his in it); none otherwise, the page then disabled. The stores are the
// session's own, as nerve lists acme and its projects to u-me in fake time; the route's workspace is a stand-in.

const held = vi.hoisted(() => ({ workspace: null as Workspace | null }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@/lib/use-session-swr", () => import("@/lib/fake-session-swr"));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => held.workspace }));
vi.mock("@/hooks/store/use-intake", () => ({ useIntake: () => tab.store }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));

const { useIntakeList } = await import("./use-intake-list");

let tab: Awaited<ReturnType<typeof intakeTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  handed.length = 0;
  held.workspace = acme;
  tab = await intakeTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

const ADMIN = 20 as const;
const GUEST = 5 as const;
/** The caller's acme lists WEB as fields say (its intake on and he its admin unless they say otherwise). */
async function webIs(fields: Partial<Project> = {}) {
  const web = projectOf("WEB", acme.id, { intake_view: true, member_role: ADMIN, ...fields });
  const store = tab.app.projectRoot.project;
  if (web.archived_at) await loadArchivedProjects(tab.nerve, store, acme, [web]);
  else await loadProjects(tab.nerve, store, acme, [web]);
  tab.nerve.calls.length = 0;
}
/** The fetch the hook handed useSessionSWR as it ran last, and its configuration. */
const last = () => handed.at(-1)!;
const identity = () => tab.app.issue.permissionReads.identity(acme.id, ["p-web"], true);

describe("useIntakeList", () => {
  it("reads the list keyed by the caller's permissions, the route's workspace, the project and the filters", async () => {
    await webIs();
    useIntakeList("p-web", open);
    expect(last()[0]).toEqual(["INTAKE_ISSUES", identity(), acme.id, "acme", "p-web", filterKey(open)]);
    // read again however soon the filters return: a confirmation joins no list, which only this read shows (R31);
    // the stand-in keeps every argument the hook gave, its configuration the third
    expect((last() as unknown[])[2]).toEqual({ dedupingInterval: 0 });
    useIntakeList("p-web", snoozedToo);
    expect(last()[0]).toEqual(["INTAKE_ISSUES", identity(), acme.id, "acme", "p-web", filterKey(snoozedToo)]);
  });

  it("fetches the filters the key names, never those of the render that made the fetcher", async () => {
    await webIs();
    const fetchList = vi.spyOn(tab.store, "fetchList").mockResolvedValue(undefined);
    useIntakeList("p-web", open);
    const [fetch, fetcher] = last();
    const [, ...args] = fetch!;
    // SWR calls the fetcher with the key's strings: here a key of other filters than the render's
    await fetcher(...(args.slice(0, -1) as string[]), filterKey(snoozedToo));
    expect(fetchList).toHaveBeenCalledExactlyOnceWith("p-web", snoozedToo);
  });

  it("gives the list as the store holds it, the pages read so far as one", async () => {
    await webIs();
    await loadList(tab.nerve, tab.store, "p-web", open, pageOf([intakeIssueOf("one")], null, 1));
    expect(useIntakeList("p-web", open).page).toEqual(pageOf([intakeIssueOf("one")], null, 1));
    expect(useIntakeList("p-web", snoozedToo).page).toBeUndefined();
  });

  it("reads for a guest of the project, and in an archived project, whose intake is read as ever (R21)", async () => {
    await webIs({ member_role: GUEST });
    useIntakeList("p-web", open);
    expect(last()[0]).not.toBeNull();
    await webIs({ archived_at: "2026-11-01T00:00:00Z" });
    useIntakeList("p-web", open);
    expect(last()[0]).not.toBeNull();
  });

  it.each<[when: string, fields: Partial<Project>, workspace: Workspace | null]>([
    ["the project's intake is off (the page is disabled)", { intake_view: false }, acme],
    ["the caller has no role in the project", { member_role: null }, acme],
    ["the route names no workspace of the caller's", {}, null],
  ])("reads nothing while %s", async (_, fields, workspace) => {
    await webIs(fields);
    held.workspace = workspace;
    const { page } = useIntakeList("p-web", open);
    expect(handed.map(([fetch]) => fetch)).toEqual([null]);
    expect(page).toBeUndefined();
    await fetchHanded();
    expect(tab.nerve.to(listPath("p-web"))).toEqual([]);
  });
});
