/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

/**
 * A queue that runs the tasks given to it one at a time, in the order given: each starts once the one before it
 * has settled, whether it succeeded or failed. A store sends its changes of one resource through one, so nerve
 * applies the changes it answers in the order they were made, and the last change answered is what nerve holds
 * (v0 design 7.7). A change that failed without an answer (the connection broke after the request reached nerve)
 * may still be applied after the next one. A task that never settles holds the queue: nerve answers every request
 * it receives within its server.request_timeout.
 */
export function oneAtATime(): <T>(task: () => Promise<T>) => Promise<T> {
  let last: Promise<unknown> = Promise.resolve();
  return <T>(task: () => Promise<T>): Promise<T> => {
    const run = last.then(task);
    last = run.catch(() => undefined);
    return run;
  };
}
