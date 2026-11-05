/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { EditorImageRead, EditorImageSnapshot } from "@nerve/editor";
import type { ContentReadLease, ContentSnapshot } from "@/store/asset/content-reads";
import type { ContentHost } from "@/store/asset/content-view";
import { matchesContent } from "@/store/asset/content-target";
export function isContentImage(snapshot: ContentSnapshot, host: ContentHost): boolean {
  return (
    snapshot.status === "ready" &&
    matchesContent(snapshot.asset, host, "description") &&
    ["image/png", "image/jpeg", "image/gif", "image/webp"].includes(snapshot.asset.content_type)
  );
}
/** No timer or error budget here: the shared content resource owns both. */
export class ImageRead implements EditorImageRead {
  private source?: ContentSnapshot;
  private snapshot: EditorImageSnapshot = { phase: "loading" };
  private ended = false;
  constructor(
    private readonly lease: ContentReadLease,
    private readonly host: ContentHost,
    private readonly message: (error: unknown) => string,
    private readonly forget: () => void
  ) {}
  getSnapshot = (): EditorImageSnapshot => {
    if (this.ended) return this.snapshot;
    const source = this.lease.getSnapshot();
    if (source !== this.source) {
      this.source = source;
      this.snapshot =
        source.status === "ready"
          ? isContentImage(source, this.host)
            ? { phase: "ready", url: source.content.url, expiresAt: source.content.expires_at }
            : { phase: "error", message: this.message(undefined) }
          : source.error
            ? { phase: "error", message: this.message(source.error) }
            : { phase: source.status };
    }
    return this.snapshot;
  };
  subscribe = (changed: () => void) => this.lease.subscribe(changed);
  loaded = () => {
    if (!this.ended && this.source?.status === "ready") this.lease.loaded(this.source.stamp);
  };
  failed = () => {
    if (!this.ended && this.source?.status === "ready") this.lease.failed(this.source.stamp);
  };
  retry = () => {
    if (!this.ended) void this.lease.retry().catch(() => undefined);
  };
  dispose = () => {
    if (this.ended) return;
    this.ended = true;
    this.source = undefined;
    this.snapshot = { phase: "unavailable" };
    this.lease.dispose();
    this.forget();
  };
}
