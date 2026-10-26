import type { ProjectDisplayFilters, ProjectDisplayProperties, ProjectIssueFilters } from "@nerve/api-client";

import {
  addProjectMembers,
  changeProjectPreferences,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  readProjectPreferences,
  slugFor,
} from "../../fixtures/api";
import { expectIssuePage } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { createIssue, listIssues, type IssueQuery } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

test("I5 (API): two members retain different complete list settings; me follows the caller, saved filters equal explicit query parameters and never apply implicitly", async ({
  api,
  db,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const a = await newAccount(api, info, "a"),
    b = await newAccount(api, info, "b");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await [a, b].reduce(async (before, who) => {
    await before;
    await inviteAndAccept(api, admin.token, slug, who, 15);
  }, Promise.resolve());
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(
    api,
    admin.token,
    project.id,
    [a, b].map((who) => ({ member_id: who.id, role: 15 as const }))
  );
  const states = await stateIds(api, admin.token, project.id);
  const started = states["In Progress"];
  if (!started) throw new Error("expected started state");
  const mine = await createIssue(api, admin.token, project.id, {
    name: "A's match",
    state_id: started,
    assignee_ids: [a.id],
    target_date: "2026-10-16",
  });
  const yours = await createIssue(api, admin.token, project.id, {
    name: "B's match",
    state_id: started,
    assignee_ids: [b.id],
    target_date: "2026-10-16",
  });
  const outside = await createIssue(api, admin.token, project.id, {
    name: "Outside range",
    state_id: started,
    assignee_ids: [a.id],
    target_date: "2026-10-25",
  });
  const backlog = await createIssue(api, admin.token, project.id, {
    name: "Backlog",
    assignee_ids: [a.id],
    target_date: "2026-10-16",
  });
  const label = await createLabel(api, admin.token, project.id, { name: "Labelled" });
  const labelled = await createIssue(api, admin.token, project.id, {
    name: "Has label",
    state_id: started,
    assignee_ids: [a.id],
    target_date: "2026-10-16",
    label_ids: [label.id],
  });
  const filters: ProjectIssueFilters = {
    state_group: ["started"],
    assignee_id: ["me"],
    label_id: ["none"],
    target_date: ["2026-10-01..2026-10-31", "2026-10-15..2026-10-20"],
  };
  const display: ProjectDisplayFilters = {
    layout: "kanban",
    group_by: "state_id",
    sub_group_by: null,
    order_by: "sort_order",
    sub_issue: false,
    show_empty_groups: false,
    calendar: { layout: "week", show_weekends: true },
  };
  const properties: ProjectDisplayProperties = {
    assignee: true,
    start_date: true,
    due_date: true,
    labels: false,
    key: true,
    priority: true,
    state: true,
    sub_issue_count: true,
    link: false,
    created_on: true,
    updated_on: true,
  };
  const savedA = {
    ...(await readProjectPreferences(api, a.token, project.id)),
    filters,
    display_filters: display,
    display_properties: properties,
  };
  expect(await changeProjectPreferences(api, a.token, project.id, savedA)).toEqual({
    status: 200,
    preferences: savedA,
  });
  const savedB = {
    ...(await readProjectPreferences(api, b.token, project.id)),
    filters: { priority: ["low"] },
    display_filters: { ...display, layout: "calendar", group_by: "priority", order_by: "priority" },
    display_properties: { ...properties, labels: true, priority: false },
  };
  expect(await changeProjectPreferences(api, b.token, project.id, savedB)).toEqual({
    status: 200,
    preferences: savedB,
  });
  // Deliberately write B last: a missing actor condition must not pass by returning A's defaults.
  expect(await readProjectPreferences(api, a.token, project.id)).toEqual(savedA);
  expect(await readProjectPreferences(api, b.token, project.id)).toEqual(savedB);
  await (
    [
      [a, savedA],
      [b, savedB],
    ] as const
  ).reduce(async (before, [who, saved]) => {
    await before;
    expect(
      await db.query(
        `SELECT filters, display_filters, display_properties, updated_by_id::text AS actor
         FROM project_user_properties WHERE project_id=$1 AND user_id=$2 AND deleted_at IS NULL`,
        [project.id, who.id]
      )
    ).toEqual([
      {
        filters: saved.filters,
        display_filters: saved.display_filters,
        display_properties: saved.display_properties,
        actor: who.id,
      },
    ]);
  }, Promise.resolve());
  const loaded = await readProjectPreferences(api, a.token, project.id);
  // Settings intentionally use broad string vocabularies; narrow the two requested values by assertion.
  expect(loaded.display_filters.group_by).toBe("state_id");
  expect(loaded.display_filters.order_by).toBe("sort_order");
  const fromSaved: IssueQuery = {
    ...loaded.filters,
    group_by: "state_id",
    order_by: "sort_order",
    sub_issue: String(loaded.display_filters.sub_issue),
  };
  const explicit: IssueQuery = {
    state_group: ["started"],
    assignee_id: ["me"],
    label_id: ["none"],
    target_date: ["2026-10-01..2026-10-31", "2026-10-15..2026-10-20"],
    group_by: "state_id",
    order_by: "sort_order",
    sub_issue: "false",
  };
  const result = await listIssues(api, a.token, project.id, fromSaved);
  expect(result).toEqual(await listIssues(api, a.token, project.id, explicit));
  expect(result).toMatchObject({
    total: 1,
    groups: [{ key: started, total: 1, data: [{ id: mine.id }], next_cursor: null }],
  });
  expect(await listIssues(api, b.token, project.id, explicit)).toMatchObject({
    total: 1,
    groups: [{ data: [{ id: yours.id }] }],
  });
  const implicit = await listIssues(api, a.token, project.id);
  expect(implicit.total).toBe(5);
  expect(implicit.data?.map((i) => i.id).toSorted()).toEqual(
    [mine.id, yours.id, outside.id, backlog.id, labelled.id].toSorted()
  );
  expect(implicit).not.toHaveProperty("groups");
  expect(await changeProjectPreferences(api, a.token, project.id, { filters: {} })).toEqual({
    status: 200,
    preferences: { ...savedA, filters: {} },
  });
  expect(await readProjectPreferences(api, b.token, project.id)).toEqual(savedB);
  expect(
    await changeProjectPreferences(api, a.token, project.id, { display_filters: { ...display, group_by: "cycle_id" } })
  ).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "display_filters.group_by", code: "not_allowed" }],
  });
  expect(await changeProjectPreferences(api, a.token, project.id, { filters: { label_id: ["wrong"] } })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "filters.label_id[0]", code: "not_allowed" }],
  });
  expect(await readProjectPreferences(api, a.token, project.id)).toEqual({ ...savedA, filters: {} });
  expectIssuePage(await listIssues(api, a.token, project.id, { priority: ["low"] }), []);
});
