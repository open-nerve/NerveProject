/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A batch of tasks, one for each id, at most a number of them at a time (P7b spec 2.9, R6): they start in the order
// given, each next one once one before it has settled; a failure stops none of the others; the answer, once all have
// settled, is each id's outcome in the order given, whatever order they settled in.
import { expect, it } from "vitest";
import { settleEach } from "./settle-each";

/** A task the test settles itself: each id's, as it was started. */
function heldTasks() {
  const started: string[] = [];
  const settles = new Map<string, { resolve: () => void; reject: (error: unknown) => void }>();
  let inFlight = 0;
  let most = 0;
  const task = (id: string) =>
    new Promise<void>((resolve, reject) => {
      started.push(id);
      most = Math.max(most, ++inFlight);
      const done = () => inFlight--;
      settles.set(id, {
        resolve: () => (done(), resolve()),
        reject: (error) => (done(), reject(error)),
      });
    });
  return { task, started, settles, most: () => most };
}
const tick = () => new Promise((resolve) => setTimeout(resolve, 0));

it("runs five tasks two at a time, in the order given, and answers each one's outcome in that order", async () => {
  const held = heldTasks();
  const refused = new Error("refused");
  const answer = settleEach(["a", "b", "c", "d", "e"], 2, held.task);
  await tick();
  expect(held.started).toEqual(["a", "b"]);
  // b fails first: c starts in its place, and a failure stops none of the others
  held.settles.get("b")!.reject(refused);
  await tick();
  expect(held.started).toEqual(["a", "b", "c"]);
  held.settles.get("c")!.resolve();
  await tick();
  expect(held.started).toEqual(["a", "b", "c", "d"]);
  held.settles.get("d")!.resolve();
  await tick();
  expect(held.started).toEqual(["a", "b", "c", "d", "e"]);
  held.settles.get("e")!.reject(refused);
  await tick();
  // a, the first started, settles last
  held.settles.get("a")!.resolve();
  expect(await answer).toEqual({
    confirmedIds: ["a", "c", "d"],
    failures: [
      { id: "b", error: refused },
      { id: "e", error: refused },
    ],
  });
  expect(held.most()).toBe(2);
});

it("answers a task that throws before it starts waiting as failed, and starts the next one", async () => {
  const thrown = new Error("thrown");
  const answer = settleEach(["a", "b"], 1, (id) => {
    if (id === "a") throw thrown;
    return Promise.resolve();
  });
  expect(await answer).toEqual({ confirmedIds: ["b"], failures: [{ id: "a", error: thrown }] });
});

it("answers no ids for none", async () => {
  expect(await settleEach([], 2, () => Promise.resolve())).toEqual({ confirmedIds: [], failures: [] });
});
