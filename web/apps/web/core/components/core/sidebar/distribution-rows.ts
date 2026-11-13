/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { AssigneeDistribution, Label, LabelDistribution, MemberUser } from "@nerve/api-client";

// The rows of a cycle's or a module's work items by assignee and by label, named for the page (P5a spec 2.4, P6a spec
// 2.5). nerve's rows carry ids alone (P4 R3): the project's members and labels give the names. One no longer the
// project's (a member who left, a deleted label: a frozen progress keeps them) is named by a fallback text; the work
// items without any are the row with a null id, named so too.

/** The page's text of a key (useTranslation's t). */
type Translate = (key: string) => string;

/** A row of the work items of an assignee, or of those without one, named for the page. */
export type AssigneeRow = AssigneeDistribution & {
  /** The member's display name; the fallback text for one no longer the project's member, or for none. */
  name: string;
  /** The account whose avatar the row shows: the project's member's; null for one no longer its member, or none. */
  avatarOf: string | null;
};

/** A row of the work items of a label, or of those without one, named for the page. */
export type LabelRow = LabelDistribution & {
  /** The label's name; the fallback text for a label no longer the project's, or for none. */
  name: string;
  /** The label's color; null for a label without one, for one no longer the project's, or for none. */
  color: string | null;
};

/** Where a row goes: those the project names, then those named by the fallback text, then the row of none. */
const NAMED = 0;
const FALLBACK = 1;
const NONE = 2;
type Placed<R> = { rank: number; row: R };

/** The rows by their place, those the project names by name (localeCompare); rows alike keep nerve's order. */
const inOrder = <R extends { name: string }>(rows: Placed<R>[]): R[] =>
  rows.toSorted((a, b) => a.rank - b.rank || a.row.name.localeCompare(b.row.name)).map(({ row }) => row);

/**
 * The work items by assignee as the page shows them, memberOf giving a project's member's profile by his account id
 * (undefined for one not its member): by name, then those no longer members, the work items without an assignee last.
 */
export function assigneeRows(
  distribution: AssigneeDistribution[],
  memberOf: (accountId: string) => MemberUser | undefined,
  t: Translate
): AssigneeRow[] {
  return inOrder(
    distribution.map((row): Placed<AssigneeRow> => {
      if (row.assignee_id === null) return { rank: NONE, row: { ...row, name: t("no_assignee"), avatarOf: null } };
      const member = memberOf(row.assignee_id);
      return member
        ? { rank: NAMED, row: { ...row, name: member.display_name, avatarOf: row.assignee_id } }
        : { rank: FALLBACK, row: { ...row, name: t("distribution.former_member"), avatarOf: null } };
    })
  );
}

/**
 * The work items by label as the page shows them, of the project's labels: by name, then the labels no longer the
 * project's, the work items without a label last.
 */
export function labelRows(distribution: LabelDistribution[], labels: Label[], t: Translate): LabelRow[] {
  const byId = new Map(labels.map((label) => [label.id, label]));
  return inOrder(
    distribution.map((row): Placed<LabelRow> => {
      if (row.label_id === null) return { rank: NONE, row: { ...row, name: t("distribution.no_labels"), color: null } };
      const label = byId.get(row.label_id);
      return label
        ? { rank: NAMED, row: { ...row, name: label.name, color: label.color || null } }
        : { rank: FALLBACK, row: { ...row, name: t("distribution.deleted_label"), color: null } };
    })
  );
}
