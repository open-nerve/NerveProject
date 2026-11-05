/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useState, useSyncExternalStore } from "react";
import type { EditorImageRead, EditorImages, EditorImageSnapshot } from "@/types";
import { isImageID } from "../input";
const missing: EditorImageSnapshot = { phase: "unavailable" };
const empty: EditorImageRead = {
  getSnapshot: () => missing,
  subscribe: () => () => {},
  loaded: () => {},
  failed: () => {},
  retry: () => {},
  dispose: () => {},
};
function useImageAvailable(images: EditorImages | undefined) {
  const subscribe = useCallback(
    (changed: () => void) => {
      images?.signal.addEventListener("abort", changed);
      return () => images?.signal.removeEventListener("abort", changed);
    },
    [images]
  );
  return useSyncExternalStore(
    subscribe,
    () => !!images && !images.signal.aborted,
    () => false
  );
}
export function useImageRead(images: EditorImages | undefined, src: string | null) {
  const available = useImageAvailable(images);
  const [held, setHeld] = useState<{ images: EditorImages; src: string; read: EditorImageRead }>();
  useLayoutEffect(() => {
    if (!images || images.signal.aborted || !isImageID(src)) return;
    const read = images.read(src);
    setHeld({ images, src, read });
    return read.dispose;
  }, [images, src, available]);
  const read = available && held && held.images === images && held.src === src ? held.read : empty;
  const snapshot = useSyncExternalStore(read.subscribe, read.getSnapshot, empty.getSnapshot);
  return { read, snapshot, available };
}
