/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { answerTiming, priorAnswerMembership, pageEvidence } from "./paged-groups.evidence";
import {
  applyGroupChange,
  type ConfirmedIssueChange,
  type GroupChangeResult,
  type ListContext,
  type PageAnswer,
  type PagedGroups,
} from "./paged-groups";

/** Only the untouched answer establishes the replay phase and old membership. */
export function replayGroups(
  value: PagedGroups,
  change: ConfirmedIssueChange,
  answer: PageAnswer,
  context: ListContext
): GroupChangeResult {
  const evidence = pageEvidence(answer);
  const timing = answerTiming(evidence, change, context);
  if (timing.phase === "newer") return { value, verifyIds: [] };
  const result = applyGroupChange(value, change, context, {
    answer,
    phase: timing.phase,
    old: (ref) => priorAnswerMembership(evidence, change, context, ref),
  });
  if (timing.inferred && change.after) result.verifyIds = [change.id];
  return result;
}
