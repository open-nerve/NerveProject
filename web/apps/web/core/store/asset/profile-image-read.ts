/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, runInAction } from "mobx";
import type { RootStore } from "@/store/root.store";
import { ScopeLifetime } from "./scope-lifetime";
import type { ImageSource } from "./image-source";

type Projection = { refs: number; blocked?: string; armed?: string; retried: boolean };

/** Permission proof is per active scope, even when several workspaces share their directory's HTTP read. */
export class ImageReadEntry {
  readonly lifetime: ScopeLifetime;
  readonly projections = observable.map<string, Projection>(undefined, { deep: false });
  proof = false;
  attempted = false;
  requested = false;
  loading = false;
  error: unknown;
  version = 0;
  errorRefreshVersion = -1;
  constructor(
    root: RootStore,
    readonly source: ImageSource,
    readonly key: string,
    readonly group: ImageReadGroup,
    onEnd: () => void
  ) {
    makeObservable(this, {
      proof: observable,
      attempted: observable,
      requested: observable,
      loading: observable,
      error: observable.ref,
      version: observable,
    });
    this.lifetime = new ScopeLifetime(root, source.scope, () => source.allowed, onEnd);
  }
  add(projection: string): void {
    const previous = this.projections.get(projection);
    runInAction(() =>
      this.projections.set(projection, { retried: false, ...previous, refs: (previous?.refs ?? 0) + 1 })
    );
  }
  release(projection: string): void {
    const held = this.projections.get(projection);
    if (!held) return;
    runInAction(() => {
      if (held.refs === 1) this.projections.delete(projection);
      else this.projections.set(projection, { ...held, refs: held.refs - 1 });
    });
    if (!this.projections.size) this.lifetime.dispose();
  }
  change(projection: string, fields: Partial<Projection>): void {
    const held = this.projections.get(projection);
    if (held) runInAction(() => this.projections.set(projection, { ...held, ...fields }));
  }
}

/** Owns only the in-flight owner read. DTOs stay in their original reconciled stores. */
export class ImageReadGroup {
  readonly entries = new Map<string, ImageReadEntry>();
  private pending: Promise<void> | undefined;
  async read(entry: ImageReadEntry): Promise<void> {
    if (!entry.lifetime.current) return;
    if (this.pending) {
      try {
        await this.pending;
      } catch (error) {
        // A newly acquired scope was not part of the failed read and still needs its first attempt.
        if (entry.attempted) throw error;
      }
      // A scope acquired after dispatch was not authorized by that request.
      return this.read(entry);
    }
    if (entry.attempted && !entry.requested) return;
    const pending = this.fetch(entry);
    this.pending = pending;
    try {
      await pending;
    } finally {
      if (this.pending === pending) this.pending = undefined;
    }
  }
  private async fetch(entry: ImageReadEntry): Promise<void> {
    // Publish pending before any observable loading notification can reenter read().
    await Promise.resolve();
    const active = [...this.entries.values()].filter((item) => item.lifetime.current);
    if (!active.length) return;
    runInAction(() => {
      for (const item of active) {
        item.attempted = true;
        item.requested = false;
        item.loading = true;
        item.error = undefined;
        item.proof = false;
        for (const [projection] of item.projections) item.change(projection, { armed: undefined });
      }
    });
    try {
      if (!active.some((item) => item.lifetime.current)) return;
      const answer = await entry.source.read();
      if (answer === undefined || answer === null) throw new Error("Image owner read was superseded or unavailable");
      runInAction(() => {
        for (const item of active)
          if (item.lifetime.current) {
            item.proof = true;
            item.version++;
          }
      });
    } catch (error) {
      const current = active.filter((item) => item.lifetime.current);
      runInAction(() => {
        for (const item of current) item.error = error;
      });
      if (current.length) throw error;
    } finally {
      runInAction(() => {
        for (const item of active) item.loading = false;
      });
    }
    return;
  }
}
