import { byId } from "../../fixtures/assert/issue";
import {
  expectModuleRows,
  expectPlanningActivity,
  lastWrite,
  moduleJoined,
  moduleLeft,
  wrote,
} from "../../fixtures/assert/issue-planning";
import { watchPage } from "../../fixtures/browser";
import { modalDialog } from "../../fixtures/dialogs";
import { ascendingIds } from "../../fixtures/issue-planning";
import { updateIssue } from "../../fixtures/issues";
import { expectListPageQuiet } from "../../fixtures/issue-list-page";
import { chooseFromRow, moduleListWorld } from "../../fixtures/planning-list-pages";
import {
  answerTo,
  bodiesSentTo,
  expectListBesideButton,
  holdAnswer,
  transitionsEnded,
} from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G11's page version (M6 P7a spec 2.2, 2.4–2.6, 2.11, R3, R12, R13; M2 handoff 5(a), M3 handoff 3): a work item's
// modules as mem uses them in his browser. Its preview's modules dropdown, beside its button with its search focused,
// offers the modules it may join, those it is in where they are, and an archived one it is in only to be taken out;
// each pick is one module added or taken out, worked out in the work item's turn from the set it then holds, so that a
// pick made while the one before is out builds on that one's answer, and the archived module stays. A row's edit
// sends the form's change, applied to the set the work item holds, a module joined meanwhile kept. The archived module
// picked in the row's list is one change taking it out alone; once nerve confirms it, it is offered no more (re-73
// M3). The API version is g11-issue-modules.spec.ts; both assert the same rows. The world is made through the API,
// nerve's answers read back through the database.

/** A module as a work item's modules dropdown names it: an archived one captioned. */
type Named = { id: string; name: string; archived?: boolean };

/**
 * The name of a button of a work item's modules that holds several, in nerve's order (the ids'): how many, then each
 * module's name, an archived one captioned. Its parts are boxes of their own, which the name parts by a space as well.
 */
function modulesNamed(...held: Named[]): RegExp {
  const names = held.toSorted(byId).map((m) => (m.archived ? `${m.name} ?, Archived` : m.name));
  return new RegExp(`^Modules: ${held.length} modules ?: ${names.join(" ?; ")}$`);
}
/** The name of a row's control: the button's (named), then the row's (row), a box of its own. */
const inRow = (named: RegExp, row: string) => new RegExp(`${named.source.slice(0, -1)} ?, ${row}$`);

test("G11 (page): Write the guide's preview on Web's list shows its modules Alpha and Gamma, archived; the list, beside its button with its search focused, offers Alpha, picked, Beta and Delta, and Gamma only to be taken out; Beta picked and then Alpha while the first change is out send Alpha, Gamma and Beta, then Gamma and Beta from the first one's answer, Gamma kept; Delta joined behind the page, its row's edit takes Beta out of the form and sends Gamma and Delta, Delta kept; Gamma picked in its row's list takes it out alone, Delta kept, and is offered no more", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const w = await moduleListWorld(api, testInfo);
  const [alpha, beta, delta] = [await w.module("Alpha"), await w.module("Beta"), await w.module("Delta")];
  const gamma = await w.module("Gamma");
  // Write the guide joins Alpha and Gamma, which the admin completes and archives.
  const item = await w.item("Write the guide", { module_ids: [alpha.id, gamma.id] });
  const created = await lastWrite(db, item.id);
  await w.archive(gamma.id);
  const shelved = { ...gamma, archived: true };

  const page = await signedInPage(w.mem.tokens);
  const watch = await watchPage(page);
  await page.goto(w.projectPath);
  const options = page.getByRole("option");

  // The preview names its modules, Gamma captioned. Opened once the preview has come in with what it shows above them
  // after its first render (the description's editor, its last edit, the work item's creator), so that nothing moves
  // the button from under its list, the list, beside the button with its search focused, offers the modules it may
  // join in nerve's order, Alpha picked, and Gamma, archived, after them, picked.
  await page.locator(`#issue-${item.id}`).click();
  const peek = page.locator("#full-screen-portal");
  await expect(peek.locator(`#editor-container-${item.id} [contenteditable="true"]`)).toBeVisible();
  await expect(peek.getByRole("button", { name: /^Last edited by/ })).toBeVisible();
  await expect(peek.getByText("Created by", { exact: true })).toBeVisible();
  await transitionsEnded(peek);
  const modules = (...held: Named[]) => peek.getByRole("button", { name: modulesNamed(...held) });
  // from now on, the first change's answer held, and every change's body read as it leaves: routed once the page has
  // loaded, which it does not leave (a route installed before makes WebKit fetch the modules it preloaded again)
  const path = `/api/v0/issues/${item.id}`;
  const release = await holdAnswer(page, "PATCH", path);
  const changes = await bodiesSentTo(page, "PATCH", path);
  await modules(alpha, shelved).click();
  await expectListBesideButton(page, modules(alpha, shelved));
  await expect(page.getByRole("combobox", { name: "Search" })).toBeFocused();
  await expect(options).toHaveText(["Delta", "Beta", "Alpha", "Gamma, Archived"]);
  expect(await options.evaluateAll((all) => all.map((option) => option.getAttribute("aria-selected")))).toEqual([
    "false",
    "false",
    "true",
    "true",
  ]);

  // Beta, then Alpha, picked while the first change is out: the second is sent from the first one's answer.
  await page.getByRole("option", { name: "Beta", exact: true }).click();
  await expect.poll(() => changes.length, "the first change sent").toBe(1);
  await page.getByRole("option", { name: "Alpha", exact: true }).click();
  await release();
  await expect.poll(() => changes.length, "the second change sent once the first is answered").toBe(2);
  expect(changes).toEqual([
    { module_ids: [...ascendingIds(alpha, gamma), beta.id] },
    { module_ids: ascendingIds(gamma, beta) },
  ]);
  await expect(modules(shelved, beta)).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(modules(shelved, beta)).toBeFocused();
  const left = await lastWrite(db, item.id);
  // the moment of the first change: that of Beta's row, Alpha's removal being the second's
  const [first] = await db.query<{ at: string }>(
    "SELECT created_at::text AS at FROM module_issues WHERE issue_id = $1 AND module_id = $2",
    [item.id, beta.id]
  );
  if (!first) throw new Error("no row of Beta for Write the guide");
  const joined = first.at;
  // In the database Gamma stays, Beta joined and Alpha left, a row of activity each.
  const rows = [
    { target: "Alpha", added: created.at, removed: left.at, adder: w.admin.email, by: w.mem.email },
    { target: "Gamma", added: created.at, removed: null, adder: w.admin.email, by: w.admin.email },
  ];
  const activity = [
    ...wrote(w.admin.email, created.at, ...[alpha, gamma].toSorted(byId).map((m) => moduleJoined(m.name))),
    ...wrote(w.mem.email, joined, moduleJoined("Beta")),
    ...wrote(w.mem.email, left.at, moduleLeft("Alpha")),
  ];
  await expectModuleRows(db, item.id, [
    ...rows,
    { target: "Beta", added: joined, removed: null, adder: w.mem.email, by: w.mem.email },
  ]);
  await expectPlanningActivity(db, item.id, activity);

  // Delta joins behind the page; the row's edit reads Write the guide as it is now, and Beta taken out of the form
  // sends the form's change applied to the set it holds: Gamma and Delta.
  await updateIssue(api, w.token, item.id, { module_ids: [gamma.id, beta.id, delta.id] });
  const added = await lastWrite(db, item.id);
  await peek.getByRole("button", { name: "Close the peek view", exact: true }).click();
  const read = await answerTo(page, "GET", path, () => chooseFromRow(page, "Write the guide", "Edit"));
  expect(((await read.json()) as { module_ids: string[] }).module_ids, "the dialog's read").toEqual(
    ascendingIds(gamma, beta, delta)
  );
  const form = modalDialog(page);
  // the dialog's list opens beside its button once the dialog has come in, its description's editor above the fields
  await expect(form.locator('[contenteditable="true"]')).toBeVisible();
  await transitionsEnded(form);
  await form.getByRole("button", { name: modulesNamed(shelved, beta) }).click();
  await expectListBesideButton(page, form.getByRole("button", { name: modulesNamed(shelved, beta) }));
  await page.getByRole("option", { name: "Beta", exact: true }).click();
  await page.keyboard.press("Escape");
  await expect(form.getByRole("button", { name: /^Modules: Gamma ?, Archived$/ })).toBeVisible();
  await form.getByRole("button", { name: "Update", exact: true }).click();
  await expect(form).toHaveCount(0);
  expect(changes).toHaveLength(3);
  expect(changes[2]).toMatchObject({ module_ids: ascendingIds(gamma, delta) });
  expect(changes[2]).not.toHaveProperty("cycle_id");
  const edited = await lastWrite(db, item.id);

  // In the database Delta stays and Beta left, by the edit.
  const later = [
    { target: "Beta", added: joined, removed: edited.at, adder: w.mem.email, by: w.mem.email },
    { target: "Delta", added: added.at, removed: null, adder: w.admin.email, by: w.admin.email },
  ];
  await expectModuleRows(db, item.id, [...rows, ...later]);
  const edits = [
    ...activity,
    ...wrote(w.admin.email, added.at, moduleJoined("Delta")),
    ...wrote(w.mem.email, edited.at, moduleLeft("Beta")),
  ];
  await expectPlanningActivity(db, item.id, edits);

  // Gamma, archived, which the row's list has after the options, picked: picking it is one change that takes it alone
  // out of the set the work item holds, Delta kept; once nerve confirms it, the list offers it no more.
  await page.getByRole("button", { name: inRow(modulesNamed(delta, shelved), "Write the guide") }).click();
  await expect(options).toHaveText(["Delta", "Beta", "Alpha", "Gamma, Archived"]);
  await page.getByRole("option", { name: /^Gamma ?, Archived$/ }).click();
  await expect(page.getByRole("button", { name: inRow(/^Modules: Delta$/, "Write the guide") })).toBeVisible();
  await expect(options).toHaveText(["Delta", "Beta", "Alpha"]);
  expect(changes).toHaveLength(4);
  expect(changes[3]).toEqual({ module_ids: [delta.id] });
  await page.keyboard.press("Escape");
  const taken = await lastWrite(db, item.id);

  // In the database Gamma left, by mem's pick, Delta staying.
  await expectModuleRows(db, item.id, [rows[0]!, { ...rows[1]!, removed: taken.at, by: w.mem.email }, ...later]);
  await expectPlanningActivity(db, item.id, [...edits, ...wrote(w.mem.email, taken.at, moduleLeft("Gamma"))]);
  await expectListPageQuiet(page, watch);
});
