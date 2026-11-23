// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import en from "../../../../../../../../../packages/i18n/src/locales/en/inbox.json";
import zh from "../../../../../../../../../packages/i18n/src/locales/zh-CN/inbox.json";
import { IssueActivityItem } from "../activity-list";
import type { IssueActivityBlockComponent } from "./helpers/activity-block";
import { IssueInboxActivity } from "./inbox";

// The activity row of a triage (P5a spec 2.7, R18; re-94 O1): nerve writes it on the field intake, its verb the
// status the work item was moved to, by its name (pending, which ends a snooze, snoozed, accepted, declined,
// duplicate); the row says each by its own text, through t(), in English and in Chinese. The contract leaves the verb
// an open string, so any other says that the intake status changed. Numbered statuses ("1" was accepted) and the
// field inbox are the old intake's, not nerve's.

const binding = vi.hoisted(() => ({
  activity: undefined as Record<string, unknown> | undefined,
  locale: "en" as "en" | "zh-CN",
  /** A locale's text of a dotted key, or undefined. */
  lookup: (tree: unknown, key: string): unknown =>
    key
      .split(".")
      .reduce<unknown>((node, part) => (node && typeof node === "object" ? Reflect.get(node, part) : undefined), tree),
}));
vi.mock("@nerve/i18n", () => ({
  useTranslation: () => ({
    t: (key: string) => {
      const text = binding.lookup(binding.locale === "en" ? en : zh, key);
      return typeof text === "string" ? text : key;
    },
  }),
}));
vi.mock("@/hooks/store/use-issue-detail", () => ({
  useIssueDetail: () => ({ activity: { getActivityById: () => binding.activity } }),
}));
vi.mock("./helpers/activity-block", () => ({
  IssueActivityBlockComponent: (props: ComponentProps<typeof IssueActivityBlockComponent>) => (
    <p data-row>{props.children}</p>
  ),
}));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

/** nerve's row of a triage to verb, on field, as the caller's locale says it. */
async function said(verb: string, locale: "en" | "zh-CN", field = "intake") {
  binding.locale = locale;
  binding.activity = { field, verb, old_value: "pending", new_value: verb, created_at: "2026-11-20T00:00:00Z" };
  // each row its own id: the observer draws it anew
  const id = `${field}-${verb}-${locale}`;
  await act(async () =>
    root.render(<IssueInboxActivity activityId={id} issueId="i-x" workspaceSlug="acme" ends={undefined} />)
  );
  return host.querySelector("[data-row]")?.textContent;
}
/** A locale's text of a key under inbox_issue.activity, or undefined. */
const text = (locale: "en" | "zh-CN", name: string): unknown =>
  binding.lookup(locale === "en" ? en : zh, `inbox_issue.activity.${name}`);

const STATUSES = ["pending", "snoozed", "accepted", "declined", "duplicate"] as const;
const LOCALES = ["en", "zh-CN"] as const;

describe("IssueInboxActivity, a triage's row by the status's name", () => {
  it.each(STATUSES.flatMap((status) => LOCALES.map((locale) => ({ status, locale }))))(
    "says a triage to $status by its own text, in $locale",
    async ({ status, locale }) => {
      const expected = text(locale, status);
      expect(typeof expected, `inbox_issue.activity.${status} in ${locale}`).toBe("string");
      expect(await said(status, locale)).toBe(expected);
    }
  );

  it.each(LOCALES)("says each status by a text of its own, none the generic one, in %s", (locale) => {
    const texts = [...STATUSES.map((status) => text(locale, status)), text(locale, "updated")];
    expect(new Set(texts).size).toBe(STATUSES.length + 1);
  });

  // the contract's verb is an open string (re-94 O1): one nerve does not write today is said generically
  it.each(LOCALES.flatMap((locale) => ["reopened", "1", "constructor"].map((verb) => ({ verb, locale }))))(
    "says a triage to $verb, a status it does not name, by the generic text, in $locale",
    async ({ verb, locale }) => {
      const expected = text(locale, "updated");
      expect(typeof expected, `inbox_issue.activity.updated in ${locale}`).toBe("string");
      expect(await said(verb, locale)).toBe(expected);
    }
  );
});

describe("IssueActivityItem, the field of a triage's row", () => {
  it.each([
    { field: "intake", drawn: true },
    // the old intake's field; nerve writes intake alone
    { field: "inbox", drawn: false },
  ])("draws a triage's row on field $field: $drawn", async ({ field, drawn }) => {
    binding.locale = "en";
    binding.activity = { field, verb: "accepted", created_at: "2026-11-20T00:00:00Z" };
    await act(async () =>
      root.render(<IssueActivityItem activityId={field} issueId="i-x" workspaceSlug="acme" ends={undefined} />)
    );
    expect(host.querySelector("[data-row]")?.textContent ?? null).toBe(drawn ? text("en", "accepted") : null);
  });
});
