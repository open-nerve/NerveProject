import {
  addProjectMembers,
  answer,
  archiveProject,
  createLabel,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Label,
  type LabelCreate,
  type Project,
} from "../../fixtures/api";
import { bearer, newAccount } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// H13's label catalog, the labels of a workspace's projects at once (M7 P6b spec 2.1, R16): the API version. A member
// of a workspace lists the labels of each of its projects he is an active member of, unarchived, those at the top and
// those under them alike: by project id, then sort order, then id, each project's as its own list lists them. A project
// he sees but is not a member of adds none, nor does an archived one, whose own list still lists its labels. A guest
// of the workspace lists those of the projects he is in. A workspace he is not a member of, or none, is not found. A
// left or deleted membership, a deleted label and the snapshot are the bootstrap tests'
// (TestListingWorkspaceLabelsListsEachProjects, TestWorkspaceLabelsKeepToTheirWorkspace) and the permission matrix's;
// the pages that read it are P6b's.

/** The answer of the caller of token's list of the labels of the workspace of slug: its status, and its labels or its problem's code. */
async function workspaceLabels(api: Api, token: string, slug: string) {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/labels", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  return { ...answer(response, error), labels: data?.data };
}

/** The labels of the project of projectId that the caller of token lists, in their order. */
async function listLabels(api: Api, token: string, projectId: string): Promise<Label[]> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/labels", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  expect(response.status, `list ${projectId}'s labels: ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/** The status of the caller of token's list of the labels of the project of projectId, and its problem's code. */
async function labelsListed(api: Api, token: string, projectId: string) {
  const { error, response } = await api.GET("/api/v0/projects/{project_id}/labels", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  return answer(response, error);
}

/** Moves the label of labelId to sortOrder among its project's, by the caller of token, an admin's of its project. */
async function moveLabel(api: Api, token: string, labelId: string, sortOrder: number): Promise<Label> {
  const { data, error, response } = await api.PATCH("/api/v0/labels/{label_id}", {
    params: { path: { label_id: labelId } },
    body: { sort_order: sortOrder },
    headers: bearer(token),
  });
  expect(response.status, `move ${labelId}: ${JSON.stringify(error)}`).toBe(200);
  if (!data) throw new Error(`moveLabel ${labelId} answered 200 without the label`);
  return data;
}

/** items by their ids, as the catalog takes its projects, and a project's labels of one sort order. */
const byId = <T extends { id: string }>(items: T[]): T[] =>
  items.toSorted((a, b) => Number(a.id > b.id) - Number(a.id < b.id));

/** The names of labels, in their order. */
const names = (labels: Label[] | undefined): string[] | undefined => labels?.map((label) => label.name);

test("H13 (API): walt, acme's member, lists the labels of Alpha, Web and Shelf, his projects, by project, then sort order, then id, a label under another too, each project's as its own list one after another; Open, public, which he sees but is no member of, adds none; once Shelf is archived its labels leave his list, its own list still listing them; gus, acme's guest and Web's, lists Web's alone; alice, the admin, lists Open's too, and beta's labels in beta alone; beta, a workspace not his, and one that does not exist are not found", async ({
  api,
}, testInfo) => {
  const [alice, walt, gus] = await Promise.all(
    ["alice", "walt", "gus"].map((label) => newAccount(api, testInfo, label))
  );
  if (!alice || !walt || !gus) throw new Error("the accounts were not registered");
  const slug = slugFor(testInfo);
  const betaSlug = slugFor(testInfo, "beta");
  await createWorkspace(api, alice.token, { name: "Acme", slug });
  await createWorkspace(api, alice.token, { name: "Beta", slug: betaSlug });
  await inviteAndAccept(api, alice.token, slug, walt, 15);
  await inviteAndAccept(api, alice.token, slug, gus, 5);
  // alice's projects: Alpha, Web and Shelf, walt's too, gus Web's guest; Open, public, of neither; Lab, in beta.
  const [alpha, web, shelf, open, lab] = await Promise.all(
    (
      [
        [slug, "Alpha", "ALP", 0],
        [slug, "Web", "WEB", 0],
        [slug, "Shelf", "SHF", 0],
        [slug, "Open", "OPN", 2],
        [betaSlug, "Lab", "LAB", 0],
      ] as const
    ).map(([where, name, identifier, network]) => createProject(api, alice.token, where, { name, identifier, network }))
  );
  if (!alpha || !web || !shelf || !open || !lab) throw new Error("missing projects");
  const walts = { member_id: walt.id, role: 15 as const };
  await Promise.all([
    addProjectMembers(api, alice.token, alpha.id, [walts]),
    addProjectMembers(api, alice.token, web.id, [walts, { member_id: gus.id, role: 5 }]),
    addProjectMembers(api, alice.token, shelf.id, [walts]),
  ]);

  // Each project's labels one after another, each last of its project's: Alpha's Bug, Feature, and UI under Bug, then
  // Feature moved before Bug; Web's Ops and Infra, Infra moved to Ops' sort order, the two then by id; Shelf's Old;
  // Open's Secret; Lab's Bench.
  const label = (project: Project, body: LabelCreate) => createLabel(api, alice.token, project.id, body);
  const bug = await label(alpha, { name: "Bug" });
  const feature = await label(alpha, { name: "Feature" });
  await label(alpha, { name: "UI", parent_id: bug.id });
  await moveLabel(api, alice.token, feature.id, bug.sort_order - 10000);
  const ops = await label(web, { name: "Ops" });
  const infra = await moveLabel(api, alice.token, (await label(web, { name: "Infra" })).id, ops.sort_order);
  await label(shelf, { name: "Old" });
  await label(open, { name: "Secret" });
  await label(lab, { name: "Bench" });
  const tied = byId([ops, infra]).map((l) => l.name);
  const own: Record<string, string[]> = {
    [alpha.id]: ["Feature", "Bug", "UI"],
    [web.id]: tied,
    [shelf.id]: ["Old"],
    [open.id]: ["Secret"],
  };
  /** The names of the labels of projects, by their ids, each project's in its own order. */
  const ownNames = (projects: Project[]) => byId(projects).flatMap((project) => own[project.id] ?? []);
  /** The labels of projects, by their ids, each as the caller of token lists its own. */
  const listedOneAfterAnother = async (token: string, projects: Project[]) =>
    (await Promise.all(byId(projects).map((project) => listLabels(api, token, project.id)))).flat();

  // walt's: Alpha's, Web's and Shelf's, each in its own order, as their own lists one after another.
  const hisProjects = [alpha, web, shelf];
  const his = await workspaceLabels(api, walt.token, slug);
  expect(his.status, "walt's list").toBe(200);
  expect(names(his.labels), "walt's names").toEqual(ownNames(hisProjects));
  expect(his.labels, "walt's list, as his projects' own").toEqual(await listedOneAfterAnother(walt.token, hisProjects));
  // Open he sees, public, but is no member of: its labels refused him, and none of them in his list.
  expect(await labelsListed(api, walt.token, open.id), "Open's labels, to walt").toEqual({
    status: 403,
    code: "forbidden",
  });

  // gus, acme's guest and Web's: Web's alone.
  const guests = await workspaceLabels(api, gus.token, slug);
  expect([guests.status, names(guests.labels)], "gus' list").toEqual([200, tied]);
  expect(guests.labels, "gus' list, as Web's own").toEqual(await listLabels(api, gus.token, web.id));

  // Shelf archived: its labels leave walt's list, though its own list still lists them.
  await archiveProject(api, alice.token, shelf.id);
  const archived = await workspaceLabels(api, walt.token, slug);
  expect(names(archived.labels), "walt's names, Shelf archived").toEqual(ownNames([alpha, web]));
  expect(archived.labels, "walt's list, Shelf archived").toEqual(await listedOneAfterAnother(walt.token, [alpha, web]));
  expect(names(await listLabels(api, walt.token, shelf.id)), "Shelf's own list, archived").toEqual(["Old"]);

  // alice, acme's admin and a member of each of its projects: Open's too, Shelf's no longer; in beta, Lab's alone.
  expect((await workspaceLabels(api, alice.token, slug)).labels, "alice's list").toEqual(
    await listedOneAfterAnother(alice.token, [alpha, web, open])
  );
  const betas = await workspaceLabels(api, alice.token, betaSlug);
  expect([betas.status, names(betas.labels)], "alice's list of beta").toEqual([200, ["Bench"]]);

  // beta, a workspace walt is not a member of, and one that does not exist: not found.
  const notFound = { status: 404, code: "workspace.not_found" };
  expect(await workspaceLabels(api, walt.token, betaSlug), "beta's labels, to walt").toEqual(notFound);
  expect(await workspaceLabels(api, gus.token, betaSlug), "beta's labels, to gus").toEqual(notFound);
  expect(await workspaceLabels(api, walt.token, slugFor(testInfo, "none")), "no workspace's labels").toEqual(notFound);
});
