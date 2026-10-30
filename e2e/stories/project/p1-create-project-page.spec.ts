import { createProject, createWorkspace, inviteAndAccept, slugFor } from "../../fixtures/api";
import { expectProjectCreated } from "../../fixtures/assert/project";
import { accountId, emailFor, named } from "../../fixtures/auth";
import {
  EMOJI_CHECK_WARNING,
  expectQuietConsole,
  recordFocuses,
  requestsElsewhere,
  watchPage,
} from "../../fixtures/browser";
import {
  closedByComposingEscape,
  closedByEscape,
  composingEscape,
  hiddenWithin,
  modalDialog,
} from "../../fixtures/dialogs";
import { pageContent, tabCycle } from "../../fixtures/project-pages";
import {
  expectListBesideButton,
  keydownsReachingDocument,
  registerOnboarded,
  sentHeld,
  sentTo,
  transitionsEnded,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// P1's page version (M3 design 2, 3.17, 3.18; P10): a project created from the projects page's dialog, its icon
// picker's emoji, which nerve serves, and the dialog's Tab cycle and lead's list. The API version is
// p1-create-project.spec.ts.

/** Where nerve serves the emoji picker's data (M3 design 7.7). */
const EMOJIBASE = "/assets/emojibase/15.3.2/en";

test("P1 (page): a new project's icon picker shows the emoji nerve serves itself, nothing asked of another address, nothing blocked", async ({
  api,
  signedInPage,
}, testInfo) => {
  const member = await registerOnboarded(api, emailFor(testInfo, "member"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, member.access_token, { name: "Acme", slug });
  const page = await signedInPage(member);
  const watch = await watchPage(page);

  await page.goto(`/${slug}/projects`);
  await page.getByRole("button", { name: "Add Project", exact: true }).click();
  const data = page.waitForResponse((answer) => new URL(answer.url()).pathname === `${EMOJIBASE}/data.json`);
  const messages = page.waitForResponse((answer) => new URL(answer.url()).pathname === `${EMOJIBASE}/messages.json`);
  await page.getByRole("button", { name: "Project icon" }).click();
  expect([(await data).status(), (await messages).status()]).toEqual([200, 200]);
  const search = page.getByRole("searchbox");
  await search.fill("rocket");
  // An Escape that ends an input method's composition in the search is the input method's: the picker stays open.
  const pickerClosed = hiddenWithin(search);
  await composingEscape(search);
  expect(await pickerClosed).toBe(false);
  await page.getByRole("gridcell", { name: "Rocket" }).click();
  await expect(page.getByRole("button", { name: "Project icon" })).toContainText("🚀");

  // The picker asks nerve, which answers its HEAD too (frimousse compares the files' ETags before it reads its cache).
  expect((await page.request.head(`${EMOJIBASE}/data.json`)).status()).toBe(200);
  expect(requestsElsewhere(page, watch)).toEqual([]);
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});

test("P1 (page): a member creates a project from the projects page, its identifier upper case as typed and its lead an admin, not a guest; a taken identifier and a forbidden name show under their fields; the form waits for nerve; Tab goes round the features step; the project shows in the list and the sidebar; in no project, the member is offered a project to create, the guest none", async ({
  api,
  baseURL,
  browser,
  db,
  signedInPage,
}, testInfo) => {
  const adminEmail = emailFor(testInfo, "admin");
  const admin = await registerOnboarded(api, adminEmail);
  const memberEmail = emailFor(testInfo, "member");
  const member = await registerOnboarded(api, memberEmail);
  const guestEmail = emailFor(testInfo, "guest");
  const guest = await registerOnboarded(api, guestEmail);
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.access_token, { name: "Acme", slug, timezone: "Asia/Shanghai" });
  await inviteAndAccept(api, admin.access_token, slug, { email: memberEmail, token: member.access_token }, 15);
  await inviteAndAccept(api, admin.access_token, slug, { email: guestEmail, token: guest.access_token }, 5);
  await named(api, admin.access_token, { display_name: "ada" });
  await named(api, guest.access_token, { display_name: "gus" });
  // WEB is taken by the time the member sends it
  await createProject(api, admin.access_token, slug, { name: "Site", identifier: "WEB" });
  const adminId = await accountId(api, admin.access_token);

  const page = await signedInPage(member);
  const watch = await watchPage(page);
  // The member, in no project yet, finds the project settings saying so, and offering him a project's creation.
  await page.goto(`/${slug}/settings/projects`);
  await expect(page.getByText("No project", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Start your first project" })).toBeVisible();
  await page.goto(`/${slug}/projects`);
  await page.getByRole("button", { name: "Add Project", exact: true }).click();
  await transitionsEnded(page.getByRole("dialog"));
  // its form has no heading: a hidden title names the dialog
  await expect(modalDialog(page)).toHaveAccessibleName("Create project");
  // Sent empty, the name's rule fails: the name takes the focus, already read invalid, its message its description (as
  // recorded when it takes it, which is when a screen reader reads it), and nothing is sent.
  const create = page.getByRole("button", { name: "Create project" });
  let focuses = await recordFocuses(page);
  await create.click();
  await expect(page.locator("#name")).toBeFocused();
  expect((await focuses()).at(-1)).toEqual({ id: "name", invalid: "true", description: "Name is required" });
  // The name gives the identifier, as typed; the identifier typed is upper case, of the characters it may have.
  await page.locator("#name").fill("Web-App");
  // An Escape that ends an input method's composition in the name is the input method's: the modal stays open.
  expect(await closedByComposingEscape(page.locator("#name"))).toBe(false);
  await expect(page.locator("#identifier")).toHaveValue("WEBAPP");
  await page.locator("#identifier").fill("we.b");
  await expect(page.locator("#identifier")).toHaveValue("WEB");
  await page.locator("#description").fill("The site");
  // The lead: the form's order puts it right before Cancel, which Shift+Tab leaves for it; the workspace's admin or a
  // member, not its guest.
  await page.getByRole("button", { name: "Cancel" }).focus();
  await page.keyboard.press("Shift+Tab");
  await expect(page.getByRole("button", { name: "Lead", exact: true }).last()).toBeFocused();
  await page.keyboard.press("Enter");
  // each after its avatar's initial
  await expect(page.getByRole("option")).toHaveText([/You$/, /ada$/]);
  await page.getByRole("option", { name: "ada" }).click();
  // The icon: an emoji from the picker.
  await page.getByRole("button", { name: "Project icon" }).click();
  await page.getByRole("searchbox").fill("rocket");
  await page.getByRole("gridcell", { name: "Rocket" }).click();

  // Refused: the name nerve does not allow and the identifier taken, each under its field.
  const refusedName = await sentTo(page, "POST", `/api/v0/workspaces/${slug}/projects`, () => create.click());
  expect(refusedName.answer.status()).toBe(422);
  await expect(page.getByText("Not allowed")).toBeVisible();
  await page.locator("#name").fill("Web App");
  await page.locator("#identifier").fill("web");
  // While nerve has it, its button is busy and the dialog has the focus (keepFocusIn).
  const refusing = await sentHeld(page, "POST", `/api/v0/workspaces/${slug}/projects`, () => create.click());
  await expect(page.getByRole("button", { name: "Creating" })).toBeDisabled();
  await expect(modalDialog(page)).toBeFocused();
  focuses = await recordFocuses(page);
  expect((await refusing.release()).status()).toBe(409);
  const taken = "A project of this workspace already has this identifier.";
  await expect(page.getByText(taken)).toBeVisible();
  // the field refused takes the focus, already read invalid, its message its description; nothing takes it first, not
  // even the button as it is enabled again (keepFocusIn's return waits for the refusal's focus)
  await expect(page.locator("#identifier")).toBeFocused();
  expect(await focuses()).toEqual([{ id: "identifier", invalid: "true", description: taken }]);
  await expect(page.locator("#identifier")).toHaveAccessibleDescription(taken);

  // Created: the form's fields alone, sent once; the form cannot be closed until nerve answers.
  await page.locator("#identifier").fill("webapp");
  const { body, release } = await sentHeld(page, "POST", `/api/v0/workspaces/${slug}/projects`, () => create.click());
  const form = {
    name: "Web App",
    identifier: "WEBAPP",
    description: "The site",
    network: 2,
    logo_props: { in_use: "emoji", emoji: { value: "128640" } },
  };
  expect(body).toEqual({ ...form, project_lead_id: adminId });
  // neither by Cancel, by the header's Close nor by Escape; nor sent again, its button busy
  await expect(page.getByRole("button", { name: "Cancel" })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Close", exact: true })).toBeDisabled();
  await expect(page.getByRole("button", { name: "Creating" })).toBeDisabled();
  expect(await closedByEscape(page)).toBe(false);
  expect((await release()).status()).toBe(201);
  await expect(page.getByText("Project created successfully")).toBeVisible();
  // The features step: Tab goes round it too, never out of the modal (the toast is a dialog, not modal).
  await page.getByRole("button", { name: "Close", exact: true }).focus();
  expect((await tabCycle(page, page.locator("[aria-modal=true]"))).join(", ")).toBe(
    "Open project, Cycles, Modules, Views, Intake, Close"
  );
  await page.getByRole("button", { name: "Close", exact: true }).click();
  // its card, by its heading (the card's link to its settings is named by the project too)
  await expect(pageContent(page).getByRole("heading", { name: "Web App", exact: true })).toBeVisible();
  await expect(page.getByRole("complementary", { name: "Main sidebar" }).getByText("Web App")).toBeVisible();

  // the project's row: the form's fields, in the workspace's time zone
  await expectProjectCreated(db, slug, { ...form, timezone: "Asia/Shanghai" }, memberEmail, adminEmail, [
    { email: memberEmail, sort_order: 65535 },
    { email: adminEmail, sort_order: 55535 },
  ]);

  // the two refusals, the browser's reports of them; a load of the project settings, and of the projects page
  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([
    [],
    [`422 POST /api/v0/workspaces/${slug}/projects`, `409 POST /api/v0/workspaces/${slug}/projects`],
    [],
    [],
  ]);
  await expectQuietConsole(page, watch, {
    warnings: [EMOJI_CHECK_WARNING, EMOJI_CHECK_WARNING],
    errors: [
      "Failed to load resource: the server responded with a status of 422 (Unprocessable Entity)",
      "Failed to load resource: the server responded with a status of 409 (Conflict)",
    ],
  });

  // The workspace's guest, in no project, finds the project settings saying so, with no project to create: nerve lets
  // the workspace's admins and members create one, not its guests.
  const theGuest = await anotherBrowser(browser, baseURL ?? "", guest);
  const guestWatch = await watchPage(theGuest.page);
  await theGuest.page.goto(`/${slug}/settings/projects`);
  await expect(theGuest.page.getByText("No project", { exact: true })).toBeVisible();
  await expect(theGuest.page.getByRole("button", { name: "Start your first project" })).toHaveCount(0);
  expect([guestWatch.apiFailures, guestWatch.oldApiRequests, guestWatch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(theGuest.page, guestWatch, { warnings: [EMOJI_CHECK_WARNING] });
  await theGuest.close();
});

test("P1 (page): Tab goes round the creation dialog and nowhere else; the lead's list opens beside its button with its search focused; Escape closes it", async ({
  api,
  signedInPage,
}, testInfo) => {
  const member = await registerOnboarded(api, emailFor(testInfo, "member"));
  const slug = slugFor(testInfo);
  await createWorkspace(api, member.access_token, { name: "Acme", slug });
  await createProject(api, member.access_token, slug, { name: "Ops", identifier: "OPS" });
  const page = await signedInPage(member);
  const watch = await watchPage(page);

  await page.goto(`/${slug}/projects`);
  await expect(pageContent(page).getByRole("heading", { name: "Ops", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Add Project", exact: true }).click();
  // Popper places the list where the button is as it opens: the modal's enter transition moves the button.
  await transitionsEnded(page.getByRole("dialog"));
  // Tab goes round the dialog's controls in the form's order, each once, and never out of it (to the page behind, or
  // to the closed menu of Ops's card); the network's select is one stop, its button, which Space opens.
  await page.locator("#name").focus();
  expect((await tabCycle(page, page.getByRole("dialog"))).join(", ")).toBe(
    "identifier, description, Public, Lead, Cancel, Create project, Close, Project icon, name"
  );
  await page.locator("#description").press("Tab");
  await expect(page.getByRole("dialog").getByRole("button", { name: "Public" })).toBeFocused();
  await page.keyboard.press("Space");
  await expect(page.getByRole("option")).toHaveText([/^Private/, /^Public/]);
  await page.keyboard.press("Escape");
  // The select owns one semantic trigger; its DropdownButton content is presentation only.
  const lead = page.getByRole("button", { name: "Lead", exact: true });
  await expect(lead).toHaveCount(1);
  await expect(lead.locator("xpath=ancestor::button")).toHaveCount(0);
  await lead.click();
  await expectListBesideButton(page, lead);
  const search = page.getByRole("combobox", { name: "Search" });
  await expect(search).toBeFocused();
  // Escape closes the list and gives its button the focus (shown); so does it once a search opened the Combobox.
  const escapes = await keydownsReachingDocument(page, "Escape");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(lead).toBeFocused();
  await expect(lead).toHaveCSS("outline-style", "solid");
  await lead.press("Enter");
  await expectListBesideButton(page, lead);
  await page.keyboard.type("nobody");
  // An Escape that ends an input method's composition (Chinese, say) is the input method's: the list stays open.
  await composingEscape(search);
  await page.keyboard.type("!");
  await expect(search).toHaveValue("nobody!");
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  expect(await escapes(), "the Escapes that reached the document, where the modal closes on one").toBe(0);
  // opened again, the list has no search
  await lead.press("Enter");
  await expect(search).toHaveValue("");
  await expect(page.getByRole("option", { name: "You" })).toBeVisible();

  expect([watch.cspViolations, watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], [], []]);
  await expectQuietConsole(page, watch, { warnings: [EMOJI_CHECK_WARNING] });
});
