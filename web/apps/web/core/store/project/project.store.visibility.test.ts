/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { reaction } from "mobx";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { sent } from "@/store/fake-queue";
import { loadArchivedProjects, loadProject, loadProjects, preferencesOf, projectOf, projectTab } from "./fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

const ws = workspaceOf("ws");
const other = workspaceOf("other");
const a = projectOf("A", ws.id, { network: 0 });
const b = projectOf("B", ws.id);
const elsewhere = projectOf("ELSE", other.id);
const archivedA = { ...a, archived_at: "2026-10-30T00:00:00Z" };
const dispose: (() => void)[] = [];
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  dispose.splice(0).forEach((close) => close());
  vi.useRealTimers();
});
async function fixture() {
  const tab = await projectTab({ workspace: ws, projects: [a, b] }, { workspace: other, projects: [elsewhere] });
  dispose.push(() => tab.projectRoot.projectFilter.dispose());
  return { ...tab, store: tab.projectRoot.project };
}

// Choosing only present records ignores the newer complete directory's negative evidence.
it.each([false, true])(
  "R06 full directory absence hides an older detail in its own class archived=%s",
  async (archived) => {
    const { nerve, store } = await fixture();
    const value = archived ? archivedA : a;
    if (archived) {
      await loadProjects(nerve, store, ws, [b]);
      await loadArchivedProjects(nerve, store, ws, [value]);
    }
    await loadProject(nerve, store, value);
    expect(store.getProjectById(a.id)).toEqual(value);
    if (archived) await loadArchivedProjects(nerve, store, ws, []);
    else await loadProjects(nerve, store, ws, [b]);
    expect(store.getProjectById(a.id)).toBeUndefined();
    expect(store.getProjectIdentifierById(a.id)).toBeUndefined();
    expect(store.getProjectById(b.id)).toEqual(b);
    expect(store.getProjectById(elsewhere.id)).toEqual(elsewhere);
    // Absence is not a deletion tombstone: a new valid detail may prove visibility again.
    const fresh = { ...value, name: "visible again", member_role: 5 as const };
    await loadProject(nerve, store, value, fresh);
    expect(store.getProjectById(a.id)).toEqual(fresh);
  }
);

// A full unarchived directory does not cover archived records, or another workspace.
it.each([false, true])("R06 unrelated directory cannot negate a detail archived=%s", async (archived) => {
  const { nerve, store } = await fixture();
  const value = archived ? archivedA : a;
  await loadProject(nerve, store, value);
  if (archived) await loadProjects(nerve, store, ws, [b]);
  else await loadArchivedProjects(nerve, store, ws, []);
  await loadProjects(nerve, store, other, []);
  expect(store.getProjectById(a.id)).toEqual(value);
});

// Capturing only the answer time makes a pre-revocation detail look like new proof.
it.each([false, true])("R06 delayed detail cannot restore a removed project after recovery=%s", async (recover) => {
  const { nerve, store } = await fixture();
  await loadProject(nerve, store, a);
  const at = nerve.calls.length;
  const reading = track(store.fetchProject(a.id));
  await until(() => nerve.calls.length === at + 1, "the delayed detail");
  const delayed = nerve.calls[at];
  await loadProjects(nerve, store, ws, [b]);
  const fresh = { ...a, name: "fresh guest proof", member_role: 5 as const };
  if (recover) await loadProjects(nerve, store, ws, [fresh, b]);
  delayed?.answer(json(200, a));
  await until(() => reading.settled, "the stale reply");
  expect(reading.error).toBeUndefined();
  expect(store.getProjectById(a.id)).toEqual(recover ? fresh : undefined);
  expect(reading.value).toEqual(recover ? fresh : null);
});

it("R06 first pending detail cannot bypass a complete directory received while it was out", async () => {
  const { nerve, store } = await fixture();
  await loadProjects(nerve, store, ws, [b]);
  const at = nerve.calls.length;
  const reading = track(store.fetchProject(a.id));
  await until(() => nerve.calls.length === at + 1, "a cold detail");
  await loadProjects(nerve, store, ws, [b]);
  nerve.calls[at]?.answer(json(200, a));
  await until(() => reading.settled, "cold delayed reply");
  expect(reading.value).toBeNull();
  expect(store.getProjectById(a.id)).toBeUndefined();
});

it("R06 a newer matching directory role beats a delayed detail while a newer detail still wins", async () => {
  const { nerve, store } = await fixture();
  const at = nerve.calls.length;
  const reading = track(store.fetchProject(a.id));
  await until(() => nerve.calls.length === at + 1, "a delayed detail");
  const publicNonmember = { ...a, network: 2 as const, member_role: null, sort_order: null };
  await loadProjects(nerve, store, ws, [publicNonmember, b]);
  nerve.calls[at]?.answer(json(200, a));
  await until(() => reading.settled, "delayed detail");
  expect(store.getProjectById(a.id)).toEqual(publicNonmember);
  expect(reading.value).toEqual(publicNonmember);
  const fresh = { ...a, name: "new request after directory" };
  await loadProject(nerve, store, a, fresh);
  expect(store.getProjectById(a.id)).toEqual(fresh);
});

it.each([403, 404, 503])("R06 directory failure leaves prior facts intact status=%s", async (status) => {
  const { nerve, store } = await fixture();
  const prior = { ...a, name: "detail" };
  await loadProject(nerve, store, a, prior);
  const at = nerve.calls.length;
  const listing = track(store.fetchProjects(ws));
  await until(() => nerve.calls.length === at + 1, "the failing directory");
  nerve.calls[at]?.answer(problem(status, "refused"));
  await until(() => listing.settled, "the refusal");
  expect(listing.error).toBeDefined();
  expect(store.getProjectById(a.id)).toEqual(prior);
});

it.each([false, true])(
  "R06 absence before or after the opposite class arrives keeps its new proof first=%s",
  async (absentFirst) => {
    const { nerve, store } = await fixture();
    await loadProject(nerve, store, a);
    if (absentFirst) await loadProjects(nerve, store, ws, [b]);
    await loadArchivedProjects(nerve, store, ws, [archivedA]);
    if (!absentFirst) await loadProjects(nerve, store, ws, [b]);
    expect(store.getProjectById(a.id)).toEqual(archivedA);
    await loadProject(nerve, store, archivedA);
    await loadArchivedProjects(nerve, store, ws, []);
    expect(store.getProjectById(a.id)).toBeUndefined();
    await loadProjects(nerve, store, ws, [a, b]);
    expect(store.getProjectById(a.id)).toEqual(a);
  }
);

it.each([false, true])(
  "R06 removal from the new archive class cannot fall back to its old class delayed=%s",
  async (delayed) => {
    const { nerve, store } = await fixture();
    await loadProject(nerve, store, a);
    const at = nerve.calls.length;
    const reading = delayed ? track(store.fetchProject(a.id)) : undefined;
    if (reading) await until(() => nerve.calls.length === at + 1, "detail before archive");
    // The unarchived directory has not refreshed yet; its copy is older than this archive proof.
    await loadArchivedProjects(nerve, store, ws, [archivedA]);
    expect(store.getProjectById(a.id)).toEqual(archivedA);
    expect.soft(store.workspaceProjectIds).toEqual([b.id]);
    expect.soft(store.joinedProjectIds).toEqual([b.id]);
    await loadArchivedProjects(nerve, store, ws, []);
    expect.soft(store.getProjectById(a.id)).toBeUndefined();
    expect.soft(store.getProjectByIdentifier(a.identifier)).toBeUndefined();
    if (reading) {
      nerve.calls[at]?.answer(json(200, a));
      await until(() => reading.settled, "old unarchived detail");
      expect.soft(reading.value).toBeNull();
      expect.soft(store.getProjectById(a.id)).toBeUndefined();
    }
    // A genuinely new detail can still restore an unarchived project.
    await loadProject(nerve, store, a);
    expect(store.getProjectById(a.id)).toEqual(a);
  }
);

it.each([false, true])(
  "R06 a detail started before its workspace was loaded respects only a newer absence refreshed=%s",
  async (refreshed) => {
    const { nerve, store, workspaceRoot } = await fixture();
    await loadProjects(nerve, store, ws, [b]);
    await loadWorkspaces(nerve, workspaceRoot, [other]);
    const at = nerve.calls.length;
    const reading = track(store.fetchProject(a.id));
    await until(() => nerve.calls.length === at + 1, "detail before workspace load");
    await loadWorkspaces(nerve, workspaceRoot, [ws, other]);
    if (refreshed) await loadProjects(nerve, store, ws, [b]);
    const fresh = { ...a, name: "newly visible" };
    nerve.calls[at]?.answer(json(200, fresh));
    await until(() => reading.settled, "detail after absence");
    expect(reading.value).toEqual(refreshed ? null : fresh);
    expect(store.getProjectById(a.id)).toEqual(refreshed ? undefined : fresh);
  }
);

it.each(["unchanged", "other project", "other class", "refused"] as const)(
  "R06 %s directory during a pending write keeps the valid confirmation",
  async (refresh) => {
    const { nerve, store } = await fixture();
    await loadProject(nerve, store, a);
    const at = nerve.calls.length;
    const writing = track(store.updateProject(a.id, { name: "confirmed" }));
    await until(() => nerve.calls.length === at + 1, "pending rename");
    if (refresh === "other class") await loadArchivedProjects(nerve, store, ws, []);
    else if (refresh === "refused") {
      const listing = track(store.fetchProjects(ws));
      await until(() => nerve.calls.length === at + 2, "refused directory");
      nerve.calls[at + 1]?.answer(problem(503, "server_busy"));
      await until(() => listing.settled, "refused directory settled");
      expect(listing.error).toBeDefined();
    } else await loadProjects(nerve, store, ws, [a, refresh === "other project" ? { ...b, name: "unrelated" } : b]);
    const confirmed = { ...a, name: "confirmed" };
    nerve.calls[at]?.answer(json(200, confirmed));
    await until(() => writing.settled, "confirmed rename");
    expect(writing.error).toBeUndefined();
    expect(store.getProjectById(a.id)).toEqual(confirmed);
  }
);

it("R06 later detail overtakes the invalidated pending detail without reviving its permission", async () => {
  const { nerve, store } = await fixture();
  await loadProject(nerve, store, a);
  const at = nerve.calls.length;
  const oldRead = track(store.fetchProject(a.id));
  await until(() => nerve.calls.length === at + 1, "old detail");
  await loadProjects(nerve, store, ws, [b]);
  const fresh = { ...a, name: "restored by fresh request", member_role: 5 as const };
  await loadProject(nerve, store, a, fresh);
  nerve.calls[at]?.answer(json(200, a));
  await until(() => oldRead.settled, "old detail settles");
  expect(oldRead.value).toBeUndefined();
  expect(store.getProjectById(a.id)).toEqual(fresh);
});

// A list's absence must include the confirmations the existing Reconciled replays.
it.each(["join", "archive", "restore"] as const)("R06 confirmed %s survives an older empty directory", async (kind) => {
  const { nerve, store } = await fixture();
  const initial = kind === "restore" ? archivedA : { ...a, member_role: null, sort_order: null };
  await loadProject(nerve, store, initial);
  const archived = kind === "archive";
  const at = nerve.calls.length;
  const listing = track(archived ? store.fetchArchivedProjects(ws) : store.fetchProjects(ws));
  await until(() => nerve.calls.length === at + 1, "older complete directory");
  const result = archived ? archivedA : a;
  const operation = kind === "join" ? "join" : kind === "archive" ? "archive" : "unarchive";
  await sent(
    nerve,
    () =>
      kind === "join" ? store.joinProject(a.id) : archived ? store.archiveProject(a.id) : store.restoreProject(a.id),
    ["POST", `/api/v0/projects/${a.id}/${operation}`],
    json(200, result)
  );
  nerve.calls[at]?.answer(json(200, { data: [] }));
  await until(() => listing.settled, "older directory after confirmation");
  expect(store.getProjectById(a.id)).toEqual(result);
});

// A confirmation recorded by an older pending read cannot undo a later directory's revocation on replay.
it("R06 a pre-revocation confirmation replay cannot revive the delayed detail", async () => {
  const { nerve, store } = await fixture();
  await loadProject(nerve, store, a);
  const at = nerve.calls.length;
  const reading = track(store.fetchProject(a.id));
  await until(() => nerve.calls.length === at + 1, "old detail");
  const renamed = { ...a, name: "confirmed before revocation" };
  await sent(
    nerve,
    () => store.updateProject(a.id, { name: renamed.name }),
    ["PATCH", `/api/v0/projects/${a.id}`],
    json(200, renamed)
  );
  await loadProjects(nerve, store, ws, [b]);
  nerve.calls[at]?.answer(json(200, a));
  await until(() => reading.settled, "old detail and its confirmation");
  expect(store.getProjectById(a.id)).toBeUndefined();
});

it.each(["update", "join", "archive", "restore"] as const)(
  "R06 late %s cannot restore a directory-revoked project even after recovery",
  async (kind) => {
    const { nerve, store } = await fixture();
    const initial = kind === "restore" ? archivedA : a;
    if (kind === "restore") {
      await loadProjects(nerve, store, ws, [b]);
      await loadArchivedProjects(nerve, store, ws, [initial]);
    }
    await loadProject(nerve, store, initial);
    const at = nerve.calls.length;
    const writing = track(
      kind === "update"
        ? store.updateProject(a.id, { name: "old write" })
        : kind === "join"
          ? store.joinProject(a.id)
          : kind === "archive"
            ? store.archiveProject(a.id)
            : store.restoreProject(a.id)
    );
    await until(() => nerve.calls.length === at + 1, "the old write");
    if (kind === "restore") await loadArchivedProjects(nerve, store, ws, []);
    else await loadProjects(nerve, store, ws, [b]);
    const fresh = { ...initial, name: "new fact", member_role: 5 as const };
    if (kind === "restore") await loadArchivedProjects(nerve, store, ws, [fresh]);
    else await loadProjects(nerve, store, ws, [fresh, b]);
    const answer = { ...(kind === "archive" ? archivedA : a), name: "old write" };
    nerve.calls[at]?.answer(json(200, answer));
    await until(() => writing.settled, "the stale confirmation");
    expect(writing.error).toBeUndefined();
    expect(store.getProjectById(a.id)).toEqual(fresh);
  }
);

it.each(["leave", "sort"] as const)(
  "R06 late %s cannot erase a newer recovered membership or its placement",
  async (kind) => {
    const { nerve, store } = await fixture();
    await loadProject(nerve, store, a);
    const at = nerve.calls.length;
    const writing = track(kind === "leave" ? store.leaveProject(a) : store.updateProjectSortOrder(a, undefined, true));
    await until(() => nerve.calls.length === at + 1, "old membership or placement write");
    await loadProjects(nerve, store, ws, [b]);
    const fresh = { ...a, name: "rejoined", member_role: 5 as const, sort_order: 99 };
    await loadProjects(nerve, store, ws, [fresh, b]);
    nerve.calls[at]?.answer(kind === "leave" ? noContent() : json(200, preferencesOf({ sort_order: 10 })));
    await until(() => writing.settled, "late acknowledgement");
    expect(writing.error).toBeUndefined();
    expect(store.getProjectById(a.id)).toEqual(fresh);
  }
);

it.each([false, true])(
  "R06 a fresh confirmed full project answer can establish visibility after absence detail=%s",
  async (detail) => {
    const { nerve, store } = await fixture();
    if (detail) await loadProject(nerve, store, a);
    await loadProjects(nerve, store, ws, [b]);
    expect(store.getProjectById(a.id)).toBeUndefined();
    const fresh = { ...a, name: "fresh authorized update" };
    await sent(
      nerve,
      () => store.updateProject(a.id, { name: fresh.name }),
      ["PATCH", `/api/v0/projects/${a.id}`],
      json(200, fresh)
    );
    expect(store.getProjectById(a.id)).toEqual(fresh);
  }
);

it.each([false, true].flatMap((archived) => [false, true].map((confirmation) => ({ archived, confirmation }))))(
  "R06 an old opposite-class directory cannot revive a later removed project archived=$archived confirmation=$confirmation",
  async ({ archived, confirmation }) => {
    const { nerve, store } = await fixture();
    const initial = archived ? archivedA : a;
    if (archived) {
      await loadProjects(nerve, store, ws, [b]);
      await loadArchivedProjects(nerve, store, ws, [initial]);
    }
    await loadProject(nerve, store, initial);
    const at = nerve.calls.length;
    const reading = track(archived ? store.fetchArchivedProjects(ws) : store.fetchProjects(ws));
    await until(() => nerve.calls.length === at + 1, "directory from the old class");
    const moved = archived ? a : archivedA;
    if (confirmation) {
      await sent(
        nerve,
        () => (archived ? store.restoreProject(a.id) : store.archiveProject(a.id)),
        ["POST", `/api/v0/projects/${a.id}/${archived ? "unarchive" : "archive"}`],
        json(200, moved)
      );
    } else if (archived) await loadProjects(nerve, store, ws, [moved, b]);
    else await loadArchivedProjects(nerve, store, ws, [moved]);
    expect(store.getProjectById(a.id)).toEqual(moved);
    if (archived) await loadProjects(nerve, store, ws, [b]);
    else await loadArchivedProjects(nerve, store, ws, []);
    expect.soft(store.getProjectById(a.id)).toBeUndefined();
    nerve.calls[at]?.answer(json(200, { data: archived ? [initial] : [initial, b] }));
    await until(() => reading.settled, "late old directory");
    expect(reading.error).toBeUndefined();
    expect(store.getProjectById(a.id)).toBeUndefined();
    expect(store.getProjectByIdentifier(a.identifier)).toBeUndefined();
  }
);

it.each([false, true])(
  "R06 a fresh recovery detail outranks an older directory across archive changes archived=%s",
  async (archived) => {
    const { nerve, store } = await fixture();
    const initial = archived ? archivedA : a;
    if (archived) {
      await loadProjects(nerve, store, ws, [b]);
      await loadArchivedProjects(nerve, store, ws, [initial]);
    }
    await loadProject(nerve, store, initial);
    const at = nerve.calls.length;
    const reading = track(archived ? store.fetchArchivedProjects(ws) : store.fetchProjects(ws));
    await until(() => nerve.calls.length === at + 1, "old directory before transition");
    if (archived) {
      await loadProjects(nerve, store, ws, [a, b]);
      await loadProjects(nerve, store, ws, [b]);
    } else {
      await loadArchivedProjects(nerve, store, ws, [archivedA]);
      await loadArchivedProjects(nerve, store, ws, []);
    }
    const fresh = { ...initial, member_role: 5 as const, name: "new permission proof" };
    await loadProject(nerve, store, initial, fresh);
    nerve.calls[at]?.answer(json(200, { data: archived ? [initial] : [initial, b] }));
    await until(() => reading.settled, "old directory after recovery");
    expect(reading.error).toBeUndefined();
    expect(store.getProjectById(a.id)).toEqual(fresh);
  }
);

it("R06 a request dispatched by the accepted directory's observer starts after that fact", async () => {
  const { nerve, store } = await fixture();
  const reads: Promise<unknown>[] = [];
  dispose.push(
    reaction(
      () => store.getProjectById(a.id)?.member_role,
      (role) => {
        if (role === 5) reads.push(store.fetchProject(a.id));
      }
    )
  );
  await loadProjects(nerve, store, ws, [{ ...a, member_role: 5 }, b]);
  const path = `/api/v0/projects/${a.id}`;
  await until(() => nerve.to(path).length === 1, "new read dispatched after the directory fact");
  const fresh = { ...a, name: "read after the guest directory", member_role: 20 as const };
  nerve.to(path)[0]?.answer(json(200, fresh));
  expect(await reads[0]).toEqual(fresh);
  expect(store.getProjectById(a.id)).toEqual(fresh);
});
