/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import type { NotificationChange, NotificationIssueRef } from "@nerve/api-client";
import type { ChangeFormat } from "./describe-change";
import { describeChange } from "./describe-change";

// What a notification card says of its change (P4 spec 2.6, R11): one text for each field and verb, its key under
// notification.card, an addition told from a removal by the values. nerve gives the values as the work item's activity
// wrote them, so a removed label or assignee and a cleared date come as new_value "", and a value set where there was
// none as old_value "" (issue_events_test.go): "has a new value" is neither null nor "". A parent's and a relation's
// other end is new_issue or old_issue, as the caller reads it now, never old_value or new_value (nerve nulls those for a
// parent; the cells below give them stand-ins that must not show); with neither, the text names no work item (H4). Days
// and priorities are shown as format says (the card's, sentence.tsx); here they are marked, to be seen passed through.

/** What a card says of a change. */
type ChangeText = ReturnType<typeof describeChange>;

const format: ChangeFormat = { date: (day) => `day(${day})`, priority: (priority) => `priority(${priority})` };

/** A change of field by verb as nerve lists it: every key there, null where it has no such value, unless given. */
const changeOf = (field: string, verb: NotificationChange["verb"], fields: Partial<NotificationChange> = {}) => ({
  field,
  verb,
  old_value: null,
  new_value: null,
  old_id: null,
  new_id: null,
  old_issue: null,
  new_issue: null,
  comment_id: null,
  excerpt: null,
  ...fields,
});

const web3: NotificationIssueRef = { id: "i-3", sequence_id: 3, identifier: "WEB", name: "Login" };
const web9: NotificationIssueRef = { id: "i-9", sequence_id: 9, identifier: "WEB", name: "Mend" };
const text = (key: string, params: Record<string, string> = {}): ChangeText => ({
  key: `notification.card.${key}`,
  params,
});

/** Each row of the table (P4 spec 2.6), a cell each way it reads its change. */
const CELLS: [string, NotificationChange, ChangeText][] = [
  ["the work item's creation", changeOf("issue", "created"), text("created")],
  [
    "a new name",
    changeOf("name", "updated", { old_value: "Fix", new_value: "Mend" }),
    text("renamed", { value: "Mend" }),
  ],
  [
    "a priority set",
    changeOf("priority", "updated", { old_value: "high", new_value: "low" }),
    text("priority", { value: "priority(low)" }),
  ],
  [
    "a priority taken off",
    changeOf("priority", "updated", { old_value: "high", new_value: "none" }),
    text("priority", { value: "priority(none)" }),
  ],
  [
    "a state",
    changeOf("state", "updated", { old_value: "Backlog", new_value: "Done", old_id: "s-1", new_id: "s-2" }),
    text("state", { value: "Done" }),
  ],
  [
    "a start date changed",
    changeOf("start_date", "updated", { old_value: "2026-10-18", new_value: "2026-10-17" }),
    text("start_date_set", { value: "day(2026-10-17)" }),
  ],
  [
    "a start date set where there was none",
    changeOf("start_date", "updated", { old_value: "", new_value: "2026-10-18" }),
    text("start_date_set", { value: "day(2026-10-18)" }),
  ],
  [
    "a start date cleared",
    changeOf("start_date", "updated", { old_value: "2026-10-18", new_value: "" }),
    text("start_date_removed"),
  ],
  [
    "a due date set where there was none",
    changeOf("target_date", "updated", { old_value: "", new_value: "2026-10-19" }),
    text("due_date_set", { value: "day(2026-10-19)" }),
  ],
  [
    "a due date cleared",
    changeOf("target_date", "updated", { old_value: "2026-10-19", new_value: "" }),
    text("due_date_removed"),
  ],
  [
    "a label added",
    changeOf("labels", "updated", { old_value: "", new_value: "UI", new_id: "l-ui" }),
    text("label_added", { value: "UI" }),
  ],
  [
    "a label removed",
    changeOf("labels", "updated", { old_value: "Bug", new_value: "", old_id: "l-bug" }),
    text("label_removed", { value: "Bug" }),
  ],
  [
    "an assignee added",
    changeOf("assignees", "updated", { old_value: "", new_value: "Carol C.", new_id: "u-carol" }),
    text("assignee_added", { value: "Carol C." }),
  ],
  [
    "an assignee removed",
    changeOf("assignees", "updated", { old_value: "out", new_value: "", old_id: "u-out" }),
    text("assignee_removed", { value: "out" }),
  ],
  [
    "an assignee of the work item's creation",
    changeOf("assignees", "created", { new_value: "Carol C.", new_id: "u-carol" }),
    text("assignee_added", { value: "Carol C." }),
  ],
  [
    "a parent set, the caller reading it",
    changeOf("parent", "updated", { old_value: "WEB-3", new_value: "WEB-9", new_id: web9.id, new_issue: web9 }),
    text("parent_set", { ref: "WEB-9 Mend" }),
  ],
  [
    "a parent that is no longer the parent, the new one (if any) hidden from the caller (re-93 m14)",
    changeOf("parent", "updated", { old_value: "WEB-3", new_value: "WEB-9", old_id: web3.id, old_issue: web3 }),
    text("parent_removed", { ref: "WEB-3 Login" }),
  ],
  [
    "a parent changed, neither end read by the caller",
    changeOf("parent", "updated", { old_value: "WEB-3", new_value: "WEB-9" }),
    text("parent_changed"),
  ],
  // a relation, by its type as the work item reads it; its text's key starts with the type
  ...["relates_to", "duplicate", "blocking", "blocked_by"].flatMap(
    (type): [string, NotificationChange, ChangeText][] => [
      [
        `a ${type} relation added`,
        changeOf(type, "created", { new_value: "WEB-9", new_id: web9.id, new_issue: web9 }),
        text(`${type}_added`, { ref: "WEB-9 Mend" }),
      ],
      [
        `a ${type} relation added, the other end hidden from the caller`,
        changeOf(type, "created", { new_value: "WEB-9" }),
        text(`${type}_added_hidden`),
      ],
      [
        `a ${type} relation removed`,
        changeOf(type, "deleted", { old_value: "WEB-3", old_id: web3.id, old_issue: web3 }),
        text(`${type}_removed`, { ref: "WEB-3 Login" }),
      ],
      [
        `a ${type} relation removed, the other end hidden from the caller`,
        changeOf(type, "deleted", { old_value: "WEB-3" }),
        text(`${type}_removed_hidden`),
      ],
    ]
  ),
  ["an archive", changeOf("archived_at", "updated", { new_value: "manual_archive" }), text("archived")],
  ["an automatic archive", changeOf("archived_at", "updated", { new_value: "archive" }), text("archived")],
  ["a restore", changeOf("archived_at", "updated", { new_value: "restore" }), text("restored")],
  ["a comment", changeOf("comment", "created", { comment_id: "c-1", excerpt: "Ship it" }), text("commented")],
  ["a comment edited", changeOf("comment", "updated", { comment_id: "c-1", excerpt: "Ship" }), text("comment_edited")],
  ["a comment deleted", changeOf("comment", "deleted", { comment_id: "c-1" }), text("comment_deleted")],
  [
    "a link added",
    changeOf("link", "created", { new_value: "https://nerve.dev" }),
    text("link_added", { value: "https://nerve.dev" }),
  ],
  [
    "a link changed",
    changeOf("link", "updated", { old_value: "https://a.dev", new_value: "https://b.dev" }),
    text("link_updated", { value: "https://b.dev" }),
  ],
  [
    "a link removed",
    changeOf("link", "deleted", { old_value: "https://a.dev" }),
    text("link_removed", { value: "https://a.dev" }),
  ],
  ["an attachment added", changeOf("attachment", "created"), text("attachment_added")],
  ["an attachment removed", changeOf("attachment", "deleted"), text("attachment_removed")],
  ["a mention in the description", changeOf("description", "mentioned"), text("mentioned_description")],
  [
    "a mention in a comment",
    changeOf("comment", "mentioned", { comment_id: "c-1", excerpt: "@Bob look" }),
    text("mentioned_comment"),
  ],
  [
    "a field it has no text for, as one the contract may add later",
    changeOf("sort_order", "updated", { old_value: "65535", new_value: "0" }),
    text("updated"),
  ],
];

describe("describeChange", () => {
  it.each(CELLS)("says %s", (_, change, said) => {
    expect(describeChange(change, format)).toEqual(said);
  });

  it("names a parent's or a relation's other end only as the caller reads it now, never by old_value or new_value", () => {
    const ends = CELLS.filter(([, change]) =>
      ["parent", "relates_to", "duplicate", "blocking", "blocked_by"].includes(change.field)
    );
    expect(ends).toHaveLength(19);
    for (const [, change] of ends) {
      const values = Object.values(describeChange(change, format).params);
      expect(values).not.toContain(change.old_value);
      expect(values).not.toContain(change.new_value);
    }
  });
});

/** A key's name under notification.card. */
const leaf = (key: string) => key.replace(/^notification\.card\./, "");

/** A locale's notification.card texts. */
const cardTexts = (locale: string): Record<string, unknown> =>
  JSON.parse(
    readFileSync(
      new URL(`../../../../../../packages/i18n/src/locales/${locale}/notification.json`, import.meta.url),
      "utf8"
    )
  ).notification.card;

describe("the card's texts", () => {
  /** What the cells say. */
  const said = () => CELLS.map(([, change]) => describeChange(change, format));

  it.each(["en", "zh-CN"])("are each in %s, with the values the card gives it and no other", (locale) => {
    const texts = cardTexts(locale);
    for (const { key, params } of said()) {
      const found = texts[leaf(key)];
      expect(typeof found, `${locale}: ${key}`).toBe("string");
      const named = [...(found as string).matchAll(/\{(\w+)\}/g)].map((match) => match[1]).toSorted();
      expect(named, `${locale}: ${key}`).toEqual(Object.keys(params).toSorted());
    }
  });

  it("are all a change can be said with: every sentence of the table is in the cells above", () => {
    const sentences = Object.entries(cardTexts("en"))
      .filter(([key, value]) => typeof value === "string" && !["till", "unread"].includes(key))
      .map(([key]) => key)
      .toSorted();
    expect([...new Set(said().map(({ key }) => leaf(key)))].toSorted()).toEqual(sentences);
  });

  it("say of a parent no longer the parent that it no longer is, not that there is none (re-93 m14)", () => {
    expect(cardTexts("zh-CN").parent_removed).toContain("不再以 {ref} 为父工作项");
    expect(cardTexts("en").parent_removed).toContain("no longer a child of {ref}");
  });
});
