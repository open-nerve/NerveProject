import { randomUUID } from "node:crypto";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import { bearer, login, newAccount } from "../../fixtures/auth";
import { validationFailed } from "../../fixtures/issue-planning";
import {
  createIssue,
  issueDeleted,
  issueRead,
  issuesListed,
  updateIssue,
  workspaceIssuesListed,
  type Issue,
} from "../../fixtures/issues";
import {
  issueDeletion,
  listedNow,
  recentVisitCalls,
  recentVisitRows,
  recordVisit,
  type RecentVisitCreate,
  type RecentVisitRow,
} from "../../fixtures/recent-visits";
import { expect, test } from "../../fixtures/test";

// H17, one's recent visits (M7 design 2, 3.9, 3.10; P3d spec 2.2–2.7, R5–R8, R13): the API version. A member of a
// workspace records his visit of a project and of a work item, each his, in the workspace and the project the server
// finds from its target, made at the moment he visited it; visited again, a target keeps its one row, visited anew, and
// moves to the top. He lists his visits the latest first, of a kind or both, a kind none of the two refused, each with
// its target as it is now. Reading a work item, a project or a list writes no visit. A target he does not see answers
// as one that never was, and writes nothing: a private project he is not in, a public one he reads but is not a member
// of, their work items, and to one who is none of its members, a workspace's. A visit of a target he no longer sees is
// kept and not listed: a guest's of another's work item once the project shows its guests their own alone, and his of
// a project he left, listed again once he joins it. His twenty-first visit in a workspace deletes for good the one he
// visited earliest, not the one made earliest, his visits elsewhere kept. A work item's deletion deletes every
// account's visits of it and of the work items under it, at its moment, by its deleter. The workspace's guest records
// and lists his own. A personal access token and a browser session alike, each round in its own workspace: a visit
// cannot be deleted through the API. The list's ties, the refusals' order, the workspace's admin who is a project's
// guest, intake items, archived targets, the deletion window, the cascade's other paths, the cap's count of hidden
// visits, the interleavings and the locks are the bootstrap tests' (TestRecentVisitsThroughTheApp,
// TestRecentVisitsFollowDeletedWorkItems, TestRecentVisitWritesInterleave, TestRecordingARecentVisitTakesItsLocks) and
// the permission matrix's; the project's and the workspace's deletions are the deletion stories' (furnishIssueRows);
// the page is P7's.

// The icon of Web.
const webLogo = { in_use: "emoji", emoji: { value: "128640" } } as const;

/** The body of a visit of the work item of entity_id. */
function issueOf(entity_id: string): RecentVisitCreate {
  return { entity_type: "issue", entity_id };
}

/** The body of a visit of the project of entity_id. */
function projectOf(entity_id: string): RecentVisitCreate {
  return { entity_type: "project", entity_id };
}

test("H17 (API): bob, acme's member, records his visits of Web and of its work item Fix, of the project the server finds, and visited again Web keeps its row and moves to the top; he lists them of a kind or both, a kind none of the two refused, each target as it is now, and his reads write no visit; Vault and Secret, private, Open and Notice, public but not his, and to olga, none of acme, Web and Fix answer as ids that never were; gus, acme's guest, records and lists his own, his visit of Fix kept and not listed once Web shows its guests their own alone; out of Web bob lists none of his visits of it, kept, until he joins it again; his twenty-first visit deletes for good Fix's, visited earliest, not Web's, made earliest, his visit elsewhere kept; alice's deletion of N1 deletes bob's visits of N1 and of N2 under it and hers of N1 at its moment, by her; with personal access tokens and browser sessions alike", async ({
  api,
  db,
}, testInfo) => {
  // Each account with its personal access token (newAccount) and a browser session besides.
  const signedIn = async (name: string) => {
    const account = await newAccount(api, testInfo, name);
    return { ...account, name, session: (await login(api, account.email)).access_token };
  };
  const [alice, bob, gus, olga] = await Promise.all([
    signedIn("alice"),
    signedIn("bob"),
    signedIn("gus"),
    signedIn("olga"),
  ]);
  type Account = typeof alice;
  // Elsewhere, bob's own workspace, with his project Side, which he visits in each round.
  const elsewhere = await createWorkspace(api, bob.token, { name: "Elsewhere", slug: slugFor(testInfo, "elsewhere") });
  const side = await createProject(api, bob.token, elsewhere.slug, { name: "Side", identifier: "SID" });
  const visits = recentVisitCalls(api);

  // With each account's credential of the kind given, a round of bob's visits in an acme of its own, from his first to
  // his visits of a deleted work item.
  const round = async (credential: "token" | "session") => {
    const as = (account: Account) => account[credential];
    const slug = slugFor(testInfo, credential);
    const acme = await createWorkspace(api, alice.token, { name: "Acme", slug });
    await inviteAndAccept(api, alice.token, slug, bob, 15);
    await inviteAndAccept(api, alice.token, slug, gus, 5);
    // alice's Web, public, with an icon, showing its guests every work item: bob its member, gus its guest; her work
    // item Fix of it, assigned to bob. Vault, her private project, hers alone, with her work item Secret; Open, her
    // public project, which bob reads but is not a member of, with her work item Notice.
    const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB", logo_props: webLogo });
    await addProjectMembers(api, alice.token, web.id, [
      { member_id: bob.id, role: 15 },
      { member_id: gus.id, role: 5 },
    ]);
    const guests = (all: boolean) => changeProject(api, alice.token, web.id, { guest_view_all_features: all });
    expect(await guests(true), "Web shows its guests every work item").toMatchObject({ status: 200 });
    const fix = await createIssue(api, alice.token, web.id, { name: "Fix", assignee_ids: [bob.id] });
    const vault = await createProject(api, alice.token, slug, { name: "Vault", identifier: "VLT", network: 0 });
    const secret = await createIssue(api, alice.token, vault.id, { name: "Secret" });
    const open = await createProject(api, alice.token, slug, { name: "Open", identifier: "OPN" });
    const notice = await createIssue(api, alice.token, open.id, { name: "Notice" });
    await recordVisit(api, as(bob), projectOf(side.id));

    // account's visits in acme as stored.
    const stored = (account: Account) => recentVisitRows(db, account.id, acme.id);
    // account's list in acme, of kind or both, as his undeleted visits of targets are stored, in that order, each with
    // its target as alice reads it now.
    const lists = async (account: Account, targets: string[], kind?: RecentVisitRow["type"]) => {
      const rows = await stored(account);
      const want = await Promise.all(
        targets.map((target) => {
          const row = rows.find((r) => r.identifier === target && r.deleted === null);
          if (!row) {
            throw new Error(`${account.name} has no visit of ${target} to list`);
          }
          return listedNow(api, alice.token, row);
        })
      );
      const listed = await visits.list(as(account), slug, kind);
      expect(listed, `${account.name}'s visits of ${kind ?? "both kinds"}`).toEqual({ status: 200, visits: want });
    };
    // A visit of account's of type and identifier in Web as stored, made at the moment it was visited: row's.
    const fresh = (
      row: RecentVisitRow | undefined,
      account: Account,
      type: RecentVisitRow["type"],
      identifier: string
    ) => ({
      id: row?.id,
      type,
      identifier,
      project: web.id,
      created: row?.created,
      visited: row?.created,
      deleted: null,
      writer: account.id,
      stamped: true,
    });
    // account's visits of a work item and a project that never were, answered.
    const neverWere = async (account: Account) => {
      const absent = randomUUID();
      return {
        issue: await visits.record(as(account), issueOf(absent)),
        project: await visits.record(as(account), projectOf(absent)),
      };
    };

    // A: bob records his visits of Web and of Fix, each one row, his, in acme and Web, made by him at the moment he
    // visited it; he lists them the latest first, each target as it is now.
    expect(
      [await visits.record(as(bob), projectOf(web.id)), await visits.record(as(bob), issueOf(fix.id))],
      "bob records Web and Fix"
    ).toEqual([{ status: 204 }, { status: 204 }]);
    const [webs, fixs] = await stored(bob);
    expect(await stored(bob), "bob's visits as stored").toEqual([
      fresh(webs, bob, "project", web.id),
      fresh(fixs, bob, "issue", fix.id),
    ]);
    await lists(bob, [fix.id, web.id]);

    // B: visited again, Web keeps its one row, its id and its making, visited anew by him, and moves to the top.
    expect(await visits.record(as(bob), projectOf(web.id)), "bob visits Web again").toEqual({ status: 204 });
    const revisited = await stored(bob);
    expect(revisited, "bob's visits, Web revisited").toEqual([{ ...webs, visited: expect.any(String) }, fixs]);
    expect(revisited[0]?.visited, "Web's visit, revisited").not.toEqual(webs?.visited);
    await lists(bob, [web.id, fix.id]);

    // C: of a kind, he lists that kind's alone; a kind none of the two is refused, listed or recorded.
    await lists(bob, [fix.id], "issue");
    await lists(bob, [web.id], "project");
    const kindRefused = validationFailed(["entity_type", "invalid_format"]);
    expect(
      [
        await visits.list(as(bob), slug, "cycle" as never),
        await visits.record(as(bob), { entity_type: "cycle", entity_id: web.id } as never),
      ],
      "bob lists and records a cycle's visit"
    ).toEqual([kindRefused, kindRefused]);

    // D: alice renames Web and Fix, makes Fix urgent and assigns it to herself too: bob lists them as they are now, his
    // visits as they were.
    const renamed = await changeProject(api, alice.token, web.id, { name: "Website" });
    expect(renamed, "Web renamed").toMatchObject({ status: 200 });
    await updateIssue(api, alice.token, fix.id, {
      name: "Fix the login",
      priority: "urgent",
      assignee_ids: [bob.id, alice.id],
    });
    await lists(bob, [web.id, fix.id]);
    expect(await stored(bob), "bob's visits, their targets changed").toEqual(revisited);

    // E: his reads of Fix, by its id and its number, of Web, of the lists of acme's projects, of Web's and acme's work
    // items and of his visits write no visit.
    const read = { headers: bearer(as(bob)) };
    const byNumber = { path: { slug, identifier: `WEB-${fix.sequence_id}` } };
    const ofWeb = { path: { project_id: web.id } };
    expect(
      [
        (await issueRead(api, as(bob), fix.id)).status,
        (await api.GET("/api/v0/workspaces/{slug}/issues/{identifier}", { params: byNumber, ...read })).response.status,
        (await api.GET("/api/v0/projects/{project_id}", { params: ofWeb, ...read })).response.status,
        (await api.GET("/api/v0/workspaces/{slug}/projects", { params: { path: { slug } }, ...read })).response.status,
        (await issuesListed(api, as(bob), web.id)).status,
        (await workspaceIssuesListed(api, as(bob), slug)).status,
        (await visits.list(as(bob), slug)).status,
        (await visits.list(as(bob), slug, "issue")).status,
        (await visits.list(as(bob), slug, "project")).status,
      ],
      "bob's reads"
    ).toEqual(Array(9).fill(200));
    expect(await stored(bob), "bob's visits, after his reads").toEqual(revisited);

    // F: a target he does not see answers as one that never was, and writes nothing: Vault and Secret, private, and
    // Open, which he reads, and Notice, public, his projects none of them; to olga, none of acme, Web and Fix, and acme
    // is no workspace of hers.
    const bobsNever = await neverWere(bob);
    expect(bobsNever, "bob's visits of targets that never were").toEqual({
      issue: { status: 404, code: "issue.not_found" },
      project: { status: 404, code: "project.not_found" },
    });
    const opened = await api.GET("/api/v0/projects/{project_id}", {
      params: { path: { project_id: open.id } },
      ...read,
    });
    expect(opened.response.status, "bob reads Open").toBe(200);
    expect(
      [
        await visits.record(as(bob), projectOf(vault.id)),
        await visits.record(as(bob), issueOf(secret.id)),
        await visits.record(as(bob), projectOf(open.id)),
        await visits.record(as(bob), issueOf(notice.id)),
      ],
      "bob records Vault, Secret, Open and Notice"
    ).toEqual([bobsNever.project, bobsNever.issue, bobsNever.project, bobsNever.issue]);
    const olgasNever = await neverWere(olga);
    expect(
      [
        await visits.record(as(olga), projectOf(web.id)),
        await visits.record(as(olga), issueOf(fix.id)),
        await visits.list(as(olga), slug),
      ],
      "olga records Web and Fix and lists acme's visits"
    ).toEqual([olgasNever.project, olgasNever.issue, { status: 404, code: "workspace.not_found" }]);
    expect([await stored(bob), await stored(olga)], "bob's and olga's visits, the refusals'").toEqual([revisited, []]);

    // G: gus, acme's guest and Web's, records and lists his visits of Web and of alice's Fix. Once Web shows its guests
    // their own work items alone, his visit of Fix is kept and not listed, and Fix answers him as one that never was.
    expect(
      [await visits.record(as(gus), projectOf(web.id)), await visits.record(as(gus), issueOf(fix.id))],
      "gus records Web and Fix"
    ).toEqual([{ status: 204 }, { status: 204 }]);
    await lists(gus, [fix.id, web.id]);
    const gusVisits = await stored(gus);
    expect(await guests(false), "Web shows its guests their own alone").toMatchObject({ status: 200 });
    await lists(gus, [web.id]);
    expect(await visits.record(as(gus), issueOf(fix.id)), "gus records Fix again").toEqual(
      (await neverWere(gus)).issue
    );
    expect(await stored(gus), "gus's visits, Fix's kept").toEqual(gusVisits);

    // H: out of Web, bob lists none of his visits of it, kept, and Web answers him as one that never was; joined again,
    // he lists them as before.
    expect(await projectMemberWrites(api, web.id).leave(as(bob)), "bob leaves Web").toEqual({ status: 204 });
    await lists(bob, []);
    expect(await visits.record(as(bob), projectOf(web.id)), "bob records Web, out of it").toEqual(bobsNever.project);
    expect(await stored(bob), "bob's visits, out of Web").toEqual(revisited);
    const joined = await api.POST("/api/v0/projects/{project_id}/join", { params: ofWeb, ...read });
    expect(joined.response.status, "bob joins Web again").toBe(200);
    await lists(bob, [web.id, fix.id]);

    // I: his twenty-first visit in acme deletes for good his visit visited earliest, Fix's (he visited Web again after
    // it), not Web's, made earliest; he lists his twenty latest, and his visit elsewhere is kept. It counts on his
    // visits in acme being Web's and Fix's alone: since he visited Web again, none of his records succeeded (C–H). The
    // twenty-one are alice's N1 to N19 of Web, N2 under N1, each made and visited in turn.
    const elsewheres = await recentVisitRows(db, bob.id, elsewhere.id);
    const ns: Issue[] = [];
    for (let n = 1; n <= 19; n++) {
      const body = { name: `N${n}`, parent_id: n === 2 ? ns[0]?.id : undefined };
      // eslint-disable-next-line no-await-in-loop -- N2 is made under N1, and the work items are numbered in turn.
      ns.push(await createIssue(api, alice.token, web.id, body));
    }
    for (const n of ns.slice(0, 18)) {
      // eslint-disable-next-line no-await-in-loop -- Each visit is later than the one before it.
      await recordVisit(api, as(bob), issueOf(n.id));
    }
    expect(await stored(bob), "bob's visits in acme before his twenty-first").toHaveLength(20);
    const [n1, n2, n19] = [ns[0], ns[1], ns[18]];
    if (!n1 || !n2 || !n19) {
      throw new Error("N1, N2 and N19 are made");
    }
    expect(await visits.record(as(bob), issueOf(n19.id)), "bob's twenty-first visit").toEqual({ status: 204 });
    const capped = await stored(bob);
    expect(capped, "bob's visits in acme, Fix's deleted for good").toEqual([
      revisited[0],
      ...ns.map((n, at) => fresh(capped[at + 1], bob, "issue", n.id)),
    ]);
    const latestFirst = ns.map((n) => n.id).toReversed();
    await lists(bob, [...latestFirst, web.id]);
    expect(await recentVisitRows(db, bob.id, elsewhere.id), "bob's visits elsewhere").toEqual(elsewheres);

    // J: alice visits N1 and deletes it, and N2 under it: bob's visits of both and hers of N1 are deleted at the
    // deletion's moment, by her, each visited as it was; bob lists neither, and N1 answers him as one that never was.
    await recordVisit(api, as(alice), issueOf(n1.id));
    const [bobs, alices] = [await stored(bob), await stored(alice)];
    expect(await issueDeleted(api, as(alice), n1.id), "alice deletes N1").toEqual({ status: 204 });
    const deletion = await issueDeletion(db, n1.id);
    expect([deletion, await issueDeletion(db, n2.id)], "N1's and N2's deletion").toEqual([
      { at: expect.any(String), by: alice.id },
      deletion,
    ]);
    const deleted = (row: RecentVisitRow) =>
      row.identifier === n1.id || row.identifier === n2.id ? { ...row, deleted: deletion.at, writer: alice.id } : row;
    expect([await stored(bob), await stored(alice)], "bob's and alice's visits, N1 deleted").toEqual([
      bobs.map(deleted),
      alices.map(deleted),
    ]);
    await lists(bob, [...latestFirst.filter((id) => id !== n1.id && id !== n2.id), web.id]);
    expect(await visits.record(as(bob), issueOf(n1.id)), "bob records N1, deleted").toEqual(bobsNever.issue);
  };

  await test.step("with personal access tokens", () => round("token"));
  await test.step("with browser sessions", () => round("session"));
});
