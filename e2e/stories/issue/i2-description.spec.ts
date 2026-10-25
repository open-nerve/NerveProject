import { addProjectMembers, createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectDescriptionActivity } from "../../fixtures/assert/issue-collaboration";
import { newAccount } from "../../fixtures/auth";
import { activityPage, createIssue, descriptionPage, descriptionVersion, updateIssue } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// These description cases extend I2 without adding a second activity expectation shape.
test("I2 (API): ordinary title/description writes audit in order, repeated saves merge and skip block-id saves preserve audit/version stamps", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const ann = await newAccount(api, testInfo, "ann");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  await inviteAndAccept(api, admin.token, slug, ann, 15);
  const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: ann.id, role: 15 }]);
  const issue = await createIssue(api, admin.token, project.id, { name: "Draft", description_html: "<p>First</p>" });
  const before = await descriptionPage(api, admin.token, issue.id);
  const renamed = await updateIssue(api, ann.token, issue.id, { name: "Final", description_html: "<p>Second</p>" });
  const firstSave = await descriptionPage(api, admin.token, issue.id);
  expect(firstSave.data).toHaveLength(2);
  expect(firstSave.data[0]?.id).not.toBe(before.data[0]?.id);
  expect((await activityPage(api, ann.token, issue.id, undefined, 100)).data.map((row) => row.field)).toEqual([
    null,
    "name",
    "description",
  ]);
  const changed = await updateIssue(api, ann.token, issue.id, { description_html: "<p>Third</p>" });
  await expectDescriptionActivity(db, issue.id, issue.description_html, changed.description_html);
  const merged = await descriptionPage(api, admin.token, issue.id);
  expect(merged.data.map((row) => row.id)).toEqual(firstSave.data.map((row) => row.id));
  expect(changed.updated_at > renamed.updated_at).toBe(true);
  const activity = await activityPage(api, ann.token, issue.id, undefined, 100);
  const version = merged.data[0];
  if (!version) throw new Error("no merged version");
  expect((await descriptionVersion(api, admin.token, version.id)).description_html).toBe("<p>Third</p>");
  const saved = await updateIssue(api, admin.token, issue.id, {
    description_html: '<p data-id="block-123">Third</p>',
    skip_activity: true,
  });
  expect(saved.description_html).toBe('<p data-id="block-123">Third</p>');
  expect(saved.updated_at).toBe(changed.updated_at);
  expect(saved.updated_by_id).toBe(changed.updated_by_id);
  expect(await activityPage(api, ann.token, issue.id, undefined, 100)).toEqual(activity);
  expect(await descriptionPage(api, admin.token, issue.id)).toEqual(merged);
  expect((await descriptionVersion(api, admin.token, version.id)).description_html).toBe("<p>Third</p>");
});

for (const property of ["href", "checked"] as const) {
  test(`I2 (API): ${property}-only content changes audit normally even with skip_activity`, async ({
    api,
    db,
  }, testInfo) => {
    const admin = await newAccount(api, testInfo, "admin");
    const slug = slugFor(testInfo);
    await createWorkspace(api, admin.token, { name: "Acme", slug });
    const project = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
    const beforeHtml = '<p><a href="https://example.test/a">Link</a><input type="checkbox"></p>';
    const nextHtml =
      property === "href"
        ? beforeHtml.replace("/a", "/b")
        : beforeHtml.replace('type="checkbox"', 'type="checkbox" checked');
    const issue = await createIssue(api, admin.token, project.id, {
      name: "Content identity",
      description_html: beforeHtml,
    });
    const initial = await descriptionPage(api, admin.token, issue.id);
    const changed = await updateIssue(api, admin.token, issue.id, { description_html: nextHtml, skip_activity: true });
    expect(changed.updated_at > issue.updated_at).toBe(true);
    await expectDescriptionActivity(db, issue.id, issue.description_html, changed.description_html);
    const versions = await descriptionPage(api, admin.token, issue.id);
    expect(versions.data.map((row) => row.id)).toEqual(initial.data.map((row) => row.id));
    const latest = versions.data[0];
    if (!latest) throw new Error("no latest description version");
    expect(latest.last_saved_at > (initial.data[0]?.last_saved_at ?? "")).toBe(true);
    expect((await descriptionVersion(api, admin.token, latest.id)).description_html).toBe(changed.description_html);
  });
}
