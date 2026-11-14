/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Cycle, Module } from "@nerve/api-client";
import type { ModuleSetChange } from "@nerve/types";

// A project's planning directory as a work item reads it (P7a spec 2.3, R4): its options, what the work item may join,
// apart from its values, what it is in. A value shows by its name as the planning store has it (getCycleById,
// getModuleById, which know the ended cycles and the archived cycles and modules it has read, as no option does), its
// project's lists read in one order for one it cannot name (nextPlanningList); the options are read as a dropdown opens
// (use-planning-options.ts).

/** A project's two lists of a domain: its list of what is not archived, and its archive. */
export type PlanningList = "list" | "archive";

/**
 * Which of a project's two lists of a domain to read next for a value of it the planning store cannot name (P7a spec
 * 2.3; P7b spec 2.2, re-74 F2): its list of what is not archived first, then its archive; neither once both are read,
 * nor while every value is named. The order alone, which every reader of names shares: what counts as read is the
 * caller's. usePlanningNames counts a list the store holds, for it reads as it renders and must not read again for a
 * value deleted meanwhile; a list's directory counts its own reads, so that it reads a list the store holds again for
 * a value created elsewhere since (issue-list-catalog.ts).
 */
export function nextPlanningList(
  unnamed: boolean,
  read: Readonly<Record<PlanningList, boolean>>
): PlanningList | undefined {
  if (!unnamed) return undefined;
  if (!read.list) return "list";
  return read.archive ? undefined : "archive";
}

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

/**
 * The set of modules a change makes of the set held: its removals taken out, its additions not in it put after. The
 * write unit makes a change's request of it in the work item's turn (issue-write-fields.ts), and a form its own value.
 */
export function modulesAfter(held: readonly string[], change: ModuleSetChange): string[] {
  const removed = new Set(change.remove);
  const modules = held.filter((module) => !removed.has(module));
  for (const module of change.add ?? []) if (!modules.includes(module)) modules.push(module);
  return modules;
}

/**
 * The change that makes the set after of the set before, modulesAfter's inverse: the modules of after not in before
 * added, those of before not in after taken out, none in both; none when both hold the same modules. A work item's
 * edit sends its modules so, from the form's start to the set it holds, applied in the write's turn to the set the
 * work item then holds (P7a spec 2.6, R3).
 */
export function moduleSetChange(before: readonly string[], after: readonly string[]): ModuleSetChange | undefined {
  const add = after.filter((module) => !before.includes(module));
  const remove = before.filter((module) => !after.includes(module));
  return add.length > 0 || remove.length > 0 ? { add, remove } : undefined;
}
