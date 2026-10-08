/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Label, LabelCreate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { inTurn, sent } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { LabelStore } from "@/store/label.store";
import { labelOf, projectOf, projectTab } from "@/store/project/fake-projects";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// More of the labels' changes (M3 design 3.16, 7.3; v0 design 7.7) than label.store.test.ts has: a change of a label of
// the caller's other workspace is made on that label's own project's list, whatever the address; and a change finds
// its label, and a move reckons its place, in its turn, from nerve's answers. The tab's address is web's, a project
// of acme; his other workspace, beta, has lab.

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id);
const lab = projectOf("LAB", beta.id);
const LIST = `/api/v0/projects/${web.id}/labels`;
const at = (label: Label) => `/api/v0/labels/${label.id}`;

const bug = labelOf(web, "bug", 65535);
const feature = labelOf(web, "feature", 75535);
const frontend = labelOf(web, "frontend", 80000, { parent_id: feature.id });
const backend = labelOf(web, "backend", 85000, { parent_id: feature.id });
/** web's labels as nerve lists them, by sort order: two at the top, two under feature. */
const listed = [bug, feature, frontend, backend];
const docs: LabelCreate = { name: "docs", color: "#3F76FF" };
const created = labelOf(web, "docs", 95000, { color: "#3F76FF" });
const labBug = labelOf(lab, "bug", 65535);
const labFeature = labelOf(lab, "feature", 75535);
const labUi = labelOf(lab, "ui", 80000, { parent_id: labFeature.id });
/** nerve's answer to a change: the fields given, and a change the request does not make, its updated_at. */
const changed = (label: Label, fields: Partial<Label>): Label => ({
  ...label,
  ...fields,
  updated_at: "2026-10-08T09:00:00Z",
});

/** The store of a tab at web's address, whose caller's acme and beta nerve listed with web and lab: it fetched web's. */
async function loaded() {
  const { nerve, api, router, workspaceRoot, projectRoot } = await projectTab(
    { workspace: acme, projects: [web] },
    { workspace: beta, projects: [lab] }
  );
  const store = new LabelStore(fakeRoot({ router, workspaceRoot, projectRoot }), api);
  await answered(nerve, () => store.fetchProjectLabels(web.id), ["GET", LIST], { data: listed }, "web's labels");
  return { nerve, projects: projectRoot.project, store };
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("LabelStore, another workspace's labels", () => {
  it("makes the changes of a label of his other workspace on its project's list, and reckons a move among them", async () => {
    const { nerve, store } = await loaded();
    const LAB = `/api/v0/projects/${lab.id}/labels`;
    const labs = [labBug, labFeature, labUi];
    await answered(nerve, () => store.fetchProjectLabels(lab.id), ["GET", LAB], { data: labs }, "lab's labels");
    const labDocs = labelOf(lab, "docs", 85535, { color: "#3F76FF" });
    await sent(nerve, () => store.createLabel(lab.id, docs), ["POST", LAB], json(201, labDocs));
    const renamed = changed(labBug, { name: "defect" });
    const rename = () => store.updateLabel(labBug.id, { name: "defect" });
    await sent(nerve, rename, ["PATCH", at(labBug)], json(200, renamed));
    // docs last under lab's feature: a step past ui's 80000, whatever web's labels are
    const moved = changed(labDocs, { parent_id: labFeature.id, sort_order: 90000 });
    const move = () => store.updateLabelPosition(labDocs.id, labFeature.id, undefined, false);
    await sent(nerve, move, ["PATCH", at(labDocs)], json(200, moved));
    expect(nerve.calls.at(-1)?.body).toEqual({ parent_id: labFeature.id, sort_order: 90000 });
    await sent(nerve, () => store.deleteLabel(labUi.id), ["DELETE", at(labUi)], noContent());

    expect(store.getProjectLabels(lab.id)).toEqual([renamed, labFeature, moved]);
    expect(store.getProjectLabels(web.id)).toEqual(listed);
  });
});

describe("LabelStore, a change in its turn", () => {
  it("fails in its turn, asking nerve nothing, for a change queued behind its label's deletion, or of a project he left", async () => {
    const { nerve, projects, store } = await loaded();
    // made while bug is held: the deletion and the creation go out one at a time, and the two changes behind them
    // find in their turn that bug is gone
    const deleted = track(store.deleteLabel(bug.id));
    const made = track(store.createLabel(web.id, docs));
    const behind = [
      track(store.updateLabelPosition(bug.id, feature.id, undefined, false)),
      track(store.deleteLabel(bug.id)),
    ];
    await inTurn(nerve, 1, ["DELETE", at(bug)], noContent());
    await inTurn(nerve, 2, ["POST", LIST], json(201, created));
    await until(() => behind.every((change) => change.settled), "the changes behind them");
    expect([deleted.error, made.value]).toEqual([undefined, created]);
    const notFound = new Error("Label not found");
    expect(behind.map((change) => change.error)).toEqual([notFound, notFound]);

    await sent(nerve, () => projects.leaveProject(web), ["POST", `/api/v0/projects/${web.id}/leave`], noContent());
    const freed = await settle(store.updateLabelPosition(frontend.id, null, undefined, false), "a move in web");
    expect(freed).toMatchObject({ settled: true, error: notFound });
    expect(nerve.calls).toHaveLength(4);
  });

  it("reckons a move from the labels nerve gave, not from a refused move's request: two moves in a row", async () => {
    const { nerve, store } = await loaded();
    // bug last under feature asks for 95000, a step past backend's 85000, and nerve refuses it: bug stays at the top.
    // frontend dropped below backend, the last under feature, then asks for 95000 too (from the refused request, 105000)
    const frontendLast = changed(frontend, { sort_order: 95000 });
    const first = track(store.updateLabelPosition(bug.id, feature.id, undefined, false));
    const second = track(store.updateLabelPosition(frontend.id, feature.id, backend.id, true));
    await inTurn(nerve, 1, ["PATCH", at(bug)], problem(422, "validation_failed"));
    await inTurn(nerve, 2, ["PATCH", at(frontend)], json(200, frontendLast));
    await until(() => second.settled, "the second move");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(second.value).toEqual(frontendLast);
    expect(nerve.calls.slice(1).map((call) => call.body)).toEqual([
      { parent_id: feature.id, sort_order: 95000 },
      { parent_id: feature.id, sort_order: 95000 },
    ]);
    expect(store.projectLabelsTree).toEqual([
      { ...bug, children: [] },
      { ...feature, children: [backend, frontendLast] },
    ]);
  });
});
