import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import {
  changed,
  createdFor,
  expectIssuePage,
  expectIssuePosition,
  expectIssueUpdated,
  madeBy,
} from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { createIssue, issueRead, listIssues, updateIssue } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

test("I6 (API): one state-and-position PATCH moves a card between Done neighbors, completes it at the write and records only one state activity", async ({
  api,
  db,
}, info) => {
  const admin = await newAccount(api, info, "admin");
  const member = await newAccount(api, info, "member");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, member, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: member.id, role: 15 }]);
  const states = await stateIds(api, member.token, project.id);
  const todo = states.Todo,
    done = states.Done;
  if (!todo || !done) throw new Error("expected Todo and Done");
  const left = await createIssue(api, admin.token, project.id, { name: "Left", state_id: done });
  const right = await createIssue(api, admin.token, project.id, { name: "Right", state_id: done });
  const move = await createIssue(api, admin.token, project.id, { name: "Move", state_id: todo });
  await updateIssue(api, admin.token, left.id, { sort_order: 100 });
  await updateIssue(api, admin.token, right.id, { sort_order: 300 });
  await updateIssue(api, admin.token, move.id, { sort_order: 900 });
  const read = () => listIssues(api, member.token, project.id, { group_by: "state_id", order_by: "sort_order" });
  expect((await read()).groups?.map((g) => [g.key, g.total])).toEqual(
    [
      [todo, 1],
      [done, 2],
    ].toSorted((a, b) => String(a[0]).localeCompare(String(b[0])))
  );
  const moved = await updateIssue(api, member.token, move.id, { state_id: done, sort_order: 200 });
  expect(moved).toMatchObject({
    state_id: done,
    sort_order: 200,
    completed_at: expect.any(String),
    updated_by_id: member.id,
  });
  expect(await issueRead(api, member.token, move.id)).toMatchObject({ status: 200, issue: moved });
  await expectIssuePosition(db, move.id, { state: done, sort: 200, from: todo, actor: member.id });
  await expectIssueUpdated(db, move.id, {
    ...madeBy(admin.email, "Move"),
    state: "Done",
    completed: true,
    by: member.email,
    activity: [...createdFor(), changed("state", "Todo", "Done", "updated the state to", "Done", "Todo")],
    wrote: 1,
  });
  const after = await read();
  expect(after.total).toBe(3);
  expect(after.groups?.map((g) => [g.key, g.total])).toEqual([[done, 3]]);
  expect(after.groups?.[0]?.data?.map((i) => i.id)).toEqual([left.id, move.id, right.id]);
  expectIssuePage(await listIssues(api, member.token, project.id, { state_id: [todo] }), []);
});
