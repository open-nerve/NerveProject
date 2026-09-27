/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// Test doubles of the browser for the auth tests: one localStorage shared by several tabs, which tells
// the other tabs about every change with a storage event. The event comes in a microtask of the writing
// task, sooner than in a browser, where it reaches the other tabs in a later task.

type StorageListener = (key: string | null, newValue: string | null) => void;

/** The localStorage of one browser: every tab's view writes the same data. */
export class SharedStorage {
  readonly data = new Map<string, string>();
  private readonly listeners = new Map<string, Set<StorageListener>>();

  /** The view of the tab `tab`: its writes reach the other tabs' listeners, never its own. */
  tab(tab: string) {
    return {
      getItem: (key: string) => this.data.get(key) ?? null,
      setItem: (key: string, value: string) => {
        this.data.set(key, String(value));
        this.notify(tab, key, String(value));
      },
      removeItem: (key: string) => {
        if (!this.data.delete(key)) return;
        this.notify(tab, key, null);
      },
      /** Subscribes this tab to the storage events of the other tabs; returns the unsubscribe. */
      onStorage: (listener: StorageListener) => {
        const set = this.listeners.get(tab) ?? new Set<StorageListener>();
        this.listeners.set(tab, set);
        set.add(listener);
        return () => {
          set.delete(listener);
        };
      },
    };
  }

  /** A write by a tab outside the tests' tabs, e.g. a test playing another tab by hand. */
  write(key: string, value: string | null): void {
    if (value === null) this.data.delete(key);
    else this.data.set(key, value);
    this.notify("elsewhere", key, value);
  }

  private notify(writer: string, key: string, value: string | null): void {
    for (const [tab, set] of this.listeners) {
      if (tab === writer) continue;
      // Like a browser, never inside the write; unlike one, before the writing task ends (a microtask).
      queueMicrotask(() => {
        for (const listener of set) listener(key, value);
      });
    }
  }
}

/** A promise the test resolves or rejects by hand. */
export function gate<T = void>() {
  let open!: (value: T) => void;
  let fail!: (reason: unknown) => void;
  const promise = new Promise<T>((resolve, reject) => {
    open = resolve;
    fail = reject;
  });
  return { promise, open, fail };
}
