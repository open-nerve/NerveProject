/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { changeProject } from "../../fixtures/api";
import { uploadsNamed } from "../../fixtures/assert/intake";
import { expectQuietConsole } from "../../fixtures/browser";
import { chooseAttachment, downloadedExactly, textFile } from "../../fixtures/content-files-page";
import { failedLoads } from "../../fixtures/cycle-pages";
import { modalDialog } from "../../fixtures/dialogs";
import { pageFileStory } from "../../fixtures/file-story";
import { submitIntakeIssue } from "../../fixtures/intake";
import {
  assetOf,
  attachmentsOf,
  confirmation,
  descriptionOf,
  holdUploads,
  intakeFilesWorld,
} from "../../fixtures/intake-files";
import { expectHeaderOf, intakeDetail, intakeRow, selecting } from "../../fixtures/intake-pages";
import { answerTo, holdAnswer } from "../../fixtures/settings-pages";
import { issueAttachmentIds, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

// H11's page version, the attachments of a work item of a project's intake (M7 design 2, 3.6; P5b spec 2.3, 2.8, R9,
// R12): while it is in the intake, its submitter adds them, a guest too, as its admins do (the intake's rule,
// intakeEditableBy, which the detail gives the attachments and the content view's canWrite takes for it); each is
// prepared, sent to the storage, local or S3, and confirmed, which the detail says (ConfirmedContentNotice), and
// downloads as it was sent. A member who did not submit it lists them and downloads them, offered neither a picker nor
// a deletion. A transfer the page began before the intake was turned off is refused at its confirmation
// (issue.intake_disabled), which the picker says; the page does not read the work item again (spec 2.2, a known
// behaviour). A transfer stopped on its way, or left behind as the selection moves, confirms nothing anywhere; each is
// held on its way, its bytes not yet sent (holdUploads). A deleted attachment is gone from the list and from nerve's
// reads; its object stays in the storage, local or S3, until the cleanup job removes it, which a page story cannot
// trigger (the job removes it only once its safety deadlines and business days have passed on nerve's clock): M5's
// files/f15-cleanup.spec.ts covers it on a clock world. The API version is h11-intake-assets.spec.ts; the attachments'
// paging, drops and lost answers are M5's F8 (R12).

pageFileStory(
  "H11 (page): gus, a guest of Web, attaches a file to the work item he submitted to its intake: prepared for it, sent to the storage and confirmed, which its detail says (File uploaded.), and downloaded byte for byte; bob, a member who did not submit it, lists it and downloads it, offered no picker and no deletion; gus's next file, prepared before alice turns the intake off, is refused at its confirmation (issue.intake_disabled), which the picker says",
  async (world, info) => {
    info.setTimeout(90_000);
    const { api, db } = world;
    const intake = await intakeFilesWorld(world, info);
    const { alice, bob, gus, web, browserName } = intake;
    const login = await submitIntakeIssue(api, gus.token, web.id, { name: "Login breaks" });

    // gus's own: his file prepared, sent and confirmed; the detail says so, and the file downloads as it was sent.
    const own = await intake.signIn(gus);
    await intake.open(own, login);
    const detail = intakeDetail(own.page);
    const files = attachmentsOf(detail);
    await files.count(0).click();
    await expect(files.panel.getByText("No attachments.", { exact: true })).toBeVisible();
    await own.settled();
    const log = textFile("console.log", "TypeError: session is undefined");
    const confirmed = confirmation(own.page);
    await chooseAttachment(own.page, files.panel, log);
    const answer = await confirmed;
    expect(answer.status()).toBe(200);
    const logId = assetOf(answer);
    await expect(detail.getByRole("status").filter({ hasText: "File uploaded." })).toBeVisible();
    await expect(files.count(1)).toBeVisible();
    expect(own.writes).toEqual([
      {
        write: "POST /api/v0/assets",
        body: {
          purpose: "ISSUE_ATTACHMENT",
          scope: { kind: "issue", id: login.id },
          name: log.name,
          size: log.bytes.length,
          content_type: log.type,
        },
      },
      { write: `POST /api/v0/assets/${logId}/complete`, body: {} },
    ]);
    await intake.expectStored(logId, log, { by: gus.id, reader: gus.token });
    await downloadedExactly(own.page, files.panel.getByRole("button", { name: `Download ${log.name}` }), log);
    await own.settled();

    // bob lists it and downloads it; no picker, no deletion.
    const reader = await intake.signIn(bob);
    await intake.open(reader, login);
    const listed = attachmentsOf(intakeDetail(reader.page));
    await listed.count(1).click();
    const download = listed.panel.getByRole("button", { name: `Download ${log.name}`, exact: true });
    await expect(download).toBeVisible();
    await expect(listed.panel.getByLabel("Choose or replace file", { exact: true })).toHaveCount(0);
    await expect(listed.panel.getByLabel("Drop one file here", { exact: true })).toHaveCount(0);
    await expect(listed.panel.getByRole("button", { name: `Delete file ${log.name}`, exact: true })).toHaveCount(0);
    await downloadedExactly(reader.page, download, log);
    await reader.settled();
    expect(reader.writes).toEqual([]);
    expect([reader.watch.apiFailures, reader.watch.oldApiRequests, reader.watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(reader.page, reader.watch);
    await reader.close();

    // gus's next file: nerve has prepared it, its answer held, when alice turns the intake off; let through, the bytes
    // go to the storage and the confirmation is refused.
    let prepared = false;
    const release = await holdAnswer(own.page, "POST", "/api/v0/assets", 1, () => {
      prepared = true;
    });
    const trace = textFile("trace.txt", "at Login.submit (login.ts:12)");
    await chooseAttachment(own.page, files.panel, trace);
    await expect(files.panel.getByText("Preparing upload…", { exact: true })).toBeVisible();
    await expect.poll(() => prepared).toBe(true);
    expect(await changeProject(api, alice.token, web.id, { intake_view: false })).toMatchObject({ status: 200 });
    const refused = confirmation(own.page);
    await release();
    const refusal = await refused;
    expect([refusal.status(), ((await refusal.json()) as { code: string }).code]).toEqual([
      409,
      "issue.intake_disabled",
    ]);
    await expect(files.panel.getByRole("alert")).toHaveText(
      "Intake is turned off in this project. Turn it on to submit or triage work items."
    );
    await expect(files.count(1)).toBeVisible();
    await expect(detail.getByRole("status").filter({ hasText: "File uploaded." })).toHaveCount(0);
    expect(await uploadsNamed(db, login.id, trace.name)).toEqual([false]);
    expect(await issueAttachmentIds(api, alice.token, login.id)).toEqual([logId]);
    const traceId = assetOf(refusal);
    expect(own.writes.slice(2).map(({ write }) => write)).toEqual([
      "POST /api/v0/assets",
      `POST /api/v0/assets/${traceId}/complete`,
    ]);

    await own.settled();
    expect([own.watch.apiFailures, own.watch.oldApiRequests, own.watch.pageErrors]).toEqual([
      [`409 POST /api/v0/assets/${traceId}/complete`],
      [],
      [],
    ]);
    await expectQuietConsole(own.page, own.watch, { errors: failedLoads(browserName, 409) });
  }
);

pageFileStory(
  "H11 (page): alice, Web's admin, stops a file's transfer to a work item of its intake on its way, and nothing is confirmed; she selects another work item while a transfer is on its way, and its late end confirms nothing, the attachments of either as they were; she deletes an attachment of that one, gone from its list and from nerve's reads, its object kept for the cleanup job",
  async (world, info) => {
    info.setTimeout(90_000);
    const { api, db, uploadStatus } = world;
    const intake = await intakeFilesWorld(world, info);
    const { alice, bob, gus, web } = intake;
    const login = await submitIntakeIssue(api, gus.token, web.id, { name: "Login breaks" });
    const signup = await submitIntakeIssue(api, bob.token, web.id, { name: "Sign up fails" });
    const trace = textFile("trace.txt", "at Login.submit (login.ts:12)");
    const steps = textFile("steps.txt", "1. Open the sign up form 2. Submit it");
    const attach = (token: string, issueId: string, { bytes, type, name }: typeof trace) =>
      storedFile(api, token, "ISSUE_ATTACHMENT", { kind: "issue", id: issueId }, bytes, type, name, uploadStatus);
    const [traced, stepped] = await Promise.all([
      attach(gus.token, login.id, trace),
      attach(bob.token, signup.id, steps),
    ]);

    // Sign up fails, the newest, selected; its attachment listed. Each upload's bytes held on their way from now on.
    const admin = await intake.signIn(alice);
    await intake.open(admin, signup);
    const detail = intakeDetail(admin.page);
    const files = attachmentsOf(detail);
    // The files listed, by their download buttons' names, in order.
    const listed = () =>
      files.panel
        .getByRole("button", { name: /^Download / })
        .evaluateAll((buttons) =>
          buttons.map((button) => button.getAttribute("aria-label")?.slice("Download ".length))
        );
    await files.count(1).click();
    await expect.poll(listed).toEqual([steps.name]);
    await admin.settled();
    const uploads = await holdUploads(admin.page);

    // Stopped on its way: nothing confirmed, the list as it was.
    const stopped = textFile("stopped.txt", "bytes whose upload alice stops");
    await chooseAttachment(admin.page, files.panel, stopped);
    await expect.poll(() => uploads.length).toBe(1);
    await expect(files.panel.getByText("Uploading file…", { exact: true })).toBeVisible();
    await files.panel.getByRole("button", { name: "Cancel upload", exact: true }).click();
    await expect(
      files.panel.getByText("Upload stopped. This does not delete any file already received.", { exact: true })
    ).toBeVisible();
    uploads[0]!.release();
    await uploads[0]!.settled;
    await admin.settled();
    expect(await uploadsNamed(db, signup.id, stopped.name)).toEqual([false]);
    await expect(files.count(1)).toBeVisible();
    await expect.poll(listed).toEqual([steps.name]);

    // Left behind as she selects Login breaks: its late end confirms nothing, neither here nor there.
    const late = textFile("late.txt", "bytes left on their way");
    await chooseAttachment(admin.page, files.panel, late);
    await expect.poll(() => uploads.length).toBe(2);
    await admin.settled();
    await intakeRow(admin.page, login).click();
    await expect(admin.page).toHaveURL(selecting(login));
    await expectHeaderOf(admin.page, login);
    await expect(descriptionOf(admin.page, login)).toBeVisible();
    await expect.poll(listed).toEqual([trace.name]);
    await admin.settled();
    uploads[1]!.release();
    await uploads[1]!.settled;
    await admin.settled();
    await expect(files.count(1)).toBeVisible();
    await expect.poll(listed).toEqual([trace.name]);
    await expect(detail.getByRole("status").filter({ hasText: "File uploaded." })).toHaveCount(0);
    expect(await uploadsNamed(db, signup.id, late.name)).toEqual([false]);
    expect(await issueAttachmentIds(api, alice.token, signup.id)).toEqual([stepped.id]);
    expect(await issueAttachmentIds(api, alice.token, login.id)).toEqual([traced.id]);
    expect(admin.writes.map(({ write }) => write)).toEqual(["POST /api/v0/assets", "POST /api/v0/assets"]);

    // Deleted, confirmed: gone from the list and from nerve's reads, the detail says so; its object kept.
    await files.panel.getByRole("button", { name: `Delete file ${trace.name}`, exact: true }).click();
    const dialog = modalDialog(admin.page);
    await expect(dialog).toContainText("Delete attachment");
    const confirm = dialog.getByRole("button", { name: "Delete file", exact: true });
    const deleted = await answerTo(admin.page, "DELETE", `/api/v0/assets/${traced.id}`, () => confirm.click());
    expect(deleted.status()).toBe(204);
    await expect(dialog).toHaveCount(0);
    await expect(detail.getByRole("status").filter({ hasText: "File deleted." })).toBeVisible();
    await expect(files.count(0)).toBeVisible();
    await expect(files.panel.getByText("No attachments.", { exact: true })).toBeVisible();
    await intake.expectStored(traced.id, trace, { by: gus.id, reader: alice.token, deletedBy: alice.id });
    expect(await issueAttachmentIds(api, alice.token, login.id)).toEqual([]);

    await admin.settled();
    expect([admin.watch.apiFailures, admin.watch.oldApiRequests, admin.watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(admin.page, admin.watch);
  }
);
