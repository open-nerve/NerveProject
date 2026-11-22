/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The intake of the tests of the intake store and its readers (P5a spec 2.3), as nerve gives it. Each export comes
// with the first test that reads it (plan, "knip 与第一个读者").

import type { IntakeCounts } from "@nerve/api-client";

/**
 * The pending work items of a workspace's intakes as nerve counts them: one count for each project pending names, by
 * its id, in id order.
 */
export const countsOf = (pending: Record<string, number>): IntakeCounts => ({
  data: Object.entries(pending)
    .toSorted(([a], [b]) => a.localeCompare(b))
    .map(([projectId, count]) => ({ project_id: projectId, pending: count })),
});
