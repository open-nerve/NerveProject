/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { randomBytes } from "node:crypto";

import { chooseAttachment, contentFilesPage, type ContentFile } from "../fixtures/content-files-page";
import { pageFileStory } from "../fixtures/file-story";
import { expect } from "../fixtures/test";

// Chromium alone lets a test hold the page's own upload to a slow line (DevTools network conditions). The transfer
// budget is the grace (15 s by default) plus the size at the minimum rate, so this upload outlasts the grace alone.
const uploadBytesPerSecond = 72 * 1024;

pageFileStory(
  "F1 (page, slow network): an attachment over 1 MiB that takes longer than 15 seconds to send is stored whole",
  async (world, info) => {
    info.setTimeout(180_000);
    const { page, baseURL } = world;
    const files = await contentFilesPage(world, info);
    const { workspace, issue } = files;
    const slow: ContentFile = { name: "slow.bin", type: "application/octet-stream", bytes: randomBytes(1_310_720) };
    expect(slow.bytes.length).toBeGreaterThan(1 << 20);

    await page.goto(`${baseURL}/${workspace.slug}/browse/FILE-${issue.sequence_id}`);
    await expect(page.getByRole("textbox", { name: "Work item title", exact: true })).toHaveValue(issue.name);
    const trigger = page.getByRole("button", { name: "Attachments 0", exact: true });
    await trigger.click();
    await expect(trigger).toHaveAttribute("aria-expanded", "true");
    const widget = page.getByRole("button", { name: /^Attachments \d+$/ }).locator("xpath=../following-sibling::*[1]");

    const devtools = await page.context().newCDPSession(page);
    await devtools.send("Network.enable");
    await devtools.send("Network.emulateNetworkConditions", {
      offline: false,
      latency: 0,
      downloadThroughput: -1,
      uploadThroughput: uploadBytesPerSecond,
    });
    const started = Date.now();
    await chooseAttachment(page, widget, slow);
    // The transfer runs at the line's speed; the checks of its requests below read them once it has finished.
    await expect(page.getByRole("button", { name: "Attachments 1", exact: true })).toBeVisible({ timeout: 90_000 });
    const elapsed = Date.now() - started;
    await devtools.send("Network.emulateNetworkConditions", {
      offline: false,
      latency: 0,
      downloadThroughput: -1,
      uploadThroughput: -1,
    });
    const id = await files.transfer(0, "ISSUE_ATTACHMENT", { kind: "issue", id: issue.id }, slow);
    const put = files.requests.find((request) => request.method() === "PUT")!;
    const sent = put.timing();
    expect(elapsed).toBeGreaterThan(15_000);
    expect(sent.responseEnd - sent.requestStart).toBeGreaterThan(15_000);

    await expect(page.getByRole("status").filter({ hasText: "File uploaded." })).toBeVisible();
    await files.persisted(id, slow);
    await info.attach("slow-upload", {
      body: JSON.stringify({
        id,
        bytes: slow.bytes.length,
        uploadBytesPerSecond,
        elapsedMs: elapsed,
        putMs: sent.responseEnd - sent.requestStart,
      }),
      contentType: "application/json",
    });
    await files.finish();
  }
);
