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

/** Active views and a coalesced revalidation intent; the canonical hook owns the request. */
export class ImageReadGroup {
  readonly entries = new Map<string, ImageReadEntry>();
  refreshing: Promise<void> | undefined;
}
