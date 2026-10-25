import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { newAccount } from "../../fixtures/auth";
import {
  createIssue,
  descriptionPage,
  descriptionVersion,
  issueArchived,
  issueUpdated,
  updateIssue,
} from "../../fixtures/issues";
import { stateIds } from "../../fixtures/states";
import { instantKey } from "../../fixtures/instant";
import { expect, test } from "../../fixtures/test";

// I13 creates independent history by changing author, without manipulating time.
test("I13 (API): versions paginate newest first and details restore through ordinary PATCH; same-author saves merge and archived issues refuse restoration", async ({
  api,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const ann = await newAccount(api, testInfo, "ann");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, ann, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: ann.id, role: 15 }]);
  const issue = await createIssue(api, admin.token, project.id, {
    name: "History",
    description_html: "<p>Initial</p>",
  });
  const initial = (await descriptionPage(api, admin.token, issue.id)).data[0];
  expect(initial).toMatchObject({ issue_id: issue.id, owned_by_id: admin.id });
  if (!initial) throw new Error("missing initial version");
  await updateIssue(api, ann.token, issue.id, { description_html: "<p>Ann draft</p>" });
  const draft = (await descriptionPage(api, admin.token, issue.id)).data[0];
  await updateIssue(api, ann.token, issue.id, { description_html: "<p>Ann final</p>" });
  const merged = await descriptionPage(api, admin.token, issue.id);
  expect(merged.data.map((row) => row.id)).toEqual([draft?.id, initial.id]);
  expect(instantKey(merged.data[0]?.last_saved_at ?? "") > instantKey(draft?.last_saved_at ?? "")).toBe(true);
  await updateIssue(api, admin.token, issue.id, { description_html: "<p>Admin final</p>" });
  const first = await descriptionPage(api, ann.token, issue.id, undefined, 1);
  expect(first.next_cursor).not.toBeNull();
  const second = await descriptionPage(api, ann.token, issue.id, first.next_cursor ?? undefined, 1);
  const third = await descriptionPage(api, ann.token, issue.id, second.next_cursor ?? undefined, 1);
  expect(third.next_cursor).toBeNull();
  expect([first.data[0]?.owned_by_id, second.data[0]?.owned_by_id, third.data[0]?.owned_by_id]).toEqual([
    admin.id,
    ann.id,
    admin.id,
  ]);
  expect([first.data[0]?.id, second.data[0]?.id, third.data[0]?.id]).toEqual([
    expect.any(String),
    draft?.id,
    initial.id,
  ]);
  const historical = await descriptionVersion(api, ann.token, initial.id);
  expect(historical).toMatchObject({ ...initial, description_html: "<p>Initial</p>" });
  const beforeRestore = await descriptionPage(api, admin.token, issue.id, undefined, 100);
  expect(
    (await updateIssue(api, admin.token, issue.id, { description_html: historical.description_html })).description_html
  ).toBe(historical.description_html);
  const afterRestore = await descriptionPage(api, admin.token, issue.id, undefined, 100);
  expect(afterRestore.data.map((row) => row.id)).toEqual(beforeRestore.data.map((row) => row.id));
  const restoredVersion = afterRestore.data[0];
  if (!restoredVersion) throw new Error("no restored version");
  expect((await descriptionVersion(api, ann.token, restoredVersion.id)).description_html).toBe(
    historical.description_html
  );
  const { Done: done } = await stateIds(api, admin.token, project.id);
  if (!done) throw new Error("no Done state");
  await updateIssue(api, admin.token, issue.id, { state_id: done });
  expect((await issueArchived(api, admin.token, issue.id)).status).toBe(200);
  expect(await issueUpdated(api, ann.token, issue.id, { description_html: "<p>Ann final</p>" })).toEqual({
    status: 409,
    code: "issue.archived",
  });
  expect(await descriptionPage(api, admin.token, issue.id, undefined, 100)).toEqual(afterRestore);
  expect(await descriptionVersion(api, ann.token, initial.id)).toEqual(historical);
});
