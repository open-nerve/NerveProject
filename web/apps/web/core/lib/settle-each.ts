/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

/**
 * The writes of a project nerve takes at once: its write gate holds two (server.write_gate.per_project) and answers one
 * that waited too long with 503 server_busy, which the web does not retry (P7b spec 2.9, R6). A batch of writes of a
 * project sends no more at once.
 */
export const WRITES_PER_PROJECT = 2;

/** Each id's outcome of a batch: those whose task succeeded, and those whose task failed with why, in the order given. */
type Outcomes = { confirmedIds: string[]; failures: { id: string; error: unknown }[] };

/**
 * Runs task for each of ids, at most atOnce of them at a time (P7b spec 2.9, R6): they start in the order given, each
 * next one once one before it has settled, whether it succeeded or failed, so a failure stops none of the others.
 * Answers once all have settled: each id's outcome, in the order given whatever order they settled in.
 */
export async function settleEach(
  ids: readonly string[],
  atOnce: number,
  task: (id: string) => Promise<unknown>
): Promise<Outcomes> {
  const settled: PromiseSettledResult<unknown>[] = [];
  let next = 0;
  // one of atOnce lanes: it runs the next task not yet started, then, once that one has settled, the one after
  const lane = async (): Promise<void> => {
    if (next >= ids.length) return;
    const index = next++;
    try {
      settled[index] = { status: "fulfilled", value: await task(ids[index]!) };
    } catch (reason) {
      settled[index] = { status: "rejected", reason };
    }
    return lane();
  };
  await Promise.all(Array.from({ length: Math.min(atOnce, ids.length) }, lane));
  return {
    confirmedIds: ids.filter((_, index) => settled[index]?.status === "fulfilled"),
    failures: ids.flatMap((id, index) => {
      const result = settled[index];
      return result?.status === "rejected" ? [{ id, error: result.reason as unknown }] : [];
    }),
  };
}
