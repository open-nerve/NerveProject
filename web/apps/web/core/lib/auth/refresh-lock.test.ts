/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SharedStorage, gate } from "./fake-browser";
import { LEASE_KEY, LOCK_NAME, leaseLock, webLock } from "./refresh-lock";

// The lease (M2 design 7.1) with fake timers: every wait is a timer the test advances, never a sleep.

function tab(storage: SharedStorage, tabId: string) {
  const view = storage.tab(tabId);
  return leaseLock({ storage: view, onStorage: (listener) => view.onStorage(listener), now: () => Date.now(), tabId });
}

/** Runs a task that records when it starts and ends and waits on a gate the test opens. */
function task(log: string[], name: string) {
  const g = gate<string>();
  const run = async () => {
    log.push(`${name} start ${Date.now()}`);
    const value = await g.promise;
    log.push(`${name} end ${Date.now()}`);
    return value;
  };
  return { run, finish: () => g.open(name) };
}

const lease = (storage: SharedStorage) => JSON.parse(storage.data.get(LEASE_KEY) ?? "null") as unknown;

describe("leaseLock", () => {
  beforeEach(() => {
    vi.useFakeTimers({ now: 0 });
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("takes a free lease once it reads it back, and deletes it after the task", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const a = task(log, "a");
    const done = tab(storage, "A").run(a.run);

    await vi.advanceTimersByTimeAsync(99);
    expect(log).toEqual([]);
    expect(LEASE_KEY).toBe("nerve.auth.refresh_lease");
    expect(lease(storage)).toEqual({ owner: "A", expires: 10_000 });
    await vi.advanceTimersByTimeAsync(1);
    expect(log).toEqual(["a start 100"]);

    a.finish();
    await expect(done).resolves.toBe("a");
    expect(storage.data.has(LEASE_KEY)).toBe(false);
  });

  it("makes another tab wait while it holds the lease, and wakes it on the release", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const [a, b] = [task(log, "a"), task(log, "b")];
    const first = tab(storage, "A").run(a.run);
    await vi.advanceTimersByTimeAsync(100);
    const second = tab(storage, "B").run(b.run);

    await vi.advanceTimersByTimeAsync(5_000);
    expect(log).toEqual(["a start 100"]);

    // The release is a storage event for B, which takes the lease at once instead of at its next poll.
    await vi.advanceTimersByTimeAsync(50);
    a.finish();
    await first;
    await vi.advanceTimersByTimeAsync(100);
    expect(log).toEqual(["a start 100", "a end 5150", "b start 5250"]);
    b.finish();
    await second;
  });

  it("finds a released lease by polling when no storage event comes", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const b = task(log, "b");
    // A tab outside the test holds the lease; it expires in 10 s but goes away sooner, silently.
    storage.data.set(LEASE_KEY, JSON.stringify({ owner: "A", expires: 10_000 }));
    const waiting = tab(storage, "B").run(b.run);

    await vi.advanceTimersByTimeAsync(1_250);
    storage.data.delete(LEASE_KEY);
    // The next poll, at 1400 ms, finds it free; B takes it and starts 100 ms later.
    await vi.advanceTimersByTimeAsync(249);
    expect(log).toEqual([]);
    await vi.advanceTimersByTimeAsync(1);
    expect(log).toEqual(["b start 1500"]);
    b.finish();
    await waiting;
  });

  it("takes over an expired lease", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    // A tab took the lease and was frozen: its lease is never released.
    storage.data.set(LEASE_KEY, JSON.stringify({ owner: "A", expires: 10_000 }));
    const b = task(log, "b");
    const waiting = tab(storage, "B").run(b.run);

    await vi.advanceTimersByTimeAsync(10_000 - 1);
    expect(log).toEqual([]);
    await vi.advanceTimersByTimeAsync(200 + 100);
    expect(log).toHaveLength(1);
    expect(lease(storage)).toMatchObject({ owner: "B" });
    b.finish();
    await waiting;
  });

  it("deletes only its own lease", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const [a, b] = [task(log, "a"), task(log, "b")];
    const first = tab(storage, "A").run(a.run);
    await vi.advanceTimersByTimeAsync(100);
    // A holds the lease longer than it lasts; B takes it over.
    const second = tab(storage, "B").run(b.run);
    await vi.advanceTimersByTimeAsync(10_000 + 300);
    expect(lease(storage)).toMatchObject({ owner: "B" });

    a.finish();
    await first;
    expect(lease(storage)).toMatchObject({ owner: "B" });
    b.finish();
    await second;
    expect(storage.data.has(LEASE_KEY)).toBe(false);
  });

  it("waits when another tab wrote its lease over this tab's before the read-back", async () => {
    const storage = new SharedStorage();
    const log: string[] = [];
    const a = task(log, "a");
    const waiting = tab(storage, "A").run(a.run);

    // Another tab saw the lease free at the same moment and wrote its own after A's.
    await vi.advanceTimersByTimeAsync(50);
    storage.write(LEASE_KEY, JSON.stringify({ owner: "B", expires: 50 + 10_000 }));
    await vi.advanceTimersByTimeAsync(3_000);
    expect(log).toEqual([]);
    expect(lease(storage)).toMatchObject({ owner: "B" });

    storage.write(LEASE_KEY, null);
    await vi.advanceTimersByTimeAsync(100);
    expect(log).toEqual(["a start 3150"]);
    a.finish();
    await waiting;
  });

  it("runs the tasks of one tab one after the other", async () => {
    const storage = new SharedStorage();
    const lock = tab(storage, "A");
    const log: string[] = [];
    const [a, b] = [task(log, "a"), task(log, "b")];
    const first = lock.run(a.run);
    const second = lock.run(b.run);

    await vi.advanceTimersByTimeAsync(1_000);
    expect(log).toEqual(["a start 100"]);
    a.finish();
    await first;
    await vi.advanceTimersByTimeAsync(100);
    expect(log).toEqual(["a start 100", "a end 1000", "b start 1100"]);
    b.finish();
    await second;
  });

  it("releases the lease when the task fails, and passes the failure on", async () => {
    const storage = new SharedStorage();
    const lock = tab(storage, "A");
    const failed = lock.run(async () => {
      throw new Error("refresh failed");
    });
    const caught = expect(failed).rejects.toThrow("refresh failed");
    await vi.advanceTimersByTimeAsync(100);
    await caught;
    expect(storage.data.has(LEASE_KEY)).toBe(false);

    const next = lock.run(async () => "next");
    await vi.advanceTimersByTimeAsync(100);
    await expect(next).resolves.toBe("next");
  });
});

describe("webLock", () => {
  it("runs the task under the lock nerve.auth.refresh and passes its result on", async () => {
    const names: string[] = [];
    const locks = {
      request: (async (name: string, granted: () => Promise<unknown>) => {
        names.push(name);
        return granted();
      }) as LockManager["request"],
    };

    await expect(webLock(locks).run(async () => 42)).resolves.toBe(42);
    expect(names).toEqual([LOCK_NAME]);
    expect(LOCK_NAME).toBe("nerve.auth.refresh");
  });
});
