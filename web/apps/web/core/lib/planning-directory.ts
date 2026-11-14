/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Cycle, Module } from "@nerve/api-client";

// A project's planning directory as a work item reads it (P7a spec 2.3, R4): its options, what the work item may join,
// apart from its values, what it is in. A value shows by its name as the planning store has it (getCycleById,
// getModuleById, which know the ended cycles and the archived cycles and modules it has read, as no option does); the
// options are read as a dropdown opens (use-planning-options.ts).

/**
 * A work item's cycle options: No cycle (null) first, which takes it out of its cycle; then those of the project's
 * cycles that are not archived (cycleIds, the store's list, in nerve's order) it may join, as the store has them: not
 * archived since, not ended (nerve refuses both: planning.archived, planning.cycle_completed). Undefined until the list
 * is read.
 */
export function cycleOptions(
  cycleIds: readonly string[] | undefined,
  cycleOf: (cycleId: string) => Cycle | undefined
): (Cycle | null)[] | undefined {
  if (!cycleIds) return undefined;
  const joinable = cycleIds
    .map((cycleId) => cycleOf(cycleId))
    .filter((cycle): cycle is Cycle => !!cycle && cycle.archived_at === null && cycle.status !== "completed");
  return [null, ...joinable];
}

/**
 * A work item's module options: those of the project's modules that are not archived (moduleIds, the store's list, in
 * nerve's order) it may join, whatever their status, as the store has them: not archived since (nerve refuses it:
 * planning.archived). Those it is in stay where nerve's order has them (R13). Undefined until the list is read.
 */
export function moduleOptions(
  moduleIds: readonly string[] | undefined,
  moduleOf: (moduleId: string) => Module | undefined
): Module[] | undefined {
  if (!moduleIds) return undefined;
  return moduleIds
    .map((moduleId) => moduleOf(moduleId))
    .filter((item): item is Module => !!item && item.archived_at === null);
}
