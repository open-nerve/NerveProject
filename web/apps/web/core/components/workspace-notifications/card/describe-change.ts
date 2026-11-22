/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { NotificationChange, NotificationIssueRef } from "@nerve/api-client";

/**
 * How a card shows the values of a change it says: a calendar day nerve gives ("2026-10-17") and a priority ("low"),
 * in the page's language and the browser's zone (sentence.tsx); describeChange reads neither.
 */
export type ChangeFormat = { date: (day: string) => string; priority: (priority: string) => string };

/** What a card says of a change: its text's key, under notification.card, and the values the text names. */
type ChangeText = { key: string; params: Record<string, string> };

/** Says a change, given how to show its values. */
type Describer = (change: NotificationChange, format: ChangeFormat) => ChangeText;

const text = (key: string, params: Record<string, string> = {}): ChangeText => ({ key, params });

/**
 * Whether the value is one: neither none (null) nor "", which is how the work item's activity, and so nerve, gives a
 * removed label or assignee and a cleared date (P4 spec 2.6).
 */
const isValue = (value: string | null): value is string => value !== null && value !== "";

/** The work item a change names, as the caller reads it now: "WEB-12 Sign-in". */
const refOf = (issue: NotificationIssueRef) => `${issue.identifier}-${issue.sequence_id} ${issue.name}`;

/** A day set, as format shows it, or cleared. */
const dayOf =
  (set: string, removed: string): Describer =>
  ({ new_value: day }, format) =>
    isValue(day) ? text(set, { value: format.date(day) }) : text(removed);

/** A label or an assignee, by its name: added, with a new value; else removed, the one before. */
const memberOf =
  (added: string, removed: string): Describer =>
  ({ old_value: before, new_value: after }) =>
    isValue(after) ? text(added, { value: after }) : text(removed, { value: before ?? "" });

/**
 * A relation of a type, as the work item reads it, made (created) and undone (deleted): its other end, new_issue or
 * old_issue, named as the caller reads it now, or not named when he does not read it (nerve gives no work item then).
 */
function relationOf(
  type: string,
  keys: { added: string; addedHidden: string; removed: string; removedHidden: string }
): Record<string, Describer> {
  return {
    [`${type}/created`]: ({ new_issue: end }) => (end ? text(keys.added, { ref: refOf(end) }) : text(keys.addedHidden)),
    [`${type}/deleted`]: ({ old_issue: end }) =>
      end ? text(keys.removed, { ref: refOf(end) }) : text(keys.removedHidden),
  };
}

/** Each change a card says, by "field/verb" (P1b spec 2.3, P4 spec 2.6's table). */
const DESCRIBERS: Record<string, Describer> = {
  "issue/created": () => text("notification.card.created"),
  "name/updated": ({ new_value: name }) => text("notification.card.renamed", { value: name ?? "" }),
  "priority/updated": ({ new_value: priority }, format) =>
    text("notification.card.priority", { value: format.priority(priority ?? "") }),
  "state/updated": ({ new_value: state }) => text("notification.card.state", { value: state ?? "" }),
  "start_date/updated": dayOf("notification.card.start_date_set", "notification.card.start_date_removed"),
  "target_date/updated": dayOf("notification.card.due_date_set", "notification.card.due_date_removed"),
  "labels/updated": memberOf("notification.card.label_added", "notification.card.label_removed"),
  "assignees/updated": memberOf("notification.card.assignee_added", "notification.card.assignee_removed"),
  "assignees/created": memberOf("notification.card.assignee_added", "notification.card.assignee_removed"),
  // nerve nulls a parent's values; its ends are the work items the caller reads now, and a new parent he does not read
  // is no new_issue either, so one old_issue alone says only that it is no longer the parent (re-93 m14)
  "parent/updated": ({ old_issue: before, new_issue: after }) => {
    if (after) return text("notification.card.parent_set", { ref: refOf(after) });
    if (before) return text("notification.card.parent_removed", { ref: refOf(before) });
    return text("notification.card.parent_changed");
  },
  ...relationOf("relates_to", {
    added: "notification.card.relates_to_added",
    addedHidden: "notification.card.relates_to_added_hidden",
    removed: "notification.card.relates_to_removed",
    removedHidden: "notification.card.relates_to_removed_hidden",
  }),
  ...relationOf("duplicate", {
    added: "notification.card.duplicate_added",
    addedHidden: "notification.card.duplicate_added_hidden",
    removed: "notification.card.duplicate_removed",
    removedHidden: "notification.card.duplicate_removed_hidden",
  }),
  ...relationOf("blocking", {
    added: "notification.card.blocking_added",
    addedHidden: "notification.card.blocking_added_hidden",
    removed: "notification.card.blocking_removed",
    removedHidden: "notification.card.blocking_removed_hidden",
  }),
  ...relationOf("blocked_by", {
    added: "notification.card.blocked_by_added",
    addedHidden: "notification.card.blocked_by_added_hidden",
    removed: "notification.card.blocked_by_removed",
    removedHidden: "notification.card.blocked_by_removed_hidden",
  }),
  // the activity's values: manual_archive, archive (nerve's, automatic) or restore
  "archived_at/updated": ({ new_value: how }) =>
    text(how === "restore" ? "notification.card.restored" : "notification.card.archived"),
  "comment/created": () => text("notification.card.commented"),
  "comment/updated": () => text("notification.card.comment_edited"),
  "comment/deleted": () => text("notification.card.comment_deleted"),
  "link/created": ({ new_value: address }) => text("notification.card.link_added", { value: address ?? "" }),
  "link/updated": ({ new_value: address }) => text("notification.card.link_updated", { value: address ?? "" }),
  "link/deleted": ({ old_value: address }) => text("notification.card.link_removed", { value: address ?? "" }),
  // an attachment is said, never linked to (M5-M7-02)
  "attachment/created": () => text("notification.card.attachment_added"),
  "attachment/deleted": () => text("notification.card.attachment_removed"),
  "description/mentioned": () => text("notification.card.mentioned_description"),
  "comment/mentioned": () => text("notification.card.mentioned_comment"),
};

/**
 * @description what a notification card says of its change (P4 spec 2.6, R11): the text of its field and verb, an
 * addition told from a removal by the values (a value is neither null nor ""), a parent's or a relation's other end
 * named as the caller reads it now (new_issue, old_issue), never by old_value or new_value; any other change, one the
 * contract may add later, as the work item updated
 */
export const describeChange = (change: NotificationChange, format: ChangeFormat): ChangeText =>
  (DESCRIBERS[`${change.field}/${change.verb}`] ?? (() => text("notification.card.updated")))(change, format);
