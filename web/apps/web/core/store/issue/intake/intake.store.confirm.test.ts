/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { IssueIntake } from "@nerve/api-client";
import { FakeNerve, answered, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { inTurn } from "@/store/fake-queue";
import {
  acme,
  beta,
  countsOf,
  countsPath,
  detailOf,
  intakeIssueOf,
  intakeTab,
  listPath,
  loadCounts,
  loadList,
  open,
  pageOf,
  snoozedToo,
} from "@/store/issue/intake/fake-intake";
import type { IntakeFilters } from "@/store/issue/intake/filters";
import type { IIntakeStore } from "@/store/issue/intake/intake.store";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));

// The confirmations of the intake store (P5a spec 2.3, 2.4, R8, R31; the contract's trap C7), which the write seam
// makes once nerve answers a write of a work item of an intake. nerve's answer is written into every list the store
// holds of the work item's project: in its place where the list's filters still select it at the moment, out of each
// list that no longer does, its total one less; into none that lacks it, which its next fetch brings. A fetch out
// meanwhile gets the confirmation again on its answer. The counts are fetched again only where this generation holds
// them, never through the counts' gate, without holding the write up.

const NOW = new Date("2026-11-10T09:00:00Z");
const EARLIER = "2026-11-10T08:00:00Z";
const LATER = "2026-11-10T09:30:00Z";
const TOMORROW = "2026-11-11T00:00:00Z";
const ME = "u-me";

beforeEach(() => {
  vi.useFakeTimers({ now: NOW });
});
afterEach(() => {
  vi.useRealTimers();
});

const snoozed: IssueIntake = { status: "snoozed", snoozed_till: TOMORROW, duplicate_to_id: null };
const accepted: IssueIntake = { status: "accepted", snoozed_till: null, duplicate_to_id: null };
const [one, two, three, four, five, six] = ["one", "two", "three", "four", "five", "six"].map((name) =>
  intakeIssueOf(name)
);
/** Snoozed till a moment come: nerve lists it as pending. */
const woke = intakeIssueOf("woke", { intake: { status: "snoozed", snoozed_till: EARLIER, duplicate_to_id: null } });
const dozing = intakeIssueOf("dozing", { intake: snoozed });
const done = intakeIssueOf("done", { intake: accepted });
const mine = intakeIssueOf("mine", { assignee_ids: [ME] });

const closed: IntakeFilters = { ...open, tab: "closed", statuses: ["accepted", "declined", "duplicate"] };
const assignedToMe: IntakeFilters = { ...snoozedToo, assignees: ["me"] };
const theirs = pageOf([intakeIssueOf("api-one", { project_id: "p-api" })]);
const docs = pageOf([intakeIssueOf("docs-one", { project_id: "p-docs", workspace_id: beta.id })]);

/**
 * A tab whose store holds four lists of p-web's intake in acme (the open tab, with the snoozed too, the closed tab,
 * those assigned to the caller), p-api's open tab in acme, and p-docs's in beta.
 */
async function heldLists() {
  const tab = await intakeTab();
  const { nerve, store } = tab;
  await loadList(nerve, store, "p-web", open, pageOf([one, two, three, woke], "c-1", 6));
  await loadList(nerve, store, "p-web", snoozedToo, pageOf([one, two, three, woke, dozing], null, 5));
  await loadList(nerve, store, "p-web", closed, pageOf([done]));
  await loadList(nerve, store, "p-web", assignedToMe, pageOf([mine]));
  await loadList(nerve, store, "p-api", open, theirs);
  await answered(nerve, () => store.fetchList(beta, "p-docs", open), ["GET", listPath("p-docs")], docs, "beta's list");
  return tab;
}

/** Every list the store holds. */
const held = (store: IIntakeStore) => ({
  open: store.getList("p-web", open),
  snoozedToo: store.getList("p-web", snoozedToo),
  closed: store.getList("p-web", closed),
  mine: store.getList("p-web", assignedToMe),
  theirs: store.getList("p-api", open),
  docs: store.getList("p-docs", open),
});

describe("IntakeStore, the confirmations", () => {
  it("writes an edit in its place in every list of its project that holds it, as a row; the totals as they were", async () => {
    const { store } = await heldLists();
    const before = held(store);
    store.confirm(detailOf("two", { name: "Sign-in fails", updated_at: LATER }));
    const renamed = intakeIssueOf("two", { name: "Sign-in fails", updated_at: LATER });
    expect(held(store)).toEqual({
      ...before,
      open: pageOf([one, renamed, three, woke], "c-1", 6),
      snoozedToo: pageOf([one, renamed, three, woke, dozing], null, 5),
    });
    // the list's row, not the detail: no HTML, no subscription
    expect(store.getList("p-web", snoozedToo)?.data[1]).toStrictEqual(renamed);
    // another project's list, another workspace's, are not so much as written
    expect(store.getList("p-api", open)).toBe(before.theirs);
    expect(store.getList("p-docs", open)).toBe(before.docs);
  });

  it("takes a triage out of each list that no longer selects it, its total one less; into none that lacks it (R31)", async () => {
    const { store } = await heldLists();
    const before = held(store);
    store.confirm(detailOf("two", { intake: snoozed }));
    const twoSnoozed = intakeIssueOf("two", { intake: snoozed });
    expect(held(store)).toEqual({
      ...before,
      open: pageOf([one, three, woke], "c-1", 5),
      snoozedToo: pageOf([one, twoSnoozed, three, woke, dozing], null, 5),
    });

    // accepted: out of the open tab's lists; the closed tab's list lacks it, and only its next fetch brings it
    store.confirm(detailOf("three", { intake: accepted, state_id: "s-todo" }));
    expect(held(store)).toEqual({
      ...before,
      open: pageOf([one, woke], "c-1", 4),
      snoozedToo: pageOf([one, twoSnoozed, woke, dozing], null, 4),
      closed: pageOf([done]),
    });
    expect(store.getList("p-api", open)).toBe(before.theirs);
    expect(store.getList("p-docs", open)).toBe(before.docs);
  });

  it("selects by the status at the moment: a snooze whose time has come stays where the pending are listed (C7)", async () => {
    const { store } = await heldLists();
    const before = held(store);
    const edit = { name: "Woken", updated_at: LATER };
    store.confirm(detailOf("woke", { ...edit, intake: woke.intake }));
    const renamed = intakeIssueOf("woke", { ...edit, intake: woke.intake });
    expect(held(store)).toEqual({
      ...before,
      open: pageOf([one, two, three, renamed], "c-1", 6),
      snoozedToo: pageOf([one, two, three, renamed, dozing], null, 5),
    });
  });

  it("reads the filters' me as the caller: an edit that unassigns him takes it out of his list", async () => {
    const { store } = await heldLists();
    const before = held(store);
    store.confirm(detailOf("mine", { assignee_ids: [ME, "u-ann"], updated_at: LATER }));
    const shared = intakeIssueOf("mine", { assignee_ids: [ME, "u-ann"], updated_at: LATER });
    expect(held(store)).toEqual({ ...before, mine: pageOf([shared]) });
    store.confirm(detailOf("mine", { assignee_ids: ["u-ann"], updated_at: LATER }));
    expect(held(store)).toEqual({ ...before, mine: pageOf([]) });
  });

  it("takes a work item no longer of an intake (its intake row gone) out of every list", async () => {
    const { store } = await heldLists();
    const before = held(store);
    store.confirm(detailOf("done", { intake: null }));
    store.confirm(detailOf("two", { intake: null }));
    expect(held(store)).toEqual({
      ...before,
      open: pageOf([one, three, woke], "c-1", 5),
      snoozedToo: pageOf([one, three, woke, dozing], null, 4),
      closed: pageOf([]),
    });
  });

  it("drops a work item from every list of its project, one less in each total that held it; others as they were", async () => {
    const { store } = await heldLists();
    const before = held(store);
    store.drop("p-web", two.id);
    const after = {
      ...before,
      open: pageOf([one, three, woke], "c-1", 5),
      snoozedToo: pageOf([one, three, woke, dozing], null, 4),
    };
    expect(held(store)).toEqual(after);
    expect(store.getList("p-api", open)).toBe(before.theirs);
    expect(store.getList("p-docs", open)).toBe(before.docs);

    // a work item no list holds (selected by its address, on a page not read): no total changes till the next fetch
    store.drop("p-web", "i-unread");
    store.confirm(detailOf("unread", { intake: accepted }));
    expect(held(store)).toEqual(after);
  });

  it("makes a confirmation again, once, on the answer of a fetch out meanwhile: a refresh's, a first fetch's", async () => {
    const { nerve, store } = await heldLists();
    // a refresh is out, read before two is snoozed: nerve lists four, new, and two still
    const k = nerve.calls.length;
    const refresh = track(store.fetchList(acme, "p-web", open));
    await until(() => nerve.calls.length === k + 1, "the refresh");
    store.confirm(detailOf("two", { intake: snoozed }));
    expect(store.getList("p-web", open)).toEqual(pageOf([one, three, woke], "c-1", 5));
    nerve.calls[k]?.answer(json(200, pageOf([four, one, two, three, woke], "c-2", 7)));
    await until(() => refresh.settled, "the refresh");
    expect(store.getList("p-web", open)).toEqual(pageOf([four, one, three, woke], "c-2", 6));

    // another, read after three is accepted: it lacks three, and its total is nerve's
    const again = track(store.fetchList(acme, "p-web", open));
    await until(() => nerve.calls.length === k + 2, "the refresh again");
    store.confirm(detailOf("three", { intake: accepted }));
    nerve.calls[k + 1]?.answer(json(200, pageOf([four, one, woke], "c-2", 5)));
    await until(() => again.settled, "the refresh again");
    expect(store.getList("p-web", open)).toEqual(pageOf([four, one, woke], "c-2", 5));

    // a first fetch is out, read before one is renamed: the rename is made on its answer
    const recent: IntakeFilters = { ...open, order: "-updated_at" };
    const first = track(store.fetchList(acme, "p-web", recent));
    await until(() => nerve.calls.length === k + 3, "the first fetch");
    store.confirm(detailOf("one", { name: "Renamed", updated_at: LATER }));
    nerve.calls[k + 2]?.answer(json(200, pageOf([one, four, woke], "c-r")));
    await until(() => first.settled, "the first fetch");
    const renamed = intakeIssueOf("one", { name: "Renamed", updated_at: LATER });
    expect(store.getList("p-web", recent)).toEqual(pageOf([renamed, four, woke], "c-r"));
  });

  it("makes a confirmation again, once, on the next page out meanwhile", async () => {
    const { nerve, store } = await heldLists();
    // the next page is out, read before five, on it, is accepted and six renamed
    const m = nerve.calls.length;
    const more = track(store.fetchMore("p-web", open));
    await until(() => nerve.calls.length === m + 1, "the next page");
    expect(nerve.calls[m]?.query.cursor).toBe("c-1");
    store.confirm(detailOf("five", { intake: accepted }));
    store.confirm(detailOf("six", { name: "Six", updated_at: LATER }));
    nerve.calls[m]?.answer(json(200, pageOf([five, six], null, 6)));
    await until(() => more.settled, "the next page");
    const renamed = intakeIssueOf("six", { name: "Six", updated_at: LATER });
    expect(store.getList("p-web", open)).toEqual(pageOf([one, two, three, woke, renamed], null, 5));
  });
});

describe("IntakeStore, the counts again", () => {
  it("fetches again the counts this generation holds of the workspace, those alone, and holds no caller up", async () => {
    const { nerve, store } = await intakeTab();
    await loadCounts(nerve, store, acme, countsOf({ "p-web": 3 }));
    // beta's counts were never fetched (its gate shut): none is asked for
    expect(store.countAgain(beta)).toBeUndefined();
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(1);

    // acme's: the call gives nothing to wait for while the request is out; then nerve's answer shows
    expect(store.countAgain(acme)).toBeUndefined();
    await inTurn(nerve, 1, ["GET", countsPath(acme)], json(200, countsOf({ "p-web": 2 })));
    await until(() => store.pendingCount(acme.slug, "p-web") === 2, "the counts again");
    expect(nerve.calls.map((call) => call.path)).toEqual([countsPath(acme), countsPath(acme)]);
  });

  it("leaves the counts as they were when the fetch again fails, failing no one; asks nothing once the session changed", async () => {
    const { nerve, api, store } = await intakeTab();
    await loadCounts(nerve, store, acme, countsOf({ "p-web": 3 }));
    store.countAgain(acme);
    await inTurn(nerve, 1, ["GET", countsPath(acme)], problem(503, "server_busy"));
    await vi.advanceTimersByTimeAsync(1_000);
    expect(store.pendingCount(acme.slug, "p-web")).toBe(3);

    FakeNerve.replaceSession(api);
    store.countAgain(acme);
    await vi.advanceTimersByTimeAsync(1_000);
    expect(nerve.calls).toHaveLength(2);
    expect(store.pendingCount(acme.slug, "p-web")).toBe(3);
  });
});
