/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The RootStore of the tests of one store: the sibling stores that store reads, and no other. The stores take the
// whole RootStore; a store that reads a sibling the test did not give fails the test, instead of reading undefined.

import type { RootStore } from "@/store/root.store";

export function fakeRoot(siblings: Partial<RootStore>): RootStore {
  const root = new Proxy(siblings, {
    get(target, name, receiver) {
      if (!Reflect.has(target, name)) throw new Error(`the store under test read root.${String(name)}, not given`);
      return Reflect.get(target, name, receiver);
    },
  });
  // The one cast of the tests' roots: a Partial is not a RootStore, and the proxy answers for the rest by failing.
  return root as RootStore;
}
