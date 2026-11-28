/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Label } from "@nerve/api-client";
import { FakeNerve, answered, json, noContent } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { acme, beta, changed, lab, labelRequests, loadLabels, ops, web, webLabels } from "@/store/fake-project-lists";
import { sent } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { LabelStore } from "@/store/label.store";
import { labelOf, loadArchivedProjects, projectOf, projectTab } from "@/store/project/fake-projects";

// The workspace's label catalog in the label store (M7/P6b spec 2.2, R17, R32), against a fake nerve that answers each
// request when the test says: acme's catalog, the labels of its projects the caller is a member of, unarchived, which
// gives each such project its part until its own list is fetched. The tab's address is web's, a project of acme; ops
// is acme's other; open is acme's public project the caller sees but is no member of; shelf is acme's archived one;
// his other workspace, beta, has lab.

const ACME = "/api/v0/workspaces/acme/labels";
const { at } = labelRequests;
const { bug, feature, frontend, backend, listed, docs, created } = webLabels;
const open = projectOf("OPEN", acme.id, { member_role: null });
const shelf = projectOf("SHELF", acme.id, { archived_at: "2026-10-20T09:00:00Z" });
const opsBug = labelOf(ops, "bug", 65535);
const shelved = labelOf(shelf, "old", 65535);
const ids = (labels: Label[]) => labels.map((label) => label.id);

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

/** A label store on a tab whose project store gives acme's web, ops and open, its archived shelf, and beta's lab. */
async function catalogTab() {
  const { nerve, api, router, workspaceRoot, projectRoot } = await projectTab(
    { workspace: acme, projects: [web, ops, open] },
    { workspace: beta, projects: [lab] }
  );
  await loadArchivedProjects(nerve, projectRoot.project, acme, [shelf]);
  nerve.calls.length = 0;
  const store = new LabelStore(fakeRoot({ router, workspaceRoot, projectRoot }), api);
  return { nerve, api, store };
}

/** The store fetches acme's catalog, and nerve lists these. */
const loadCatalog = (nerve: FakeNerve, store: LabelStore, labels: Label[]) =>
  answered(nerve, () => store.fetchWorkspaceLabels(acme), ["GET", ACME], { data: labels }, "acme's labels");

describe("LabelStore, the workspace's catalog", () => {
  it("reads acme's catalog once, by its slug, and gives each project of it its labels, in their order", async () => {
    const { nerve, store } = await catalogTab();
    const fetched = await loadCatalog(nerve, store, [opsBug, backend, frontend, bug, feature]);
    expect(fetched.value).toEqual([opsBug, backend, frontend, bug, feature]);
    expect(nerve.calls.map((call) => [call.method, call.path])).toEqual([["GET", ACME]]);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
    expect(store.getProjectLabelIds(web.id)).toEqual(ids(listed));
    expect(store.getProjectLabelsTree(web.id)).toEqual([
      { ...bug, children: [] },
      { ...feature, children: [frontend, backend] },
    ]);
    expect(store.getProjectLabels(ops.id)).toEqual([opsBug]);
    expect(store.getLabelById(opsBug.id)).toEqual(opsBug);
    expect(Object.keys(store.labelMap)).toEqual(ids([opsBug, backend, frontend, bug, feature]));
    expect(store.getProjectLabels(lab.id)).toBeUndefined();
  });

  it("gives a project's own list once fetched, and its copy of a label the catalog has too", async () => {
    const { nerve, store } = await catalogTab();
    await loadCatalog(nerve, store, [opsBug, ...listed]);
    const renamed = changed(bug, { name: "defect" });
    await loadLabels(nerve, store, [renamed, feature]);
    expect(store.getProjectLabels(web.id)).toEqual([renamed, feature]);
    expect(store.getLabelById(bug.id)).toEqual(renamed);
    expect(store.getProjectLabels(ops.id)).toEqual([opsBug]);
  });

  it("gives nothing of the catalog for an archived project, which it may still list: not listed is not no labels", async () => {
    const { nerve, store } = await catalogTab();
    // read before shelf was archived
    await loadCatalog(nerve, store, [opsBug, shelved]);
    expect(store.getProjectLabels(shelf.id)).toBeUndefined();
    expect(store.getProjectLabels(ops.id)).toEqual([opsBug]);
    await loadLabels(nerve, store, [shelved], shelf);
    expect(store.getProjectLabels(shelf.id)).toEqual([shelved]);
  });

  it("gives nothing of the catalog for a project the caller sees but is no member of (member_role null)", async () => {
    const { nerve, store } = await catalogTab();
    await loadCatalog(nerve, store, [opsBug, ...listed]);
    expect(store.getProjectLabels(open.id)).toBeUndefined();
    expect(store.getProjectLabelsTree(open.id)).toBeUndefined();
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });

  it("keeps each workspace's catalog under its id: beta's does not replace acme's", async () => {
    const { nerve, store } = await catalogTab();
    await loadCatalog(nerve, store, listed);
    const bench = labelOf(lab, "bench", 65535);
    const BETA = "/api/v0/workspaces/beta/labels";
    await answered(nerve, () => store.fetchWorkspaceLabels(beta), ["GET", BETA], { data: [bench] }, "beta's labels");
    expect(store.getProjectLabels(web.id)).toEqual(listed);
    expect(store.getProjectLabels(lab.id)).toEqual([bench]);
  });

  it("keeps the catalog it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await catalogTab();
    await loadCatalog(nerve, store, listed);
    FakeNerve.replaceSession(api);
    const cut = await settle(store.fetchWorkspaceLabels(acme), "the refetch the session cut");
    expect(cut).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });
});

describe("LabelStore, the changes on the catalog", () => {
  it("writes a created, a changed and a moved label into the catalog as nerve answers them: web's own list not fetched", async () => {
    const { nerve, store } = await catalogTab();
    await loadCatalog(nerve, store, [opsBug, ...listed]);
    await sent(nerve, () => store.createLabel(web.id, docs), ["POST", labelRequests.LIST], json(201, created));
    expect(store.getProjectLabels(web.id)).toEqual([...listed, created]);

    const renamed = changed(bug, { name: "defect" });
    await sent(nerve, () => store.updateLabel(bug.id, { name: "defect" }), ["PATCH", at(bug)], json(200, renamed));
    expect(store.getLabelById(bug.id)).toEqual(renamed);

    // backend dropped before frontend: its place reckoned from the catalog's labels under feature
    const first = changed(backend, { sort_order: 70000 });
    await sent(
      nerve,
      () => store.updateLabelPosition(backend.id, feature.id, frontend.id, false),
      ["PATCH", at(backend)],
      json(200, first)
    );
    expect(nerve.calls[3]?.body).toEqual({ parent_id: feature.id, sort_order: 70000 });
    expect(store.getProjectLabelsTree(web.id)?.find((top) => top.id === feature.id)?.children).toEqual([
      first,
      frontend,
    ]);
    expect(store.getProjectLabels(ops.id)).toEqual([opsBug]);
  });

  it("deletes a label and the labels under it from the catalog", async () => {
    const { nerve, store } = await catalogTab();
    await loadCatalog(nerve, store, [opsBug, ...listed]);
    await sent(nerve, () => store.deleteLabel(feature.id), ["DELETE", at(feature)], noContent());
    expect(store.getProjectLabels(web.id)).toEqual([bug]);
    expect(store.getLabelById(frontend.id)).toBeUndefined();
    expect(Object.keys(store.labelMap)).toEqual(ids([opsBug, bug]));
  });

  it("keeps a label created while the catalog is fetched on the catalog the fetch shows", async () => {
    const { nerve, store } = await catalogTab();
    await loadCatalog(nerve, store, listed);
    const refetched = track(store.fetchWorkspaceLabels(acme));
    await until(() => nerve.calls.length === 2, "the refetch");
    await sent(nerve, () => store.createLabel(web.id, docs), ["POST", labelRequests.LIST], json(201, created));
    // read before the creation
    nerve.calls[1]?.answer(json(200, { data: listed }));
    await until(() => refetched.settled, "the refetch");
    expect(store.getProjectLabels(web.id)).toEqual([...listed, created]);
  });
});
