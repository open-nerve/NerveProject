/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { changeProjectPreferences, readProjectPreferences } from "../../fixtures/api";
import { chooseAttachment, contentFilesPage, textFile as text } from "../../fixtures/content-files-page";
import { deferred } from "../../fixtures/deferred";
import { modalDialog } from "../../fixtures/dialogs";
import { pageFileStory } from "../../fixtures/file-story";
import { createIssue, issueRead } from "../../fixtures/issues";
import { storedFile, type Asset } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

pageFileStory(
  "F8 (page): the detail widget uploads by chooser and drop, pages 51 attachments, downloads exact bytes, deletes with explicit confirmation and sorts by the server count",
  async (world, info) => {
    info.setTimeout(180_000);
    const { api, page, baseURL, db, uploadStatus } = world;
    const files = await contentFilesPage(world, info);
    const { owner, workspace, project, issue } = files;
    const scope = { kind: "issue", id: issue.id } as const;
    const seeds = Array.from({ length: 49 }, (_, index) =>
      text(`seed-${String(index).padStart(2, "0")}.txt`, `seeded attachment ${index}`)
    );
    const seeded: Asset[] = [];
    await seeds.reduce(async (prior, file) => {
      await prior;
      seeded.push(
        await storedFile(api, owner.token, "ISSUE_ATTACHMENT", scope, file.bytes, file.type, file.name, uploadStatus)
      );
    }, Promise.resolve());
    const one = await createIssue(api, owner.token, project.id, { name: "One attachment" });
    const zero = await createIssue(api, owner.token, project.id, { name: "No attachments" });
    await storedFile(
      api,
      owner.token,
      "ISSUE_ATTACHMENT",
      { kind: "issue", id: one.id },
      Buffer.from("one"),
      "text/plain",
      "one.txt",
      uploadStatus
    );

    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
    await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(issue.name);
    const trigger = (count: number) => page.getByRole("button", { name: `Attachments ${count}`, exact: true });
    await trigger(49).focus();
    await page.keyboard.press("Enter");
    await expect(trigger(49)).toHaveAttribute("aria-expanded", "true");
    // The comment composer has its own picker: every control below belongs to the widget's own panel, which a
    // confirmed command's canonical re-read remounts.
    const widget = page.getByRole("button", { name: /^Attachments \d+$/ }).locator("xpath=../following-sibling::*[1]");
    const downloads = widget.getByRole("button", { name: /^Download / });
    await expect(downloads).toHaveCount(49);
    await expect(widget.getByRole("button", { name: "Load more", exact: true })).toHaveCount(0);

    // The native chooser: the selection moves focus to the one action, which the keyboard performs.
    const chosen = text("chosen.txt", "bytes chosen through the native file chooser");
    const chooser = page.waitForEvent("filechooser");
    await widget.getByLabel("Choose or replace file", { exact: true }).click();
    await (await chooser).setFiles({ name: chosen.name, mimeType: chosen.type, buffer: chosen.bytes });
    await expect(widget.getByText("Ready to upload.", { exact: true })).toBeVisible();
    await expect(widget.getByRole("button", { name: "Upload file", exact: true })).toBeFocused();
    await page.keyboard.press("Enter");
    const chosenID = await files.transfer(0, "ISSUE_ATTACHMENT", scope, chosen);
    // Confirmed: the host announces it, the re-read remounts the widget ready for the next file with the focus back on
    // its input, and the count is the server's.
    await expect(page.getByRole("status").filter({ hasText: "File uploaded." })).toBeVisible();
    await expect(widget.getByText("Choose a file.", { exact: true })).toBeVisible();
    await expect(widget.getByLabel("Choose or replace file", { exact: true })).toBeFocused();
    await expect(trigger(50)).toBeVisible();
    await files.persisted(chosenID, chosen);

    // A native DataTransfer drop of one file onto the labelled drop zone.
    const dropped = text("dropped.txt", "bytes dropped onto the attachment zone");
    const data = await page.evaluateHandle(
      ({ name, type, body }) => {
        const transfer = new DataTransfer();
        transfer.items.add(new File([body], name, { type }));
        return transfer;
      },
      { name: dropped.name, type: dropped.type, body: dropped.bytes.toString() }
    );
    await widget.getByLabel("Drop one file here", { exact: true }).dispatchEvent("drop", { dataTransfer: data });
    await widget.getByRole("button", { name: "Upload file", exact: true }).click();
    const droppedID = await files.transfer(1, "ISSUE_ATTACHMENT", scope, dropped);
    await expect(trigger(51)).toBeVisible();
    await files.persisted(droppedID, dropped);

    // A stopped upload: the person cancels while its bytes are on their way, and nothing confirms the file afterwards.
    const stopped = text("stopped.txt", "bytes whose upload the person stops");
    const putReached = deferred();
    const putReleased = deferred();
    let putHeld = false;
    await page.route(
      () => true,
      async (route) => {
        if (route.request().method() !== "PUT" || putHeld) return route.fallback();
        putHeld = true;
        putReached.resolve();
        await putReleased.promise;
        // The page has cancelled it: continuing a request the page dropped sends nothing.
        await route.continue().catch(() => undefined);
      }
    );
    await chooseAttachment(page, widget, stopped);
    await putReached.promise;
    await widget.getByRole("button", { name: "Cancel upload", exact: true }).click();
    await expect(
      widget.getByText("Upload stopped. This does not delete any file already received.", { exact: true })
    ).toBeVisible();
    putReleased.resolve();
    await page.waitForTimeout(1500);
    const [stoppedRow] = await db.query<{ id: string; uploaded: boolean }>(
      "SELECT id, is_uploaded AS uploaded FROM file_assets WHERE attributes->>'name'=$1",
      [stopped.name]
    );
    expect(stoppedRow).toMatchObject({ uploaded: false });
    expect(files.calls("POST", `/api/v0/assets/${stoppedRow!.id}/complete`)).toHaveLength(0);
    await expect(trigger(51)).toBeVisible();

    // Fifty rows, then the server's continuation: every one of the 51 exactly once.
    await expect(downloads).toHaveCount(50);
    const more = widget.getByRole("button", { name: "Load more", exact: true });
    await more.click();
    await expect(downloads).toHaveCount(51);
    await expect(more).toHaveCount(0);
    expect((await downloads.allTextContents()).length).toBe(51);
    const listed = await downloads.evaluateAll((buttons) => buttons.map((button) => button.getAttribute("aria-label")));
    expect(listed.toSorted()).toEqual([...seeds, chosen, dropped].map((file) => `Download ${file.name}`).toSorted());
    const reads = files.calls("GET", `/api/v0/issues/${issue.id}/attachments`).map((request) => new URL(request.url()));
    expect(reads.filter((url) => url.searchParams.has("cursor"))).toHaveLength(1);

    // The browser's own downloads hold the stored bytes.
    await files.downloaded(widget.getByRole("button", { name: `Download ${dropped.name}`, exact: true }), dropped);
    await files.downloaded(widget.getByRole("button", { name: `Download ${seeds[7]!.name}`, exact: true }), seeds[7]!);

    // Escape cancels with no deletion; the confirmation deletes exactly once.
    const removed = seeded[3]!;
    await widget.getByRole("button", { name: `Delete file ${removed.name}`, exact: true }).click();
    await expect(modalDialog(page)).toContainText("Delete attachment");
    await page.keyboard.press("Escape");
    await expect(modalDialog(page)).toHaveCount(0);
    expect(files.calls("DELETE", `/api/v0/assets/${removed.id}`)).toHaveLength(0);
    await widget.getByRole("button", { name: `Delete file ${removed.name}`, exact: true }).click();
    await modalDialog(page).getByRole("button", { name: "Delete file", exact: true }).click();
    await expect(modalDialog(page)).toHaveCount(0);
    await expect(widget.getByRole("button", { name: `Delete file ${removed.name}`, exact: true })).toHaveCount(0);
    await expect(trigger(50)).toBeVisible();
    await expect(page.getByRole("status").filter({ hasText: "File deleted." })).toBeVisible();
    await expect(widget.getByRole("button", { name: "Reload files", exact: true })).toBeFocused();
    expect(files.calls("DELETE", `/api/v0/assets/${removed.id}`)).toHaveLength(1);
    expect(await db.query("SELECT is_deleted, deleted_by_id FROM file_assets WHERE id=$1", [removed.id])).toEqual([
      { is_deleted: true, deleted_by_id: owner.id },
    ]);

    // A deletion whose answer is lost is never sent again by itself: the dialog asks for an explicit read first.
    const unknown = seeded[4]!;
    await page.route(
      `**/api/v0/assets/${unknown.id}`,
      async (route) => {
        if (route.request().method() !== "DELETE") return route.fallback();
        await route.fetch();
        await route.abort("failed");
      },
      { times: 1 }
    );
    await widget.getByRole("button", { name: `Delete file ${unknown.name}`, exact: true }).click();
    await modalDialog(page).getByRole("button", { name: "Delete file", exact: true }).click();
    await expect(modalDialog(page).getByRole("alert")).toBeVisible();
    await expect(
      modalDialog(page).getByText(
        "The deletion result is unknown. Read the latest status before explicitly trying again.",
        { exact: true }
      )
    ).toBeVisible();
    await expect(modalDialog(page).getByRole("button", { name: "Delete file", exact: true })).toBeDisabled();
    await modalDialog(page).getByRole("button", { name: "Check latest status", exact: true }).click();
    await expect(widget.getByRole("button", { name: `Delete file ${unknown.name}`, exact: true })).toHaveCount(0);
    await expect(modalDialog(page).getByRole("button", { name: "Delete file", exact: true })).toBeEnabled();
    expect(files.calls("GET", `/api/v0/assets/${unknown.id}`)).toHaveLength(1);
    expect(files.calls("DELETE", `/api/v0/assets/${unknown.id}`)).toHaveLength(1);
    // Only the person's second confirmation sends it again; the server's answer is the same idempotent 204.
    await modalDialog(page).getByRole("button", { name: "Delete file", exact: true }).click();
    await expect(modalDialog(page)).toHaveCount(0);
    await expect(trigger(49)).toBeVisible();
    const deletions = files.calls("DELETE", `/api/v0/assets/${unknown.id}`);
    expect(deletions).toHaveLength(2);
    expect((await deletions[1]!.response())?.status()).toBe(204);
    expect(await issueRead(api, owner.token, issue.id)).toMatchObject({ issue: { attachment_count: 49 } });

    // The spreadsheet sorts by the server's count, both ways, and shows each total.
    const preferences = await readProjectPreferences(api, owner.token, project.id);
    const spreadsheet = {
      ...preferences,
      display_filters: { ...preferences.display_filters, layout: "spreadsheet", order_by: "-created_at" },
    } as typeof preferences;
    expect(await changeProjectPreferences(api, owner.token, project.id, spreadsheet)).toMatchObject({ status: 200 });
    await page.goto(`${baseURL}/${workspace.slug}/projects/${project.id}/issues`);
    const table = page.getByRole("table");
    const rows = table.locator("tbody > tr");
    const order = () =>
      rows.evaluateAll((elements) =>
        elements.map((row) => row.querySelector('td[id^="issue-"]')?.id.slice(6)).filter(Boolean)
      );
    const heading = table.getByRole("columnheader", { name: "Attachments", exact: true });
    for (const [label, direction, ids] of [
      ["Most Least", "descending", [issue.id, one.id, zero.id]],
      ["Least Most", "ascending", [zero.id, one.id, issue.id]],
    ] as const) {
      // eslint-disable-next-line no-await-in-loop -- Each direction is chosen from the actual header in turn.
      await heading.getByRole("button").click();
      // eslint-disable-next-line no-await-in-loop
      await page.getByRole("menuitem", { name: label, exact: true }).click();
      // eslint-disable-next-line no-await-in-loop
      await expect(heading).toHaveAttribute("aria-sort", direction);
      // eslint-disable-next-line no-await-in-loop
      await expect.poll(order).toEqual(ids);
    }
    await expect(table.getByText("49 attachments", { exact: true })).toBeVisible();
    await expect(table.getByText("1 attachment", { exact: true })).toBeVisible();
    await expect(table.getByText("0 attachments", { exact: true })).toBeVisible();
    // The lost answer, and the verification's read of the file that deletion removed.
    await files.finish(
      (failure) =>
        failure.endsWith(`DELETE /api/v0/assets/${unknown.id}`) || failure === `404 GET /api/v0/assets/${unknown.id}`
    );
  }
);
