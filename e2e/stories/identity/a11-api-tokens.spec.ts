import { readFile } from "node:fs/promises";

import type { components } from "@nerve/api-client";
import type { Download } from "@playwright/test";

import { expectTokenStored } from "../../fixtures/assert/identity";
import { bearer, createPAT, emailFor, login, newRecord, register, writeRecord } from "../../fixtures/auth";
import { EMOJI_CHECK_WARNING, expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  answerTo,
  expectListBesideButton,
  expectTokenGone,
  holdAnswer,
  parseCsv,
  recordTokensShown,
  registerOnboarded,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// A11, personal access tokens (M2 design 2).

const weekMs = 7 * 24 * 60 * 60 * 1000;

test("A11 (page): a token created on the page shows once, and nowhere after; the list shows it without it, and revokes it", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await registerOnboarded(api, emailFor(testInfo)));
  const watch = await watchPage(page);
  const log: string[] = [];
  page.on("console", (message) => log.push(message.text()));
  await page.goto("/settings/profile/api-tokens");
  await expect(page.getByText("No Personal token yet")).toBeVisible();

  // Created with a name, a description and a week. A day of one's own is picked in a calendar that opens beside
  // its button, and that Escape closes; the week replaces it.
  await page.getByRole("button", { name: "Add access token" }).first().click();
  await page.getByLabel("Title").fill("deploy");
  await page.getByLabel("Description").fill("ci, #1");
  await page.getByRole("button", { name: "Set expiration date" }).click();
  await page.getByRole("option", { name: "Custom date" }).click();
  // The dropdown's button, which holds the button of its label.
  const day = page.getByRole("button", { name: "Set date" }).first();
  await day.click();
  await expectListBesideButton(page, day);
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await day.click();
  await expectListBesideButton(page, day);
  await page.getByRole("grid").getByRole("button", { disabled: false }).last().click();
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(page.getByText(/^Expires .+ at /)).toBeVisible();
  await page.getByRole("button", { name: "Custom date" }).click();
  await page.getByRole("option", { name: "1 week" }).click();
  const downloaded = page.waitForEvent("download", { timeout: 10_000 });
  const answer = await answerTo(page, "POST", "/api/v0/me/api-tokens", () =>
    page.getByRole("button", { name: "Generate token" }).click()
  );
  expect(answer.status()).toBe(201);
  const created = (await answer.json()) as components["schemas"]["ApiTokenCreated"];
  expect(created).toMatchObject({ label: "deploy", description: "ci, #1", last_used: null });
  const createdAt = Date.now();
  expect(Math.abs(new Date(created.expired_at ?? "").getTime() - (createdAt + weekMs))).toBeLessThan(60_000);
  await expectTokenStored(db, created.id, created.token, created.expired_at ?? "");

  // The token shows this once, and the CSV downloaded has it: the header, and one record of four whole fields,
  // each comma of the description and of the date in its own.
  await expect(page.getByText(created.token)).toBeVisible();
  const csv = parseCsv(await readFile(await (await downloaded).path(), "utf8"));
  expect(csv).toEqual([
    ["Title", "Description", "Expiry", "Secret key"],
    ["deploy", "ci, #1", expect.stringMatching(/^[A-Z][a-z]{2} \d{2}, \d{4}$/), created.token],
  ]);

  // Closed, the list has the new token, without the token itself: nowhere on the page, in its address, its
  // storage or its console; nor after going to another tab and back, nor after a reload.
  await page.getByRole("button", { name: "Close" }).click();
  await expect(page.getByText(created.token)).toHaveCount(0);
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  // Opened again, the modal asks for a new token.
  await page.getByRole("button", { name: "Add access token" }).first().click();
  await expect(page.getByLabel("Title")).toHaveValue("");
  await expect(page.getByText(created.token)).toHaveCount(0);
  await page.getByRole("button", { name: "Cancel" }).click();
  await expect(page.getByLabel("Title")).toHaveCount(0);
  await expect(page.getByText("Never used")).toBeVisible();
  await expectTokenGone(page, created.token, log);
  await page.getByRole("button", { name: "Security" }).click();
  await expect(page).toHaveURL("/settings/profile/security");
  await page.getByRole("button", { name: "Personal Access Tokens" }).click();
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  await expectTokenGone(page, created.token, log);
  await page.reload();
  await expect(page.getByText("deploy", { exact: true })).toBeVisible();
  await expectTokenGone(page, created.token, log);

  // The token works until it is revoked on the page, which then lists it no more. The revoke button shows
  // without a hover.
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(200);
  await page.getByRole("button", { name: "Revoke token" }).click();
  const revoked = await answerTo(page, "DELETE", `/api/v0/api-tokens/${created.id}`, () =>
    page.getByRole("button", { name: "Delete" }).click()
  );
  expect(revoked.status()).toBe(204);
  await expect(page.getByText("No Personal token yet")).toBeVisible();
  const [row] = await db.query<{ deleted_at: Date | null }>("SELECT deleted_at FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(row?.deleted_at).not.toBeNull();
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(401);

  expect(watch.apiFailures).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  // Two loads of the page: the first, and the reload.
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING] });
});

test("A11 (page): what a modal's opening left unfinished stays there: a token answered after it closed shows nowhere; its cleanup spares the next opening", async ({
  api,
  signedInPage,
}, testInfo) => {
  const page = await signedInPage(await registerOnboarded(api, emailFor(testInfo)));
  const watch = await watchPage(page);
  const log: string[] = [];
  page.on("console", (message) => log.push(message.text()));
  const downloads: Download[] = [];
  page.on("download", (download) => downloads.push(download));
  // The page's timers run on Playwright's clock, which keeps time as time goes until the story stops it (below).
  await page.clock.install();
  await page.goto("/settings/profile/api-tokens");
  await expect(page.getByText("No Personal token yet")).toBeVisible();
  // Every token the document holds from now on, even for a moment: a look at the page afterwards misses one that
  // showed and went.
  const shown = await recordTokensShown(page);

  /** Generates label in the modal opened now, and cancels while nerve's answer is held from the page. */
  const generateThenCancel = async (label: string): Promise<() => Promise<void>> => {
    const release = await holdAnswer(page, "POST", "/api/v0/me/api-tokens");
    await page.getByRole("button", { name: "Add access token" }).first().click();
    await page.getByLabel("Title").fill(label);
    await page.getByRole("button", { name: "Set expiration date" }).click();
    await page.getByRole("option", { name: "1 week" }).click();
    await Promise.all([
      page.waitForRequest(
        (request) => request.method() === "POST" && new URL(request.url()).pathname === "/api/v0/me/api-tokens",
        { timeout: 10_000 }
      ),
      page.getByRole("button", { name: "Generate token" }).click(),
    ]);
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByLabel("Title")).toHaveCount(0);
    return release;
  };

  // Answered after the modal closed, the create lists the token, and does nothing else: the token never shows,
  // nor is it anywhere on the page, and nothing downloads. The list shows it once the create's continuation has
  // run, which is where the modal would have taken the token and downloaded its CSV.
  const cancelledAnswer = await answerTo(page, "POST", "/api/v0/me/api-tokens", await generateThenCancel("cancelled"));
  expect(cancelledAnswer.status()).toBe(201);
  const cancelled = (await cancelledAnswer.json()) as components["schemas"]["ApiTokenCreated"];
  await expect(page.getByText("cancelled", { exact: true })).toBeVisible();
  expect(await shown()).toEqual([]);
  expect(downloads).toHaveLength(0);
  await expectTokenGone(page, cancelled.token, log);

  // So too when the modal opens again before the answer comes: the modal opened again still asks for a new token.
  const release = await generateThenCancel("late");
  await page.getByRole("button", { name: "Add access token" }).first().click();
  await expect(page.getByLabel("Title")).toHaveValue("");
  const answer = await answerTo(page, "POST", "/api/v0/me/api-tokens", release);
  expect(answer.status()).toBe(201);
  const late = (await answer.json()) as components["schemas"]["ApiTokenCreated"];
  expect(late).toMatchObject({ label: "late" });
  await expect(page.getByText("late", { exact: true })).toBeVisible();
  await expect(page.getByLabel("Title")).toHaveValue("");
  expect(await shown()).toEqual([]);
  expect(downloads).toHaveLength(0);
  await expectTokenGone(page, late.token, log);

  // The modal's own create shows its token, which the record has, and its CSV is the page's only download:
  // Chromium reports downloads in the order they begin, so one that a late answer had begun would have come first.
  await page.getByLabel("Title").fill("own");
  await page.getByRole("button", { name: "Set expiration date" }).click();
  await page.getByRole("option", { name: "1 week" }).click();
  const downloaded = page.waitForEvent("download", { timeout: 10_000 });
  const ownAnswer = await answerTo(page, "POST", "/api/v0/me/api-tokens", () =>
    page.getByRole("button", { name: "Generate token" }).click()
  );
  expect(ownAnswer.status()).toBe(201);
  const own = (await ownAnswer.json()) as components["schemas"]["ApiTokenCreated"];
  await expect(page.getByText(own.token)).toBeVisible();
  const csv = parseCsv(await readFile(await (await downloaded).path(), "utf8"));
  expect(downloads).toHaveLength(1);
  expect(csv[1]?.[3]).toBe(own.token);
  expect(await shown()).toEqual([own.token]);
  await expectTokenGone(page, cancelled.token, log);
  await expectTokenGone(page, late.token, log);

  // A close cleans the modal up 350 ms later, which must not touch an opening that came meanwhile: closed and
  // opened again at once, the modal keeps showing the token it creates, past that cleanup. The page's clock stops
  // at the close. It moves a frame at a time while the modal leaves (its transition waits for frames, and for
  // CSS that runs in real time), far from 350 ms, and not at all while the next token is created.
  await page.clock.pauseAt(await page.evaluate(() => Date.now() + 1_000));
  await page.getByRole("button", { name: "Close" }).click();
  let moved = 0;
  await expect
    .poll(
      async () => {
        await page.clock.runFor(16);
        moved += 16;
        return page.getByText(own.token).count();
      },
      { intervals: [40], timeout: 5_000 }
    )
    .toBe(0);
  expect(moved, "the modal left before the close's cleanup came due").toBeLessThan(350);
  await page.getByRole("button", { name: "Add access token" }).first().click();
  await page.getByLabel("Title").fill("next");
  await page.getByRole("switch", { name: "Never expires" }).click();
  const nextAnswer = await answerTo(page, "POST", "/api/v0/me/api-tokens", () =>
    page.getByRole("button", { name: "Generate token" }).click()
  );
  expect(nextAnswer.status()).toBe(201);
  const next = (await nextAnswer.json()) as components["schemas"]["ApiTokenCreated"];
  await expect(page.getByText(next.token)).toBeVisible();
  // The clock runs past the cleanup. React renders what a timer changed in a task it posts as the timer runs:
  // a message posted after the clock has run arrives after that render.
  await page.clock.runFor(1_000);
  await page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        const channel = new MessageChannel();
        channel.port1.addEventListener("message", () => resolve());
        channel.port1.start();
        channel.port2.postMessage(undefined);
      })
  );
  await expect(page.getByText(next.token)).toBeVisible();
  await expect(page.getByLabel("Title")).toHaveCount(0);
  await page.clock.resume();
  await expect(page.getByText(next.token)).toBeVisible();
  expect(await shown()).toEqual([own.token, next.token]);
  expect(downloads).toHaveLength(2);

  expect(watch.apiFailures).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("A11 (page): the list is the account's own: when another tab signs another account in, it shows that account's", async ({
  api,
  context,
  signedInPage,
}, testInfo) => {
  const x = emailFor(testInfo, "x");
  const y = emailFor(testInfo, "y");
  const tabA = await signedInPage(await registerOnboarded(api, x));
  const watch = await watchPage(tabA);
  await createPAT(api, (await login(api, x)).access_token, { label: "token of x" });
  await createPAT(api, (await registerOnboarded(api, y)).access_token, { label: "token of y" });
  await tabA.goto("/settings/profile/api-tokens");
  await expect(tabA.getByText("token of x")).toBeVisible();

  // Tab B keeps a sign-in of Y as the token manager does; tab A follows, and its list is Y's.
  const tabB = await context.newPage();
  await tabB.goto("/site.webmanifest.json");
  const listed = answerTo(tabA, "GET", "/api/v0/me/api-tokens", async () => {
    await writeRecord(tabB, newRecord(await login(api, y)));
  });
  expect((await listed).status()).toBe(200);
  await expect(tabA.getByText("token of y")).toBeVisible();
  await expect(tabA.getByText("token of x")).toHaveCount(0);
  await expect(tabA).toHaveURL("/settings/profile/api-tokens");

  expect(watch.apiFailures).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  await expectQuietConsole(tabA, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("A11 (API): a token creates, lists page by page and revokes another; a revoked or expired token fails", async ({
  api,
  db,
}, testInfo) => {
  const admin = await createPAT(api, (await register(api, emailFor(testInfo))).access_token, { label: "admin" });
  const expiredAt = new Date(Date.now() + weekMs).toISOString();

  const created = await createPAT(api, admin.token, { label: "deploy", description: "ci", expired_at: expiredAt });

  expect(created).toMatchObject({ label: "deploy", description: "ci", last_used: null });
  expect(created.token).toMatch(/^nrv_pat_[A-Za-z0-9_-]{43}$/);
  await expectTokenStored(db, created.id, created.token, expiredAt);

  // The list, a token a page, newest first, never shows a token itself.
  const list = async (cursor?: string): Promise<components["schemas"]["ApiToken"][]> => {
    const { data, response } = await api.GET("/api/v0/me/api-tokens", {
      params: { query: { limit: 1, cursor } },
      headers: bearer(admin.token),
    });
    expect(response.status).toBe(200);
    const page = data?.data ?? [];
    return data?.next_cursor ? [...page, ...(await list(data.next_cursor))] : page;
  };
  const listed = await list();
  expect(listed.map((t) => t.id)).toEqual([created.id, admin.id]);
  expect(JSON.stringify(listed)).not.toContain(created.token);

  // The token authenticates, and its use is recorded.
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(200);
  const [used] = await db.query<{ last_used: Date | null }>("SELECT last_used FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(used?.last_used).not.toBeNull();

  // Revoked, it stops at once and leaves the list.
  const revoked = await api.DELETE("/api/v0/api-tokens/{token_id}", {
    params: { path: { token_id: created.id } },
    headers: bearer(admin.token),
  });
  expect(revoked.response.status).toBe(204);
  const [row] = await db.query<{ deleted_at: Date | null }>("SELECT deleted_at FROM api_tokens WHERE id = $1", [
    created.id,
  ]);
  expect(row?.deleted_at).not.toBeNull();
  expect((await api.GET("/api/v0/me", { headers: bearer(created.token) })).response.status).toBe(401);
  expect((await list()).map((t) => t.id)).toEqual([admin.id]);

  // Another account's token is not found, and it keeps working.
  const other = await createPAT(api, (await register(api, emailFor(testInfo, "other"))).access_token);
  const foreign = await api.DELETE("/api/v0/api-tokens/{token_id}", {
    params: { path: { token_id: other.id } },
    headers: bearer(admin.token),
  });
  expect(foreign.response.status).toBe(404);
  expect(foreign.error?.code).toBe("identity.api_token_not_found");
  expect((await api.GET("/api/v0/me", { headers: bearer(other.token) })).response.status).toBe(200);

  // A token past its expiry fails too.
  const old = await createPAT(api, admin.token, { label: "old" });
  await db.query("UPDATE api_tokens SET expired_at = now() - interval '1 minute' WHERE id = $1", [old.id]);
  expect((await api.GET("/api/v0/me", { headers: bearer(old.token) })).response.status).toBe(401);
});
