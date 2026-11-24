/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { contentOf, intakeRowOf, uploadsNamed } from "../../fixtures/assert/intake";
import { expectQuietConsole } from "../../fixtures/browser";
import { pasteInto, pngFile } from "../../fixtures/content-files-page";
import { failedLoads } from "../../fixtures/cycle-pages";
import { sentWithin } from "../../fixtures/dialogs";
import { pageFileStory } from "../../fixtures/file-story";
import { submitIntakeIssue, triageIntakeIssue } from "../../fixtures/intake";
import {
  assetOf,
  confirmation,
  descriptionOf,
  intakeFilesWorld,
  pasteImage,
  type PersonPage,
} from "../../fixtures/intake-files";
import { identifierOf, intakeEditPath } from "../../fixtures/intake-pages";
import type { Issue } from "../../fixtures/issues";
import { expectDecoded, redPixel } from "../../fixtures/profile-images-page";
import { holdAnswer } from "../../fixtures/settings-pages";
import { expect } from "../../fixtures/test";

// H11's page version, the images of the description of a work item of a project's intake (M7 design 2, 3.6; P5b spec
// 2.2, 2.8, R9, R12): while it is in the intake, its submitter adds them, a guest too, as its admins do (the intake's
// rule, intakeEditableBy, which the content view's canWrite takes for it); each is prepared, sent to the storage, local
// or S3, confirmed, and bound by the intake's change of the description. A member who did not submit it reads it,
// images too, and adds none. A transfer the page began before someone else triaged the work item is refused at its
// confirmation (issue.intake_closed), which the image says; the page does not read the work item again (spec 2.2, a
// known behaviour). Accepted, the work item's own page (/browse/…) follows issue.update: its guest submitter reads it,
// a member adds images by the work item's change. The API version is h11-intake-assets.spec.ts; the uploads' paging,
// drops, copies and lifetimes are M5's page stories' (F8–F16, R12).

const imageIds = (html: string) => [...html.matchAll(/src="([0-9a-f-]{36})"/g)].map((match) => match[1]);

/**
 * Checks that who reads the description of the work item issue alone, as it shows: not editable, its images shown,
 * and a paste of an image taken by nothing and sending no upload.
 */
async function expectReadsAlone(who: PersonPage, issue: Issue, images: number) {
  const description = descriptionOf(who.page, issue);
  await expect(description).toHaveAttribute("contenteditable", "false");
  await expect(description.locator("img.image-component")).toHaveCount(images);
  const paste = async () => {
    expect(await pasteInto(description, {}, [pngFile("not-mine.png")])).toBe(false);
  };
  expect(await sentWithin(who.page, "POST", "/api/v0/assets", paste), "an upload prepared").toBe(false);
  await expect(description.locator(".image-upload-component")).toHaveCount(0);
}

pageFileStory(
  "H11 (page): gus, a guest of Web, pastes an image into the description of the work item he submitted to its intake: prepared for it, sent to the storage, confirmed and bound by the intake's change, its bytes stored; bob, a member who did not submit it, sees it in a description he reads alone and uploads nothing; gus's next image, prepared before alice declines the work item, is refused at its confirmation (issue.intake_closed), which the image says",
  async (world, info) => {
    info.setTimeout(90_000);
    const { api, db } = world;
    const intake = await intakeFilesWorld(world, info);
    const { alice, bob, gus, web, browserName } = intake;
    // No description: opening it writes nothing (an empty one gets no block ids).
    const login = await submitIntakeIssue(api, gus.token, web.id, { name: "Login breaks" });

    // gus's own: his paste is prepared, sent and confirmed, then the description saved with it, by the intake's change.
    const own = await intake.signIn(gus);
    await intake.open(own, login);
    const description = descriptionOf(own.page, login);
    await expect(description).toHaveAttribute("contenteditable", "true");
    const shot = pngFile("login-error.png");
    const { id: shotId, save } = await pasteImage(own.page, login, shot, intakeEditPath(login));
    expect(save.status()).toBe(200);
    await expectDecoded(description.locator("img.image-component"));
    expect(own.writes).toEqual([
      {
        write: "POST /api/v0/assets",
        body: {
          purpose: "ISSUE_DESCRIPTION",
          scope: { kind: "issue", id: login.id },
          name: shot.name,
          size: shot.bytes.length,
          content_type: shot.type,
        },
      },
      { write: `POST /api/v0/assets/${shotId}/complete`, body: {} },
      { write: `PATCH ${intakeEditPath(login)}`, body: { description_html: expect.any(String) } },
    ]);
    const saved = (own.writes[2]!.body as { description_html: string }).description_html;
    expect(imageIds(saved)).toEqual([shotId]);
    await intake.expectStored(shotId, shot, { by: gus.id, reader: gus.token });
    expect(await contentOf(db, login.id)).toMatchObject({ description_html: saved, written_by: gus.id });
    await own.settled();

    // bob reads it, the image too, and uploads nothing.
    const reader = await intake.signIn(bob);
    await intake.open(reader, login);
    await expectDecoded(descriptionOf(reader.page, login).locator("img.image-component"));
    await expectReadsAlone(reader, login, 1);
    expect(reader.writes).toEqual([]);
    await reader.settled();
    expect([reader.watch.apiFailures, reader.watch.oldApiRequests, reader.watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(reader.page, reader.watch);
    await reader.close();

    // gus's next image: nerve has prepared it, its answer held, when alice declines the work item; let through, the
    // bytes go to the storage and the confirmation is refused.
    let prepared = false;
    const release = await holdAnswer(own.page, "POST", "/api/v0/assets", 1, () => {
      prepared = true;
    });
    await description.focus();
    await own.page.keyboard.press("ControlOrMeta+End");
    expect(await pasteInto(description, {}, [pngFile("second.png", redPixel)])).toBe(true);
    const placeholder = description.locator(".image-upload-component");
    await expect(placeholder.getByText("Preparing image", { exact: true })).toBeVisible();
    await expect.poll(() => prepared).toBe(true);
    await triageIntakeIssue(api, alice.token, login.id, { status: "declined" });
    const refused = confirmation(own.page);
    await release();
    const refusal = await refused;
    const secondId = assetOf(refusal);
    expect([refusal.status(), ((await refusal.json()) as { code: string }).code]).toEqual([409, "issue.intake_closed"]);
    await expect(placeholder.getByRole("alert")).toHaveText(
      "The intake work item has been accepted, declined or marked a duplicate. It can no longer be edited in the intake."
    );
    await expect(description.locator("img.image-component")).toHaveCount(1);
    expect(await uploadsNamed(db, login.id, "second.png")).toEqual([false]);
    expect(await intakeRowOf(db, login.id)).toMatchObject({ status: "declined" });
    expect((await contentOf(db, login.id)).description_html).toBe(saved);
    expect(own.writes.slice(3).map(({ write }) => write)).toEqual([
      "POST /api/v0/assets",
      `POST /api/v0/assets/${secondId}/complete`,
    ]);

    await own.settled();
    expect([own.watch.apiFailures, own.watch.oldApiRequests, own.watch.pageErrors]).toEqual([
      [`409 POST /api/v0/assets/${secondId}/complete`],
      [],
      [],
    ]);
    await expectQuietConsole(own.page, own.watch, { errors: failedLoads(browserName, 409) });
  }
);

pageFileStory(
  "H11 (page): accepted, the work item gus submitted to Web's intake takes images by issue.update on its own page (/browse/…): gus, its guest, reads its description alone and uploads nothing; bob, a member, pastes an image into it, prepared, sent to the storage, confirmed and bound by the work item's change, its bytes stored",
  async (world, info) => {
    info.setTimeout(90_000);
    const { api, baseURL, db } = world;
    const intake = await intakeFilesWorld(world, info);
    const { alice, bob, gus, slug, web } = intake;
    const signup = await submitIntakeIssue(api, gus.token, web.id, { name: "Sign up fails" });
    await triageIntakeIssue(api, alice.token, signup.id, { status: "accepted" });
    const page = `${baseURL}/${slug}/browse/${identifierOf(signup)}`;

    // bob's: his paste bound by the work item's own change.
    const member = await intake.signIn(bob);
    await member.page.goto(page);
    const description = descriptionOf(member.page, signup);
    await expect(description).toHaveAttribute("contenteditable", "true");
    await member.settled();
    const picture = pngFile("signup-form.png", redPixel);
    const { id: pictureId, save } = await pasteImage(member.page, signup, picture, `/api/v0/issues/${signup.id}`);
    expect(save.status()).toBe(200);
    await expectDecoded(description.locator("img.image-component"));
    expect(member.writes.map(({ write }) => write)).toEqual([
      "POST /api/v0/assets",
      `POST /api/v0/assets/${pictureId}/complete`,
      `PATCH /api/v0/issues/${signup.id}`,
    ]);
    expect(member.writes[0]!.body).toMatchObject({
      purpose: "ISSUE_DESCRIPTION",
      scope: { kind: "issue", id: signup.id },
    });
    const saved = (member.writes[2]!.body as { description_html: string }).description_html;
    expect(imageIds(saved)).toEqual([pictureId]);
    await intake.expectStored(pictureId, picture, { by: bob.id, reader: bob.token });
    expect(await contentOf(db, signup.id)).toMatchObject({ description_html: saved, written_by: bob.id });
    await member.settled();
    expect([member.watch.apiFailures, member.watch.oldApiRequests, member.watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(member.page, member.watch);

    // gus's: his own work item, which he reads alone now, bob's image too.
    const guest = await intake.signIn(gus);
    await guest.page.goto(page);
    await expectDecoded(descriptionOf(guest.page, signup).locator("img.image-component"));
    await expectReadsAlone(guest, signup, 1);
    expect(guest.writes).toEqual([]);
    await guest.settled();
    expect([guest.watch.apiFailures, guest.watch.oldApiRequests, guest.watch.pageErrors]).toEqual([[], [], []]);
    await expectQuietConsole(guest.page, guest.watch);
    await guest.close();
  }
);
