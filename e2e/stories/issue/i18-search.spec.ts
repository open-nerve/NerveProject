import { readFileSync } from "node:fs";
import type { TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMembershipOf,
  projectMemberWrites,
  slugFor,
  type Api,
} from "../../fixtures/api";
import { expectIssueCollection } from "../../fixtures/assert/issue";
import { newAccount } from "../../fixtures/auth";
import { instantKey } from "../../fixtures/instant";
import {
  createIssue,
  createIssueRelations,
  issueRelationDeleted,
  searchIssues,
  updateIssue,
} from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// Every precondition is an API write; selector pages and command UI belong to P15.
async function searchWorld(api: Api, info: TestInfo) {
  const admin = await newAccount(api, info, "admin");
  const member = await newAccount(api, info, "member");
  const slug = slugFor(info);
  await createWorkspace(api, admin.token, { name: "Search stories", slug });
  await inviteAndAccept(api, admin.token, slug, member, 15);
  return { admin, member, slug };
}

test("I18 (API): visible same-sequence issues rank before title hits; hidden higher scores cannot consume the limit, and shared Unicode identifier cases select the exact visible project", async ({
  api,
}, info) => {
  const { admin, member, slug } = await searchWorld(api, info);
  const fixture = JSON.parse(
    readFileSync(new URL("../../../api/testdata/identifier-fold.json", import.meta.url), "utf8")
  ) as { cases: { name: string; identifiers: string[]; asked: string; want: string | null }[] };
  const folded = fixture.cases.find((row) => row.name === "I and İ stored, ı asked: I, its upper case");
  const dotted = fixture.cases.find((row) => row.name === "I and İ stored, İ asked");
  if (!folded || !dotted) throw new Error("the shared Unicode identifier examples are missing");
  const projects: Awaited<ReturnType<typeof createProject>>[] = [];
  await folded.identifiers.reduce(async (before, identifier) => {
    await before;
    const project = await createProject(api, admin.token, slug, { name: identifier, identifier, network: 0 });
    await addProjectMembers(api, admin.token, project.id, [{ member_id: member.id, role: 15 }]);
    projects.push(project);
  }, Promise.resolve());
  const [plain, dottedProject] = projects;
  if (!plain || !dottedProject) throw new Error("missing visible projects");
  const a = await createIssue(api, member.token, plain.id, { name: "Orbit release" });
  const b = await createIssue(api, member.token, dottedProject.id, { name: "Orbit release" });
  expect(a.sequence_id).toBe(b.sequence_id);
  expect(instantKey(a.created_at) < instantKey(b.created_at)).toBe(true);
  const titleHit = await createIssue(api, member.token, plain.id, { name: "1" });
  const hidden = await createProject(api, admin.token, slug, { name: "Hidden", identifier: "SECRET", network: 0 });
  await createIssue(api, admin.token, hidden.id, { name: "1" });
  await createIssue(api, admin.token, hidden.id, { name: "Orbit" });
  // Equal titles have equal similarity; their known creation order settles the numeric branch.
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "1" }), [b.id, a.id, titleHit.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "1", limit: 1 }), [b.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "Orbit", limit: 1 }), [b.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "1", project_id: plain.id }), [
    a.id,
    titleHit.id,
  ]);
  expect(plain.identifier).toBe(folded.want);
  expect(dottedProject.identifier).toBe(dotted.want);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: `${folded.asked}-${a.sequence_id}` }), [a.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: `${dotted.asked}-${b.sequence_id}` }), [b.id]);
});

test("I18 (API): own-only hides another author's matching issue and parent; parent and sub-issue selectors traverse a five-level tree through that hidden intermediate node", async ({
  api,
}, info) => {
  const { admin, member, slug } = await searchWorld(api, info);
  const project = await createProject(api, admin.token, slug, { name: "Tree", identifier: "TREE" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: member.id, role: 15 }]);
  expect((await changeProject(api, admin.token, project.id, { guest_view_all_features: false })).status).toBe(200);
  const root = await createIssue(api, member.token, project.id, { name: "Tree" });
  const source = await createIssue(api, member.token, project.id, { name: "Tree", parent_id: root.id });
  const hidden = await createIssue(api, admin.token, project.id, { name: "Tree", parent_id: source.id });
  const deep = await createIssue(api, member.token, project.id, { name: "Tree", parent_id: hidden.id });
  const leaf = await createIssue(api, member.token, project.id, { name: "Tree", parent_id: deep.id });
  const allowed = await createIssue(api, member.token, project.id, { name: "Tree" });
  await createIssue(api, admin.token, project.id, { name: "Tree" });
  expect(
    await projectMemberWrites(api, project.id).change(
      admin.token,
      await projectMembershipOf(api, admin.token, project.id, member.id),
      5
    )
  ).toEqual({ status: 200 });
  await changeRole(api, admin.token, slug, member.id, 5);
  const all = await searchIssues(api, member.token, slug, { q: "Tree" });
  expectIssueCollection(all, [allowed.id, leaf.id, deep.id, source.id, root.id]);
  expect(all.data.find((row) => row.id === deep.id)?.parent_id).toBeNull();
  expect(all.data.find((row) => row.id === source.id)?.parent_id).toBe(root.id);
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { q: "Tree", for: "parent", issue_id: source.id }),
    [allowed.id]
  );
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { q: "Tree", for: "sub_issue", issue_id: leaf.id }),
    [allowed.id]
  );
  // These otherwise matching numeric/identifier branches cannot bypass the exclusions either.
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { q: `TREE-${leaf.sequence_id}`, for: "parent", issue_id: source.id }),
    []
  );
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { q: `${root.sequence_id}`, for: "sub_issue", issue_id: leaf.id }),
    []
  );
});

test("I18 (API): relation selection excludes every live stored type in both directions before limit and returns a target again after its relation is deleted", async ({
  api,
}, info) => {
  const { admin, member, slug } = await searchWorld(api, info);
  const project = await createProject(api, admin.token, slug, { name: "Relations", identifier: "REL" });
  const other = await createProject(api, admin.token, slug, { name: "Other", identifier: "OTHER" });
  await [project, other].reduce(async (before, p) => {
    await before;
    await addProjectMembers(api, admin.token, p.id, [{ member_id: member.id, role: 15 }]);
  }, Promise.resolve());
  const source = await createIssue(api, member.token, project.id, { name: "Related" });
  const allowed = await createIssue(api, member.token, other.id, { name: "Related" });
  const relations: {
    relation: Awaited<ReturnType<typeof createIssueRelations>>[number];
    target: Awaited<ReturnType<typeof createIssue>>;
  }[] = [];
  const directions = (["relates_to", "duplicate", "blocked_by"] as const).flatMap((relation_type) =>
    [false, true].map((reverse) => ({ relation_type, reverse }))
  );
  await directions.reduce(async (before, { relation_type, reverse }) => {
    await before;
    const target = await createIssue(api, member.token, other.id, { name: "Related" });
    // blocked_by is stored as requested; unlike blocking it does not swap endpoints.
    const [relation] = await createIssueRelations(api, member.token, reverse ? target.id : source.id, {
      relation_type,
      issue_ids: [reverse ? source.id : target.id],
    });
    if (!relation) throw new Error("the new relation is missing");
    relations.push({ relation, target });
  }, Promise.resolve());
  const query = { q: "Related", for: "relation", issue_id: source.id };
  expectIssueCollection(await searchIssues(api, member.token, slug, query), [allowed.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { ...query, limit: 1 }), [allowed.id]);
  const restored = relations.at(-1)!;
  expect(await issueRelationDeleted(api, member.token, restored.relation.id)).toEqual({ status: 204 });
  expectIssueCollection(await searchIssues(api, member.token, slug, query), [restored.target.id, allowed.id]);
});

test("I18 (API): two- and three-character Chinese contains queries and literal symbols work; empty words use updated order and date none independently", async ({
  api,
}, info) => {
  const { admin, member, slug } = await searchWorld(api, info);
  const project = await createProject(api, admin.token, slug, { name: "Words", identifier: "WORDS" });
  await addProjectMembers(api, admin.token, project.id, [{ member_id: member.id, role: 15 }]);
  const short = await createIssue(api, member.token, project.id, { name: "中文" });
  const long = await createIssue(api, member.token, project.id, { name: "中文词" });
  const dated = await createIssue(api, member.token, project.id, {
    name: "Dated",
    start_date: "2026-10-01",
    target_date: "2026-10-02",
  });
  const literal = await createIssue(api, member.token, project.id, { name: "literal %_\\" });
  const descriptionOnly = await createIssue(api, member.token, project.id, {
    name: "Other",
    description_html: "<p>中文词 literal %_\\</p>",
    target_date: "2026-10-03",
  });
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "中文" }), [short.id, long.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "中文词" }), [long.id]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "%_\\" }), [literal.id]);
  const changed = await updateIssue(api, member.token, short.id, { priority: "high" });
  expect(instantKey(changed.updated_at) > instantKey(descriptionOnly.updated_at)).toBe(true);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: " \t " }), [
    short.id,
    descriptionOnly.id,
    literal.id,
    dated.id,
    long.id,
  ]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { q: "", target_date: ["none"] }), [
    short.id,
    literal.id,
    long.id,
  ]);
  expectIssueCollection(await searchIssues(api, member.token, slug, { start_date: ["none"] }), [
    short.id,
    descriptionOnly.id,
    literal.id,
    long.id,
  ]);
  expectIssueCollection(
    await searchIssues(api, member.token, slug, { target_date: ["2026-10-01..2026-10-31", "..2026-10-02"], limit: 1 }),
    [dated.id]
  );
});
