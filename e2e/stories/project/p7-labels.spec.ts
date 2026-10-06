import {
  addProjectMembers,
  amidAnotherWorkspace,
  answer,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Label,
  type LabelCreate,
  type LabelUpdate,
} from "../../fixtures/api";
import { expectLabels, type LabelRow } from "../../fixtures/assert/project";
import { bearer, newAccount } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P7, a project's labels (M3 design 2, 3.16): creating them, putting one
// under another, renaming and moving one, deleting one with the label under
// it, with what the names and the two levels keep. The page version comes
// with the labels' settings page (P11).

/** The labels of the project of projectId that the caller of token lists, in their order. */
async function listLabels(api: Api, token: string, projectId: string): Promise<Label[]> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/labels", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  expect(response.status, `list ${projectId}'s labels: ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/** The writes on labels, each by the caller of token: creating one in a project, changing and deleting one named by its id. */
function writes(api: Api) {
  return {
    create: async (token: string, projectId: string, body: LabelCreate) => {
      const { error, response } = await api.POST("/api/v0/projects/{project_id}/labels", {
        params: { path: { project_id: projectId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    update: async (token: string, labelId: string, body: LabelUpdate) => {
      const { error, response } = await api.PATCH("/api/v0/labels/{label_id}", {
        params: { path: { label_id: labelId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, labelId: string) => {
      const { error, response } = await api.DELETE("/api/v0/labels/{label_id}", {
        params: { path: { label_id: labelId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

/** rows with the label of name changed as to says. */
function changed(rows: LabelRow[], name: string, to: Partial<LabelRow>): LabelRow[] {
  return rows.map((l) => (l.name === name ? { ...l, ...to } : l));
}

/** The refusal of a parent_id that 3.16's rules do not allow. */
const parentRefused = { status: 422, code: "validation_failed", errors: [{ field: "parent_id", code: "not_allowed" }] };

test("P7 (API): an admin of a project creates Bug and Feature, each after the labels before it; another admin creates UI, which the first puts under Bug; the other renames it Widgets and moves Feature before Bug; bug, in another case, is taken, and a color too long, a third level, a parent of another project, a label its own parent, a parent for a label with a label under it and a member's writes are refused, each changing nothing; Bug is deleted with Widgets at one moment, after which neither is found, and bug is free again, after Feature, the deleted labels' places no longer counted; a member lists the labels by their order, another project has a Bug of its own, and an archived project's labels are listed, created after its labels, and changed as any other's", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann and mem, acme's members: ann Web's other admin, mem its member.
  const [ann, mem] = await Promise.all(["ann", "mem"].map((label) => newAccount(api, testInfo, label)));
  if (!ann || !mem) {
    throw new Error("the accounts were not registered");
  }
  await Promise.all([ann, mem].map((who) => inviteAndAccept(api, admin.token, slug, who, 15)));
  const web = await amidAnotherWorkspace(api, admin.token, testInfo, () =>
    createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" })
  );
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: ann.id, role: 20 },
    { member_id: mem.id, role: 15 },
  ]);
  const { create, update, remove } = writes(api);
  // A new project has no labels.
  expect(await listLabels(api, mem.token, web.id), "Web's labels as made").toEqual([]);

  // ann creates Bug, Web's first label, at 65535, then Feature after it, at 65535 + 10000; the admin creates UI at the
  // top, after both, without a color (M3 design 4.10).
  const bug = await createLabel(api, ann.token, web.id, { name: "Bug", color: "#EF4444" });
  expect(
    [bug.project_id, bug.workspace_id === web.workspace_id, bug.name, bug.color, bug.parent_id, bug.sort_order],
    "Bug as created"
  ).toEqual([web.id, true, "Bug", "#EF4444", null, 65535]);
  const feature = await createLabel(api, ann.token, web.id, { name: "Feature", color: "#22C55E" });
  const ui = await createLabel(api, admin.token, web.id, { name: "UI" });
  expect([feature.sort_order, ui.sort_order, ui.color], "Feature and UI as created").toEqual([75535, 85535, ""]);
  let labels: LabelRow[] = [
    { name: "Bug", color: "#EF4444", parent: null, sort_order: 65535, deleted: false, by: ann.email },
    { name: "Feature", color: "#22C55E", parent: null, sort_order: 75535, deleted: false, by: ann.email },
    { name: "UI", color: "", parent: null, sort_order: 85535, deleted: false, by: admin.email },
  ];
  await expectLabels(db, web.id, labels);

  // ann drags UI under Bug, as the settings page does: its parent changes, its name and place kept.
  expect(await update(ann.token, ui.id, { parent_id: bug.id }), "ann puts UI under Bug").toEqual({ status: 200 });
  labels = changed(labels, "UI", { parent: "Bug", by: ann.email });
  await expectLabels(db, web.id, labels);
  // The admin renames UI Widgets, its parent and place kept, and moves Feature before Bug.
  expect(await update(admin.token, ui.id, { name: "Widgets" }), "the admin renames UI").toEqual({ status: 200 });
  labels = changed(labels, "UI", { name: "Widgets", by: admin.email });
  await expectLabels(db, web.id, labels);
  expect(await update(admin.token, feature.id, { sort_order: 55535 }), "the admin moves Feature").toEqual({
    status: 200,
  });
  labels = changed(labels, "Feature", { sort_order: 55535, by: admin.email });
  await expectLabels(db, web.id, labels);
  // mem lists them by their order, not the order they were made in, Widgets under Bug.
  expect(
    (await listLabels(api, mem.token, web.id)).map((l) => [l.name, l.parent_id]),
    "Web's labels as mem lists them"
  ).toEqual([
    ["Feature", null],
    ["Bug", null],
    ["Widgets", bug.id],
  ]);

  // Ops, ann's other project of acme, has a Bug of its own: a name is unique in its project alone.
  const ops = await createProject(api, ann.token, slug, { name: "Ops", identifier: "OPS" });
  const opsBug = await createLabel(api, ann.token, ops.id, { name: "Bug" });
  const opsLabels: LabelRow[] = [
    { name: "Bug", color: "", parent: null, sort_order: 65535, deleted: false, by: ann.email },
  ];

  // Refused, each changing nothing of what was just read: ann's creation of QA with a color of 256 characters, one
  // more than a color has; her creation of bug, Bug in another case, and the admin's renaming of Feature WIDGETS;
  // ann's creation of Icons under Widgets, a third level; her moves of Feature under Ops's Bug, of another project,
  // and under itself; her move of Bug, which has Widgets under it, under Feature; mem's change of Feature, his
  // creation of QA and his deletion of Feature, a member's.
  expect(
    [
      await create(ann.token, web.id, { name: "QA", color: "#".repeat(256) }),
      await create(ann.token, web.id, { name: "bug" }),
      await update(admin.token, feature.id, { name: "WIDGETS" }),
      await create(ann.token, web.id, { name: "Icons", parent_id: ui.id }),
      await update(ann.token, feature.id, { parent_id: opsBug.id }),
      await update(ann.token, feature.id, { parent_id: feature.id }),
      await update(ann.token, bug.id, { parent_id: feature.id }),
      await update(mem.token, feature.id, { name: "Story" }),
      await create(mem.token, web.id, { name: "QA" }),
      await remove(mem.token, feature.id),
    ],
    "a long color, bug, WIDGETS, Icons under Widgets, Feature under Ops's Bug and under itself, Bug under Feature, mem's writes"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "color", code: "too_long" }] },
    { status: 409, code: "project.label_name_taken" },
    { status: 409, code: "project.label_name_taken" },
    parentRefused,
    parentRefused,
    parentRefused,
    parentRefused,
    { status: 403, code: "forbidden" },
    { status: 403, code: "forbidden" },
    { status: 403, code: "forbidden" },
  ]);
  await expectLabels(db, web.id, labels);
  await expectLabels(db, ops.id, opsLabels);

  // ann deletes Bug, and with it Widgets, under it: both at one moment, by her.
  expect(await remove(ann.token, bug.id), "ann deletes Bug").toEqual({ status: 204 });
  labels = changed(changed(labels, "Bug", { deleted: true, by: ann.email }), "Widgets", {
    deleted: true,
    by: ann.email,
  });
  await expectLabels(db, web.id, labels);
  expect(
    await db.query(
      `SELECT count(DISTINCT deleted_at)::int AS moments FROM labels WHERE project_id = $1 AND deleted_at IS NOT NULL`,
      [web.id]
    ),
    "the moments of Bug's and Widgets's deletion"
  ).toEqual([{ moments: 1 }]);

  // Not found, changing nothing: ann's deletion of Bug again and her renaming of Widgets; Icons under Bug, a parent
  // deleted, is refused as one of no project.
  expect(
    [
      await remove(ann.token, bug.id),
      await update(ann.token, ui.id, { name: "UI" }),
      await create(ann.token, web.id, { name: "Icons", parent_id: bug.id }),
    ],
    "ann's deletion of Bug again, her renaming of Widgets, Icons under Bug"
  ).toEqual([
    { status: 404, code: "project.label_not_found" },
    { status: 404, code: "project.label_not_found" },
    parentRefused,
  ]);
  await expectLabels(db, web.id, labels);

  // bug is free again: ann creates it after Feature, at 55535 + 10000; the places of the deleted labels, Widgets's
  // 85535 the greatest of all, no longer count.
  expect((await createLabel(api, ann.token, web.id, { name: "bug" })).sort_order, "bug as created").toBe(65535);
  labels = [...labels, { name: "bug", color: "", parent: null, sort_order: 65535, deleted: false, by: ann.email }];
  await expectLabels(db, web.id, labels);
  expect(
    (await listLabels(api, mem.token, web.id)).map((l) => l.name),
    "Web's labels as mem lists them, Bug deleted"
  ).toEqual(["Feature", "bug"]);

  // ann archives Ops: its labels are listed, created, Runbook after Bug, and changed, Runbook moved to the top at
  // last, as any other project's (M3 design 3.19).
  const archived = await api.POST("/api/v0/projects/{project_id}/archive", {
    params: { path: { project_id: ops.id } },
    headers: bearer(ann.token),
  });
  expect(archived.response.status, `ann archives Ops: ${JSON.stringify(archived.error)}`).toBe(200);
  expect(
    (await listLabels(api, ann.token, ops.id)).map((l) => l.name),
    "the labels of Ops, archived"
  ).toEqual(["Bug"]);
  const runbook = await createLabel(api, ann.token, ops.id, { name: "Runbook", parent_id: opsBug.id });
  expect(await update(ann.token, opsBug.id, { name: "Incident" }), "ann renames Ops's Bug").toEqual({ status: 200 });
  const archivedOps: LabelRow[] = [
    { name: "Incident", color: "", parent: null, sort_order: 65535, deleted: false, by: ann.email },
    { name: "Runbook", color: "", parent: "Incident", sort_order: 75535, deleted: false, by: ann.email },
  ];
  await expectLabels(db, ops.id, archivedOps);
  expect(await update(ann.token, runbook.id, { parent_id: null }), "ann moves Runbook to the top").toEqual({
    status: 200,
  });
  await expectLabels(db, ops.id, changed(archivedOps, "Runbook", { parent: null }));

  // The admin's projects of Elsewhere, made around Web, have no labels: no write on Web's labels wrote another
  // project's.
  const elsewhere = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug: slugFor(testInfo, "elsewhere") } },
    headers: bearer(admin.token),
  });
  expect(elsewhere.data?.data.length, `Elsewhere's projects: ${JSON.stringify(elsewhere.error)}`).toBe(2);
  await Promise.all((elsewhere.data?.data ?? []).map((project) => expectLabels(db, project.id, [])));
});
