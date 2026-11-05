/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { EditorImages, EditorImageRead } from "@nerve/editor";
import { SessionChangedError } from "@/lib/auth/token-manager";
/** Inert HTML identities only; never create a browser image or retain a signed address. */
export function contentImageIDs(html: string): { ids: string[]; incomplete: boolean } {
  if (typeof DOMParser === "undefined") return { ids: [], incomplete: false };
  const document = new DOMParser().parseFromString(html, "text/html");
  const ids = new Set<string>();
  let incomplete = false;
  for (const node of document.querySelectorAll("image-component")) {
    const id = node.getAttribute("src") ?? "";
    if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(id)) ids.add(id);
    else incomplete = true;
  }
  return { ids: [...ids], incomplete };
}
/** Explicit input recovery shares the current view's reads, error budget and cancellation. */
export async function verifyEditorImages(images: EditorImages | undefined, html: string): Promise<void> {
  const { ids, incomplete } = contentImageIDs(html);
  if (incomplete) throw new Error("content_files.image.unfinished");
  if (!ids.length) return;
  if (!images || images.signal.aborted) throw new SessionChangedError();
  const reads: EditorImageRead[] = [];
  const stops: (() => void)[] = [];
  try {
    for (const id of ids) reads.push(images.read(id));
    await new Promise<void>((resolve, reject) => {
      const cancelled = () => reject(new SessionChangedError());
      images.signal.addEventListener("abort", cancelled, { once: true });
      stops.push(() => images.signal.removeEventListener("abort", cancelled));
      const check = () => {
        if (images.signal.aborted) return cancelled();
        const snapshots = reads.map((read) => read.getSnapshot());
        const failed = snapshots.find((snapshot) => snapshot.phase === "error");
        if (failed?.phase === "error") reject(new Error(failed.message));
        else if (snapshots.some((snapshot) => snapshot.phase === "unavailable")) cancelled();
        else if (snapshots.every((snapshot) => snapshot.phase === "ready")) resolve();
      };
      for (const read of reads) stops.push(read.subscribe(check));
      check();
    });
    if (images.signal.aborted) throw new SessionChangedError();
  } finally {
    for (const stop of stops) stop();
    for (const read of reads) read.dispose();
  }
}
