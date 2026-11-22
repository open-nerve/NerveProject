/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The intake of the tests of the intake store and its readers (P5a spec 2.3), as nerve gives it. Each export comes
// with the first test that reads it (plan, "knip 与第一个读者").

import type { IntakeCounts, IntakeIssue, IntakeIssuePage } from "@nerve/api-client";
import { issueOf } from "@/store/issue/fake-issues";

/**
 * The pending work items of a workspace's intakes as nerve counts them: one count for each project pending names, by
 * its id, in id order.
 */
export const countsOf = (pending: Record<string, number>): IntakeCounts => ({
  data: Object.entries(pending)
    .toSorted(([a], [b]) => a.localeCompare(b))
    .map(([projectId, count]) => ({ project_id: projectId, pending: count })),
});

/**
 * A work item of p-web's intake in acme (id-acme) as nerve lists it, its id and its name named by name: pending,
 * unless fields say so.
 */
export const intakeIssueOf = (name: string, fields: Partial<IntakeIssue> = {}): IntakeIssue => ({
  ...issueOf(`i-${name}`, "p-web", "id-acme"),
  name,
  intake: { status: "pending", snoozed_till: null, duplicate_to_id: null },
  ...fields,
});

/**
 * A page of a project's intake as nerve lists it: the last unless a cursor follows it; its total, the number its
 * filters select, the number of its rows unless given.
 */
export const pageOf = (
  data: IntakeIssue[],
  nextCursor: string | null = null,
  total = data.length
): IntakeIssuePage => ({
  total,
  data,
  next_cursor: nextCursor,
});
