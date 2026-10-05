import {
  addProjectMembers,
  amidAnotherWorkspace,
  answer,
  createProject,
  createState,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type State,
  type StateCreate,
  type StateUpdate,
} from "../../fixtures/api";
import { expectStates, statesOfANewProject, type StateRow } from "../../fixtures/assert/project";
import { accountId, bearer, createPAT, emailFor, register } from "../../fixtures/auth";
import { expect, test } from "../../fixtures/test";

// P6, a project's states (M3 design 2, 3.17): creating one, changing it,
// making it the default and deleting one, with what the default and every
// group keep. The page version comes with the states' settings page (P11).

/** The states of the project of projectId that the caller of token lists, in their order. */
async function listStates(api: Api, token: string, projectId: string): Promise<State[]> {
  const { data, error, response } = await api.GET("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: projectId } },
    headers: bearer(token),
  });
  expect(response.status, `list ${projectId}'s states: ${JSON.stringify(error)}`).toBe(200);
  return data?.data ?? [];
}

/** The states of the workspace of slug that the caller of token lists, each as its project's id and its name, in their order. */
async function listWorkspaceStates(api: Api, token: string, slug: string): Promise<[string, string][]> {
  const { data, error, response } = await api.GET("/api/v0/workspaces/{slug}/states", {
    params: { path: { slug } },
    headers: bearer(token),
  });
  expect(response.status, `list ${slug}'s states: ${JSON.stringify(error)}`).toBe(200);
  return (data?.data ?? []).map((s) => [s.project_id, s.name]);
}

/** The writes on a state named by its id, each by the caller of token. */
function writes(api: Api) {
  return {
    update: async (token: string, stateId: string, body: StateUpdate) => {
      const { error, response } = await api.PATCH("/api/v0/states/{state_id}", {
        params: { path: { state_id: stateId } },
        body,
        headers: bearer(token),
      });
      return answer(response, error);
    },
    remove: async (token: string, stateId: string) => {
      const { error, response } = await api.DELETE("/api/v0/states/{state_id}", {
        params: { path: { state_id: stateId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
    markDefault: async (token: string, stateId: string) => {
      const { error, response } = await api.POST("/api/v0/states/{state_id}/mark-default", {
        params: { path: { state_id: stateId } },
        headers: bearer(token),
      });
      return answer(response, error);
    },
  };
}

/** rows with the state of name changed as to says. */
function changed(rows: StateRow[], name: string, to: Partial<StateRow>): StateRow[] {
  return rows.map((s) => (s.name === name ? { ...s, ...to } : s));
}

test("P6 (API): an admin of a project creates Review in the started group at 70000, after its states but the triage state; another admin changes its color and moves it before In Progress, and the first makes it the default, Backlog the default no longer; deleting the default, deleting the only state of a group or moving it to another, a triage state and a member's change are refused, each changing nothing; In Progress is deleted, Review left in its group; In Progress is found no more, and the triage state is not found; a member lists the states but the triage state, an archived project lists none, and the workspace's list leaves out the triage states and the archived project's", async ({
  api,
  db,
}, testInfo) => {
  const account = async (label: string) => {
    const email = emailFor(testInfo, label);
    const token = (await createPAT(api, (await register(api, email)).access_token)).token;
    return { email, token, id: await accountId(api, token) };
  };
  const admin = await account("admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  // ann and mem, acme's members: ann Web's other admin, mem its member.
  const [ann, mem] = await Promise.all(["ann", "mem"].map(account));
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
  const { update, remove, markDefault } = writes(api);
  const made = await listStates(api, ann.token, web.id);
  const [backlog, inProgress] = ["Backlog", "In Progress"].map((name) => made.find((s) => s.name === name)?.id);
  if (!backlog || !inProgress) {
    throw new Error("Web was made without Backlog or In Progress");
  }
  // Web's states as expectStates reads them, each as Web's creation made it, the admin's.
  let states = statesOfANewProject(admin.email);
  await expectStates(db, web.id, states);

  // ann creates Review in the started group: after Web's states, the triage state's 65000 left out, at 55000 + 15000;
  // not the default.
  const review = await createState(api, ann.token, web.id, { name: "Review", color: "#8B5CF6", group: "started" });
  expect([review.name, review.group, review.sequence, review.default], "Review as created").toEqual([
    "Review",
    "started",
    70000,
    false,
  ]);
  states = [
    ...states,
    {
      name: "Review",
      color: "#8B5CF6",
      group: "started",
      sequence: 70000,
      default: false,
      deleted: false,
      by: ann.email,
    },
  ];
  await expectStates(db, web.id, states);

  // The admin changes its color, its sequence kept, then moves it between Todo (25000) and In Progress (35000), as a
  // drag does, its color kept.
  expect(await update(admin.token, review.id, { color: "#3E63DD" }), "the admin's change of Review's color").toEqual({
    status: 200,
  });
  states = changed(states, "Review", { color: "#3E63DD", by: admin.email });
  await expectStates(db, web.id, states);
  expect(await update(admin.token, review.id, { sequence: 30000 }), "the admin's move of Review").toEqual({
    status: 200,
  });
  states = changed(states, "Review", { sequence: 30000 });
  await expectStates(db, web.id, states);

  // ann makes Review the default: Backlog is the default no longer, both written by her; Web has one default.
  expect(await markDefault(ann.token, review.id), "ann makes Review the default").toEqual({ status: 204 });
  states = changed(changed(states, "Backlog", { default: false, by: ann.email }), "Review", {
    default: true,
    by: ann.email,
  });
  await expectStates(db, web.id, states);

  // Refused, each changing nothing of what was just read: ann's deletion of Review, the default; her deletion of
  // Backlog, its group's only state, and her move of it to the started group, which has two, Review and In Progress,
  // its own group counted; her creation of a state in the triage group; mem's change of Review, a member's.
  const triage = await api.POST("/api/v0/projects/{project_id}/states", {
    params: { path: { project_id: web.id } },
    body: { name: "Intake", color: "#4E5355", group: "triage" } as unknown as StateCreate,
    headers: bearer(ann.token),
  });
  expect(
    [
      await remove(ann.token, review.id),
      await remove(ann.token, backlog),
      await update(ann.token, backlog, { group: "started" }),
      answer(triage.response, triage.error),
      await update(mem.token, review.id, { name: "Checked" }),
    ],
    "ann's deletions of Review and of Backlog, her move of Backlog, her triage state, mem's change of Review"
  ).toEqual([
    { status: 409, code: "project.state_default" },
    { status: 409, code: "project.state_last_in_group" },
    { status: 409, code: "project.state_last_in_group" },
    { status: 422, code: "validation_failed", errors: [{ field: "group", code: "not_allowed" }] },
    { status: 403, code: "forbidden" },
  ]);
  await expectStates(db, web.id, states);

  // ann deletes In Progress, which the admin wrote last: the started group keeps Review.
  expect(await remove(ann.token, inProgress), "ann deletes In Progress").toEqual({ status: 204 });
  states = changed(states, "In Progress", { deleted: true, by: ann.email });
  await expectStates(db, web.id, states);

  // Not found, changing nothing: ann's deletion of In Progress again, and her renaming of Web's triage state, which no
  // state operation sees.
  const [triageState] = await db.query<{ id: string }>(
    `SELECT id FROM states WHERE project_id = $1 AND "group" = 'triage' AND deleted_at IS NULL`,
    [web.id]
  );
  if (!triageState) {
    throw new Error("Web was made without its triage state");
  }
  expect(
    [await remove(ann.token, inProgress), await update(ann.token, triageState.id, { name: "Intake" })],
    "ann's deletion of In Progress again, her renaming of the triage state"
  ).toEqual([
    { status: 404, code: "project.state_not_found" },
    { status: 404, code: "project.state_not_found" },
  ]);
  await expectStates(db, web.id, states);

  // mem lists Web's states: by sequence, the triage state and the deleted In Progress left out.
  expect(
    (await listStates(api, mem.token, web.id)).map((s) => [s.name, s.default]),
    "Web's states as mem lists them"
  ).toEqual([
    ["Backlog", false],
    ["Todo", false],
    ["Review", true],
    ["Done", false],
    ["Cancelled", false],
  ]);

  // Ops, ann's project of acme: acme's states as she lists them are Web's and Ops's, by project id, no triage state
  // among them, until she archives Ops; then Ops lists none, and acme's are Web's.
  const ops = await createProject(api, ann.token, slug, { name: "Ops", identifier: "OPS" });
  const webStates = ["Backlog", "Todo", "Review", "Done", "Cancelled"].map((name): [string, string] => [web.id, name]);
  const opsStates = ["Backlog", "Todo", "In Progress", "Done", "Cancelled"].map((name): [string, string] => [
    ops.id,
    name,
  ]);
  expect(await listWorkspaceStates(api, ann.token, slug), "acme's states as ann lists them").toEqual(
    web.id < ops.id ? [...webStates, ...opsStates] : [...opsStates, ...webStates]
  );
  const archived = await api.POST("/api/v0/projects/{project_id}/archive", {
    params: { path: { project_id: ops.id } },
    headers: bearer(ann.token),
  });
  expect(archived.response.status, `ann archives Ops: ${JSON.stringify(archived.error)}`).toBe(200);
  expect(await listStates(api, ann.token, ops.id), "the states of Ops, archived").toEqual([]);
  expect(await listWorkspaceStates(api, ann.token, slug), "acme's states, Ops archived").toEqual(webStates);

  // The admin's projects of Elsewhere, made around Web, have their states as they were made: no write on Web's
  // states wrote another project's.
  const elsewhere = await api.GET("/api/v0/workspaces/{slug}/projects", {
    params: { path: { slug: slugFor(testInfo, "elsewhere") } },
    headers: bearer(admin.token),
  });
  expect(elsewhere.data?.data.length, `Elsewhere's projects: ${JSON.stringify(elsewhere.error)}`).toBe(2);
  await Promise.all(
    (elsewhere.data?.data ?? []).map((project) => expectStates(db, project.id, statesOfANewProject(admin.email)))
  );
});
