/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observable, makeObservable, runInAction } from "mobx";
import type { RootStore } from "@/store/root.store";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { ImageSource, type ProfileImageSource } from "./image-source";
import { ImageReadEntry, ImageReadGroup } from "./profile-image-read";
export type { ProfileImageSource } from "./image-source";

export class ProfileImageLease {
  private disposed = false;
  constructor(
    private readonly source: ImageSource,
    private readonly entry: ImageReadEntry
  ) {
    makeObservable<ProfileImageLease, "disposed">(this, { disposed: observable });
    entry.add(source.projection);
  }
  get key(): string {
    return this.entry.key;
  }
  get current(): boolean {
    return !this.disposed && this.entry.lifetime.current;
  }
  get stamp(): string {
    return JSON.stringify([this.entry.version, this.source.projection, this.source.fact]);
  }
  get expiry(): number | undefined {
    const value = this.source.fact?.expiresAt;
    const time = value ? Date.parse(value) : NaN;
    return Number.isFinite(time) ? time : undefined;
  }
  get url(): string | undefined {
    return this.current &&
      this.entry.proof &&
      this.state?.blocked !== this.stamp &&
      this.expiry !== undefined &&
      this.expiry > Date.now()
      ? this.source.fact?.url || undefined
      : undefined;
  }
  get loading(): boolean {
    return this.current && this.entry.loading;
  }
  get error(): unknown {
    if (!this.current) return undefined;
    if (this.entry.error !== undefined) return this.entry.error;
    if (this.state?.blocked === this.stamp) return new Error("Image could not be loaded");
    if (this.entry.proof && this.source.fact?.url && (this.expiry === undefined || this.expiry <= Date.now()))
      return new Error("Image authorization has expired");
    return undefined;
  }
  get canRead(): boolean {
    return this.current && !!this.source.fact?.url;
  }
  private get state() {
    return this.entry.projections.get(this.source.projection);
  }
  async read(): Promise<void> {
    if (!this.canRead) return;
    try {
      await this.entry.group.read(this.entry);
    } catch (error) {
      if (this.current) throw error;
    }
  }
  requestRetry(): void {
    if (!this.canRead) return;
    this.entry.change(this.source.projection, { blocked: undefined, retried: false, armed: undefined });
    runInAction(() => {
      this.entry.requested = true;
    });
  }
  arm(): void {
    if (this.url) this.entry.change(this.source.projection, { armed: this.stamp });
  }
  expire(stamp: string): boolean {
    if (
      !this.canRead ||
      stamp !== this.stamp ||
      this.state?.armed !== stamp ||
      (this.expiry !== undefined && this.expiry > Date.now())
    )
      return false;
    this.entry.change(this.source.projection, { armed: undefined });
    runInAction(() => {
      this.entry.requested = true;
    });
    return true;
  }
  fail(stamp: string): boolean {
    if (!this.current || stamp !== this.stamp || !this.entry.proof || this.state?.blocked === stamp) return false;
    const retry = !this.state?.retried && this.entry.errorRefreshVersion !== this.entry.version;
    this.entry.change(this.source.projection, { blocked: stamp, retried: true, armed: undefined });
    if (retry)
      runInAction(() => {
        this.entry.requested = true;
        this.entry.errorRefreshVersion = this.entry.version;
      });
    return retry;
  }
  loaded(stamp: string): void {
    if (this.url && stamp === this.stamp) this.entry.change(this.source.projection, { retried: false });
  }
  dispose(): void {
    if (this.disposed) return;
    runInAction(() => {
      this.disposed = true;
    });
    this.entry.release(this.source.projection);
  }
}

export class ProfileImages {
  private readonly groups = new Map<string, ImageReadGroup>();
  private generation = 0;
  private disposed = false;
  constructor(private readonly root: RootStore) {}
  acquire(value: ProfileImageSource): ProfileImageLease {
    this.root.issue.assertActive();
    if (this.disposed) throw new SessionChangedError();
    const source = new ImageSource(this.root, value);
    let group = this.groups.get(source.group);
    if (!group) {
      group = new ImageReadGroup();
      this.groups.set(source.group, group);
    }
    let entry = group.entries.get(source.entry);
    if (!entry) {
      const owner = group;
      const created = new ImageReadEntry(this.root, source, `${source.entry}:${++this.generation}`, owner, () => {
        if (owner.entries.get(source.entry) === created) owner.entries.delete(source.entry);
        if (!owner.entries.size && this.groups.get(source.group) === owner) this.groups.delete(source.group);
      });
      owner.entries.set(source.entry, created);
      entry = created;
    }
    return new ProfileImageLease(source, entry);
  }
  dispose(): void {
    if (this.disposed) return;
    this.disposed = true;
    for (const group of this.groups.values()) for (const entry of group.entries.values()) entry.lifetime.dispose();
    this.groups.clear();
  }
}
