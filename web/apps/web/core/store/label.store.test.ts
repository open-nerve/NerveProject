/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Label, LabelCreate, LabelUpdate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn, sent } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { LabelStore } from "@/store/label.store";
import { labelOf, projectOf, projectTab } from "@/store/project/fake-projects";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The labels of the caller's projects (M3 design 3.16, 7.3), against a fake nerve that answers each request when the
// test says. The tab's address is web's, a project of acme; ops is acme's other; his other workspace, beta, has lab.

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id);
const ops = projectOf("OPS", acme.id);
const lab = projectOf("LAB", beta.id);
const LIST = `/api/v0/projects/${web.id}/labels`;

const bug = labelOf(web, "bug", 65535);
const feature = labelOf(web, "feature", 75535);
const frontend = labelOf(web, "frontend", 80000, { parent_id: feature.id });
const backend = labelOf(web, "backend", 85000, { parent_id: feature.id });
/** web's labels as nerve lists them, by sort order: two at the top, two under feature. */
const listed = [bug, feature, frontend, backend];
const opsBug = labelOf(ops, "bug", 65535);
/** A label created in web, as the page sends it and as nerve answers it: last, at the top. */
const docs: LabelCreate = { name: "docs", color: "#3F76FF" };
const created = labelOf(web, "docs", 95000, { color: "#3F76FF" });
/** nerve's answer to a change: the fields given, and a change the request does not make, its updated_at. */
const changed = (label: Label, fields: Partial<Label>): Label => ({
  ...label,
  ...fields,
  updated_at: "2026-10-08T09:00:00Z",
});
const ids = (labels: Label[]) => labels.map((label) => label.id);
const at = (label: Label) => `/api/v0/labels/${label.id}`;

/** The store of a tab at web's address, whose caller's acme and beta nerve listed with their projects. */
async function labelStore() {
  const { nerve, api, router, workspaceRoot, projectRoot } = await projectTab(
    { workspace: acme, projects: [web, ops] },
    { workspace: beta, projects: [lab] }
  );
  const store = new LabelStore(fakeRoot({ router, workspaceRoot, projectRoot }), api);
  return { nerve, api, projects: projectRoot.project, store };
}

/** The store fetches web's labels, and nerve lists these. */
const load = (nerve: FakeNerve, store: LabelStore, labels: Label[]) =>
  answered(nerve, () => store.fetchProjectLabels(web.id), ["GET", LIST], { data: labels }, "the labels");

/** A store whose labels of web nerve listed. */
async function loaded() {
  const tab = await labelStore();
  await load(tab.nerve, tab.store, listed);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("LabelStore, the labels", () => {
  it("keeps a project's labels by sort order, then id, and the labels under each label at the top", async () => {
    const { nerve, store } = await labelStore();
    const tied = labelOf(web, "api", 85000, { parent_id: feature.id });
    const fetched = await load(nerve, store, [backend, frontend, bug, tied, feature]);
    expect(fetched.value).toEqual([backend, frontend, bug, tied, feature]);
    const ordered = [bug, feature, frontend, tied, backend];
    expect(store.getProjectLabels(web.id)).toEqual(ordered);
    expect(store.projectLabels).toEqual(ordered);
    expect(store.getProjectLabelIds(web.id)).toEqual(ids(ordered));
    expect(store.projectLabelsTree).toEqual([
      { ...bug, children: [] },
      { ...feature, children: [frontend, tied, backend] },
    ]);
    expect(store.getLabelById(frontend.id)).toEqual(frontend);
    expect(Object.keys(store.labelMap)).toEqual(ids([backend, frontend, bug, tied, feature]));
    expect(store.getProjectLabels(ops.id)).toBeUndefined();
  });

  it("keeps each project's labels apart, and none of a project left", async () => {
    const { nerve, projects, store } = await loaded();
    await answered(
      nerve,
      () => store.fetchProjectLabels(ops.id),
      ["GET", `/api/v0/projects/${ops.id}/labels`],
      {
        data: [opsBug],
      },
      "ops's labels"
    );
    expect(store.getProjectLabels(ops.id)).toEqual([opsBug]);
    expect(store.getProjectLabels(web.id)).toEqual(listed);

    await sent(nerve, () => projects.leaveProject(ops), ["POST", `/api/v0/projects/${ops.id}/leave`], noContent());
    expect(store.getProjectLabels(ops.id)).toBeUndefined();
    expect(store.getLabelById(opsBug.id)).toBeUndefined();
    expect(Object.keys(store.labelMap)).toEqual(ids(listed));
  });

  it("fails when nerve refuses them, keeping none, and again, keeping the labels it had", async () => {
    const { nerve, store } = await labelStore();
    const refused = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 1, "the labels");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectLabels(web.id)).toBeUndefined();

    await load(nerve, store, listed);
    const again = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });

  it("keeps the labels it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchProjectLabels(web.id), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });
});

describe("LabelStore, the changes", () => {
  it("adds a created label as nerve answers it: last at the top, or under the parent the page names", async () => {
    const { nerve, store } = await loaded();
    const made = await sent(nerve, () => store.createLabel(web.id, docs), ["POST", LIST], json(201, created));
    expect(nerve.calls[1]?.body).toEqual(docs);
    expect(made.value).toEqual(created);
    expect(store.getProjectLabels(web.id)).toEqual([...listed, created]);

    const under: LabelCreate = { name: "infra", parent_id: feature.id };
    const infra = labelOf(web, "infra", 105000, { parent_id: feature.id, color: "" });
    await sent(nerve, () => store.createLabel(web.id, under), ["POST", LIST], json(201, infra));
    expect(nerve.calls[2]?.body).toEqual(under);
    expect(store.projectLabelsTree?.find((top) => top.id === feature.id)?.children).toEqual([frontend, backend, infra]);
  });

  it("changes a label as nerve answers it, and moves one under a parent to the top with a null parent", async () => {
    const { nerve, store } = await loaded();
    const renamed = changed(bug, { name: "defect" });
    await sent(nerve, () => store.updateLabel(bug.id, { name: "defect" }), ["PATCH", at(bug)], json(200, renamed));
    expect(nerve.calls[1]?.body).toEqual({ name: "defect" });
    expect(store.getLabelById(bug.id)).toEqual(renamed);

    const top = changed(frontend, { parent_id: null });
    const freed = track(store.updateLabel(frontend.id, { parent_id: null }));
    await until(() => nerve.calls.length === 3, "the move");
    expect(nerve.calls[2]).toMatchObject({ method: "PATCH", path: at(frontend), body: { parent_id: null } });
    expect(store.getLabelById(frontend.id)).toEqual(frontend);
    nerve.calls[2]?.answer(json(200, top));
    await until(() => freed.settled, "the answer");
    expect(store.projectLabelsTree).toEqual([
      { ...renamed, children: [] },
      { ...feature, children: [backend] },
      { ...top, children: [] },
    ]);
  });

  // a drop: what it is, the label dropped, updateLabelPosition's arguments after the label, and what it sends nerve
  const drops: [string, Label, [string | null, string | undefined, boolean], LabelUpdate][] = [
    [
      "before a parent's first",
      backend,
      [feature.id, frontend.id, false],
      { parent_id: feature.id, sort_order: 70000 },
    ],
    ["between two at the top", frontend, [null, feature.id, false], { parent_id: null, sort_order: 70535 }],
    ["last under a top label", bug, [feature.id, undefined, false], { parent_id: feature.id, sort_order: 95000 }],
    ["at the list's end, at the top", backend, [null, bug.id, true], { parent_id: null, sort_order: 85535 }],
    ["under a top label with none: the parent alone", frontend, [bug.id, undefined, false], { parent_id: bug.id }],
  ];
  it.each(drops)("moves a label dropped %s, as nerve answers it", async (_drop, label, to, body) => {
    const { nerve, store } = await loaded();
    const moved = changed(label, body);
    const sentMove = await sent(
      nerve,
      () => store.updateLabelPosition(label.id, ...to),
      ["PATCH", at(label)],
      json(200, moved)
    );
    expect(nerve.calls[1]?.body).toEqual(body);
    expect(sentMove.value).toEqual(moved);
    expect(store.getLabelById(label.id)).toEqual(moved);
  });

  it("sends nothing for a label dropped under its own parent on no label", async () => {
    const { nerve, store } = await loaded();
    const kept = await settle(store.updateLabelPosition(frontend.id, feature.id, undefined, false), "the drop");
    expect(kept).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
  });

  it("deletes a label and the labels under it", async () => {
    const { nerve, store } = await loaded();
    const deleted = await sent(nerve, () => store.deleteLabel(feature.id), ["DELETE", at(feature)], noContent());
    expect(deleted.error).toBeUndefined();
    expect(store.getProjectLabels(web.id)).toEqual([bug]);
    expect(store.getLabelById(frontend.id)).toBeUndefined();
  });

  const refusals: { change: string; send: (store: LabelStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "a creation",
      send: (store) => store.createLabel(web.id, { name: "BUG" }),
      refusal: problem(409, "project.label_name_taken"),
    },
    {
      change: "a change",
      send: (store) => store.updateLabel(feature.id, { parent_id: bug.id }),
      refusal: problem(422, "validation_failed"),
    },
    {
      change: "a move",
      send: (store) => store.updateLabelPosition(feature.id, bug.id, undefined, false),
      refusal: problem(422, "validation_failed"),
    },
    {
      change: "a deletion",
      send: (store) => store.deleteLabel(bug.id),
      refusal: problem(404, "project.label_not_found"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, store } = await loaded();
    const refused = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });

  it.each(refusals.slice(2))(
    "fails, asking nerve nothing, for $change of a label it does not have",
    async ({ send }) => {
      const { nerve, store } = await labelStore();
      const refused = await settle(send(store), "the change");
      expect(refused).toMatchObject({ settled: true, error: new Error("Label not found") });
      expect(nerve.calls).toEqual([]);
    }
  );

  it("sends each change once nerve has answered the one before it, a move's place reckoned from that answer", async () => {
    const { nerve, store } = await loaded();
    const refused = track(store.updateLabel(bug.id, { name: "defect" }));
    const first = track(store.updateLabelPosition(backend.id, feature.id, frontend.id, false));
    const second = track(store.updateLabelPosition(bug.id, feature.id, frontend.id, false));
    await inTurn(nerve, 1, ["PATCH", at(bug)], problem(503, "server_busy"));
    const backendFirst = changed(backend, { sort_order: 70000 });
    await inTurn(nerve, 2, ["PATCH", at(backend)], json(200, backendFirst));
    // frontend's neighbour before it is now backend, as nerve answered the first move
    const bugBetween = changed(bug, { parent_id: feature.id, sort_order: 75000 });
    await inTurn(nerve, 3, ["PATCH", at(bug)], json(200, bugBetween));
    await until(() => second.settled, "the last change");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(first.value).toEqual(backendFirst);
    expect(nerve.calls[3]?.body).toEqual({ parent_id: feature.id, sort_order: 75000 });
    expect(store.projectLabelsTree?.[0]).toEqual({ ...feature, children: [backendFirst, bugBetween, frontend] });
  });

  it("fetches the labels while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateLabel(bug.id, { name: "defect" }), request: ["PATCH", at(bug)] },
      { send: () => store.fetchProjectLabels(web.id), request: ["GET", LIST], body: { data: [bug, feature] } }
    );
    expect(store.getProjectLabels(web.id)).toEqual([bug, feature]);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it, after the change has finished). Of two
// fetches of one project's labels, only the newer writes.
describe("LabelStore, while a fetch is out", () => {
  it("shows once a label created during a refetch whose list has it, and the changes confirmed meanwhile", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 2, "the refetch");
    await sent(nerve, () => store.createLabel(web.id, docs), ["POST", LIST], json(201, created));
    const renamed = changed(bug, { name: "defect" });
    await sent(nerve, () => store.updateLabel(bug.id, { name: "defect" }), ["PATCH", at(bug)], json(200, renamed));
    await sent(nerve, () => store.deleteLabel(feature.id), ["DELETE", at(feature)], noContent());
    // read after the creation, before the change and the deletion
    nerve.calls[1]?.answer(json(200, { data: [...listed, created] }));
    await until(() => refetched.settled, "the refetch");

    expect(store.getProjectLabels(web.id)).toEqual([renamed, created]);
    expect(refetched.value && ids(refetched.value)).toEqual(ids([renamed, created]));
  });

  it("lets each project's newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = await labelStore();
    const older = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 1, "the older labels");
    const opsLabels = track(store.fetchProjectLabels(ops.id));
    await until(() => nerve.calls.length === 2, "ops's labels");
    const newer = track(store.fetchProjectLabels(web.id));
    await until(() => nerve.calls.length === 3, "the newer labels");
    nerve.calls[2]?.answer(json(200, { data: [bug, feature] }));
    await until(() => newer.settled, "the newer labels");
    // a newer fetch of web's labels does not overtake one of ops's
    nerve.calls[1]?.answer(json(200, { data: [opsBug] }));
    await until(() => opsLabels.settled, "ops's labels");
    // read before frontend and backend were made
    nerve.calls[0]?.answer(json(200, { data: listed }));
    await until(() => older.settled, "the older labels");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getProjectLabels(web.id)).toEqual([bug, feature]);
    expect(store.getProjectLabels(ops.id)).toEqual([opsBug]);
  });
});
