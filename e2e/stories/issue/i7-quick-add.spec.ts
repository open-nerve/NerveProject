import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { createdFor, expectIssueCreated, expectIssuePage, madeBy } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { createIssue, issueCreated, listIssues, updateIssue } from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

test("I7 (API): quick-add uses ordinary creation in the chosen state, after that state's greatest position plus 10000; blank names are refused", async ({
  api,
  db,
}, info) => {
  const member = await newAccount(api, info, "member");
  const slug = slugFor(info);
  await createWorkspace(api, member.token, { name: "Acme", slug });
  const project = await createProject(api, member.token, slug, { name: "Web", identifier: "WEB" });
  const states = await stateIds(api, member.token, project.id);
  const todo = states.Todo;
  if (!todo || !states.Backlog) throw new Error("expected Todo and Backlog");
  const first = await createIssue(api, member.token, project.id, { name: "First", state_id: todo });
  const last = await createIssue(api, member.token, project.id, { name: "Last", state_id: todo });
  const other = await createIssue(api, member.token, project.id, { name: "Other", state_id: states.Backlog });
  await updateIssue(api, member.token, first.id, { sort_order: 100 });
  await updateIssue(api, member.token, last.id, { sort_order: 300 });
  await updateIssue(api, member.token, other.id, { sort_order: 90000 });
  const added = await createIssue(api, member.token, project.id, { name: "  Quick add  ", state_id: todo });
  expect(added).toMatchObject({ name: "Quick add", state_id: todo, sort_order: 10300, sequence_id: 4 });
  await expectIssueCreated(db, added.id, {
    ...madeBy(member.email, "Quick add"),
    sequence_id: 4,
    state: "Todo",
    sort_order: 10300,
    by: member.email,
    activity: createdFor(),
  });
  expectIssuePage(await listIssues(api, member.token, project.id, { state_id: [todo], order_by: "sort_order" }), [
    first.id,
    last.id,
    added.id,
  ]);
  expect(await issueCreated(api, member.token, project.id, { name: "   ", state_id: todo })).toEqual({
    status: 422,
    code: "validation_failed",
    errors: [{ field: "name", code: "too_short" }],
  });
});
