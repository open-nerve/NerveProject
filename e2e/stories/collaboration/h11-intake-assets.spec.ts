import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { imageHTML } from "../../fixtures/assert/storage";
import { login, newAccount } from "../../fixtures/auth";
import { intakeIssueUpdated, submitIntakeIssue, triageIntakeIssue } from "../../fixtures/intake";
import {
  fileCompleted,
  fileDeleted,
  fileMetadata,
  filePixel,
  filePrepared,
  issueAttachmentIds,
  prepareFile,
  readFile,
  storedFile,
  uploadFile,
} from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

// H11's images and attachments of a work item in a project's intake (M7 design 2, 3.6; P2c spec 1, 2.2, 2.5, 4): the
// API version. While a work item is in the intake its submitter writes them, a guest too, and the project's admins:
// gus, a guest of Web whom it shows only his own work items, uploads an image to the one he submitted and binds it by
// his change of its description, and adds an attachment, prepared, uploaded and completed; he, alice, Web's admin, and
// bob, its member, list and read them. bob, who did not submit it, may write neither (forbidden). Declined, the work
// item takes no more writes, alice's neither (issue.intake_closed), and reads as before. Accepted, issue.update
// decides: gus may write no more, bob may. With Web's intake turned off, a pending work item takes no writes, alice's
// neither (issue.intake_disabled). A personal access token and a browser session answer alike. The duplicate's target,
// the other roles and refusals and their order, a snooze whose time has come, comments, the locks and the
// interleavings are the bootstrap tests' and the permission matrix's.

const closed = { status: 409, code: "issue.intake_closed" };
const off = { status: 409, code: "issue.intake_disabled" };
const forbidden = { status: 403, code: "forbidden" };

test("H11 (API): Web's guest uploads an image to the work item he submitted to its intake, binds it by his change of its description, and adds an attachment, which he, its admin and a member list and read; the member, who did not submit it, may write neither; declined, it takes no more writes, its admin's neither, and reads as before; accepted, its guest submitter may write no more; with the intake turned off, a pending one takes no writes; a personal access token and a browser session alike", async ({
  api,
}, testInfo) => {
  const [alice, bob, gus] = await Promise.all([
    newAccount(api, testInfo, "alice"),
    newAccount(api, testInfo, "bob"),
    newAccount(api, testInfo, "gus"),
  ]);
  const slug = slugFor(testInfo);
  await createWorkspace(api, alice.token, { name: "Acme", slug });
  await inviteAndAccept(api, alice.token, slug, bob, 15);
  await inviteAndAccept(api, alice.token, slug, gus, 5);
  // Web, of which bob is a member and gus a guest, shows its guests only their own work items, as a project does
  // unless its admin says otherwise; its intake is turned on.
  const web = await createProject(api, alice.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, alice.token, web.id, [
    { member_id: bob.id, role: 15 },
    { member_id: gus.id, role: 5 },
  ]);
  expect(
    await changeProject(api, alice.token, web.id, { intake_view: true }),
    "Web's intake turned on by alice"
  ).toMatchObject({ status: 200, project: { intake_view: true } });
  // newAccount's token is its personal access token; each one's browser session besides.
  const session = async ({ email }: { email: string }) => (await login(api, email)).access_token;
  const [aliceSession, bobSession, gusSession] = await Promise.all([session(alice), session(bob), session(gus)]);

  // gus submits Login breaks, which alice will decline, Sign up fails, which she will accept, and Login is slow,
  // pending when she turns the intake off.
  const breaks = await submitIntakeIssue(api, gus.token, web.id, { name: "Login breaks" });
  const signup = await submitIntakeIssue(api, gusSession, web.id, { name: "Sign up fails" });
  const slow = await submitIntakeIssue(api, gus.token, web.id, { name: "Login is slow" });
  const log = Buffer.from("TypeError: session is undefined");
  const aLog = (id: string) =>
    ({
      purpose: "ISSUE_ATTACHMENT",
      scope: { kind: "issue", id },
      name: "console.log",
      size: log.length,
      content_type: "text/plain",
    }) as const;

  // gus uploads a screenshot to Login breaks with his personal access token: prepared, uploaded and completed, it is a
  // preparation of his until his change of the description, with his browser session, binds it. With his browser
  // session he adds the console's log, an attachment once completed.
  const screenshot = await storedFile(
    api,
    gus.token,
    "ISSUE_DESCRIPTION",
    { kind: "issue", id: breaks.id },
    filePixel,
    "image/png",
    "screenshot.png"
  );
  expect(
    await intakeIssueUpdated(api, gusSession, breaks.id, { description_html: imageHTML(screenshot.id) }),
    "gus's change of Login breaks' description"
  ).toMatchObject({ status: 200, issue: { id: breaks.id } });
  expect(
    await fileMetadata(api, gus.token, screenshot.id),
    "the screenshot, once the description has it"
  ).toMatchObject({ is_bound: true });
  const attachment = await storedFile(
    api,
    gusSession,
    "ISSUE_ATTACHMENT",
    aLog(breaks.id).scope,
    log,
    "text/plain",
    "console.log"
  );
  // gus prepares and uploads the network's trace, which he completes once alice has declined Login breaks.
  const trace = await prepareFile(api, gus.token, { ...aLog(breaks.id), name: "network.har" });
  await uploadFile(trace.upload, log);

  // gus, alice and bob, each with either credential, list Login breaks' attachment and read it and the screenshot.
  const readers = [
    ["gus", gus.token],
    ["gus's session", gusSession],
    ["alice", alice.token],
    ["alice's session", aliceSession],
    ["bob", bob.token],
    ["bob's session", bobSession],
  ] as const;
  const readAsBefore = (when: string) =>
    Promise.all(
      readers.map(async ([who, token]) => {
        expect(await issueAttachmentIds(api, token, breaks.id), `Login breaks' attachments ${when}, as ${who}`).toEqual(
          [attachment.id]
        );
        expect(await readFile(api, token, attachment.id), `the log ${when}, as ${who}`).toEqual(log);
        expect(await readFile(api, token, screenshot.id), `the screenshot ${when}, as ${who}`).toEqual(filePixel);
      })
    );
  await readAsBefore("as gus added them");

  // bob, who did not submit Login breaks, may neither add an attachment to it nor delete gus's.
  expect(await filePrepared(api, bob.token, aLog(breaks.id)), "bob's attachment to Login breaks").toEqual(forbidden);
  expect(await fileDeleted(api, bobSession, attachment.id), "bob's deletion of gus's log").toEqual(forbidden);
  await readAsBefore("after bob's refused writes");

  // alice declines Login breaks: it takes no more writes, hers neither, and reads as before.
  await triageIntakeIssue(api, alice.token, breaks.id, { status: "declined" });
  expect(await filePrepared(api, gus.token, aLog(breaks.id)), "gus's attachment to Login breaks, declined").toEqual(
    closed
  );
  expect(
    await filePrepared(api, aliceSession, aLog(breaks.id)),
    "alice's attachment to Login breaks, declined"
  ).toEqual(closed);
  expect(await fileCompleted(api, gusSession, trace.asset.id), "gus's completion of the trace, declined").toEqual(
    closed
  );
  expect(await fileDeleted(api, alice.token, attachment.id), "alice's deletion of gus's log, declined").toEqual(closed);
  await readAsBefore("once declined");

  // alice accepts Sign up fails: issue.update decides, which gus, a guest, is refused and bob is given.
  await triageIntakeIssue(api, alice.token, signup.id, { status: "accepted" });
  expect(await filePrepared(api, gusSession, aLog(signup.id)), "gus's attachment to Sign up fails, accepted").toEqual(
    forbidden
  );
  expect(await filePrepared(api, bob.token, aLog(signup.id)), "bob's attachment to Sign up fails, accepted").toEqual({
    status: 201,
  });

  // gus prepares a log on Login is slow, pending; alice turns Web's intake off: it takes no more writes, hers
  // neither.
  const draft = await prepareFile(api, gus.token, aLog(slow.id));
  expect(
    await changeProject(api, alice.token, web.id, { intake_view: false }),
    "Web's intake turned off by alice"
  ).toMatchObject({ status: 200, project: { intake_view: false } });
  expect(
    await filePrepared(api, gus.token, aLog(slow.id)),
    "gus's attachment to Login is slow, the intake off"
  ).toEqual(off);
  expect(
    await filePrepared(api, aliceSession, aLog(slow.id)),
    "alice's attachment to Login is slow, the intake off"
  ).toEqual(off);
  expect(
    await fileDeleted(api, gusSession, draft.asset.id),
    "gus's deletion of his preparation, the intake off"
  ).toEqual(off);
});
