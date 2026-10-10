/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { makeObservable, observable, runInAction } from "mobx";
// lib
import { SessionChangedError } from "@/lib/auth/token-manager";

/**
 * A change nerve confirmed, as it makes a value: an idempotent function of the value (made on a value that has it
 * already, it changes nothing), so it can be made again on a fetch's answer, which may have it or not.
 */
export type Change<V> = (value: V) => V;

/** The answers written so far, of every Reconciled value: each fetch's answer written takes the next number. */
let answersWritten = 0;

/**
 * One value a store shows of what nerve holds (a list, a map, a document), reconciled between its fetches and the
 * changes nerve confirmed (v0 design 7.7): only the newest fetch writes, and the changes nerve confirmed while it was
 * out are made again on its answer, in the order nerve confirmed them, as the answer may have been read before them.
 * A change to a value not fetched yet changes nothing: the fetch brings the whole value. Nothing here queues: a store
 * sends its changes one at a time (one-at-a-time.ts), and confirms each once nerve has answered it.
 */
export class Reconciled<V> {
  /** The value, once fetched. */
  value: V | undefined = undefined;
  /**
   * When nerve's answer the value shows was written, as the order of the answers written (0 until fetched): of two
   * values holding the same thing (a project in its list and in its own read), the greater was answered last.
   */
  answeredAt = 0;
  /** The sequence number of the newest fetch, the one fetch that may write. */
  private newestFetch = 0;
  /**
   * The changes nerve confirmed while the newest fetch is out, in the order confirmed; undefined once it has ended (an
   * older fetch still out writes nothing).
   */
  private changesDuringFetch: Change<V>[] | undefined = undefined;

  constructor() {
    makeObservable(this, { value: observable.ref, answeredAt: observable });
  }

  /**
   * Fetches the value with read, and shows it with the changes nerve confirmed meanwhile; gives what it shows. A fetch
   * that a newer one overtook gives undefined whatever its answer, a failure too, and writes nothing: the newer fetch
   * decides what the store shows. A change of session while it is out is no failure either: the new session's stores
   * fetch their own (store-context.tsx), and this one gives undefined. Any other failure of the newest fetch fails
   * it, and the value stays as it was.
   */
  async fetch(read: () => Promise<V>): Promise<V | undefined> {
    const sequence = ++this.newestFetch;
    const changes: Change<V>[] = [];
    this.changesDuringFetch = changes;
    try {
      const fetched = await read();
      if (sequence !== this.newestFetch) return undefined;
      const value = changes.reduce<V>((made, change) => change(made), fetched);
      runInAction(() => {
        this.value = value;
        this.answeredAt = ++answersWritten;
      });
      return value;
    } catch (error) {
      if (sequence !== this.newestFetch || error instanceof SessionChangedError) return undefined;
      throw error;
    } finally {
      if (sequence === this.newestFetch) this.changesDuringFetch = undefined;
    }
  }

  /** A change nerve confirmed: it is made on the value, once fetched, and on the answer of the fetch that is out. */
  confirm(change: Change<V>): void {
    runInAction(() => {
      if (this.value !== undefined) this.value = change(this.value);
    });
    this.changesDuringFetch?.push(change);
  }
}

/** A Reconciled value for each key (for one, a workspace's id): each key has its own newest fetch. */
export class ReconciledByKey<V> {
  private readonly entries = observable.map<string, Reconciled<V>>({}, { deep: false });

  /** The key's value, once fetched; nothing for no key. */
  get(key: string | undefined): V | undefined {
    return key === undefined ? undefined : this.entries.get(key)?.value;
  }

  /** When the key's value was answered (Reconciled.answeredAt); 0 for a key not fetched. */
  answeredAt(key: string): number {
    return this.entries.get(key)?.answeredAt ?? 0;
  }

  /** The values of the keys fetched, whatever their keys (for one, to find an item in any of them). */
  values(): V[] {
    return [...this.entries.values()].flatMap((entry) => (entry.value === undefined ? [] : [entry.value]));
  }

  /** Fetches the key's value with read (Reconciled.fetch). */
  fetch(key: string, read: () => Promise<V>): Promise<V | undefined> {
    return this.entry(key).fetch(read);
  }

  /** A change nerve confirmed to the key's value (Reconciled.confirm); for no key, there is nothing it could change. */
  confirm(key: string | undefined, change: Change<V>): void {
    if (key !== undefined) this.entries.get(key)?.confirm(change);
  }

  private entry(key: string): Reconciled<V> {
    const held = this.entries.get(key);
    if (held) return held;
    const entry = new Reconciled<V>();
    runInAction(() => {
      this.entries.set(key, entry);
    });
    return entry;
  }
}

/** An item of a list nerve gives, which its id names. The changes below make such a list. */
type Listed = { id: string };

/** The items nerve made, before the list, but for those it lists already. */
export const prepended =
  <T extends Listed>(items: T[]): Change<T[]> =>
  (list) => [...items.filter((item) => !list.some((listed) => listed.id === item.id)), ...list];

/** The item as nerve gave it, in its place where the list has it. */
export const replaced =
  <T extends Listed>(item: T): Change<T[]> =>
  (list) =>
    list.map((listed) => (listed.id === item.id ? item : listed));

/** The item as nerve gave it, in its place where the list has it, else last. */
export const upserted =
  <T extends Listed>(item: T): Change<T[]> =>
  (list) =>
    list.some((listed) => listed.id === item.id) ? replaced(item)(list) : [...list, item];

/** The list without the item id names. */
export const dropped =
  <T extends Listed>(id: string): Change<T[]> =>
  (list) =>
    list.filter((listed) => listed.id !== id);
