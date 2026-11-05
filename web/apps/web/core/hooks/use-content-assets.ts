/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useContext, useLayoutEffect, useRef, useState } from "react";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import type { ContentHost, ContentUse, ContentView } from "@/store/asset/content-view";
import type { UploadSession } from "@/store/asset/upload-session";
/** Canonical owners decide whether a host is present; effects own the view, never render-time allocation. */
export function useContentView(host: ContentHost | undefined): ContentView | undefined {
  const root = useContext(StoreContext);
  const identity = JSON.stringify(host);
  const held = useRef<{ root: RootStore; identity: string | undefined; view: ContentView } | undefined>(undefined);
  const [, redraw] = useState(0);
  useLayoutEffect(() => {
    const previous = held.current;
    if (host && previous?.root === root && previous.identity === identity && previous.view.current) return;
    previous?.view.dispose();
    held.current = undefined;
    if (host && root.issue.active) {
      const view = root.asset.openContent(host);
      held.current = { root, identity, view };
      if (view.current) redraw((value) => value + 1);
    }
  }, [root, host, identity]);
  useLayoutEffect(
    () => () => {
      held.current?.view.dispose();
      held.current = undefined;
    },
    []
  );
  return held.current?.root === root && held.current.identity === identity && held.current.view.current
    ? held.current.view
    : undefined;
}
export function useContentUpload(view: ContentView | undefined, use: ContentUse): UploadSession | undefined {
  const root = useContext(StoreContext);
  const active = !!view?.canWrite(use);
  const held = useRef<{ root: RootStore; view: ContentView; use: ContentUse; upload: UploadSession } | undefined>(
    undefined
  );
  const [, redraw] = useState(0);
  useLayoutEffect(() => {
    const previous = held.current;
    if (active && previous?.root === root && previous.view === view && previous.use === use && previous.upload.current)
      return;
    previous?.upload.dispose();
    held.current = undefined;
    if (active && view) {
      held.current = { root, view, use, upload: root.asset.createContentUpload(view, use) };
      redraw((value) => value + 1);
    }
  }, [root, view, use, active]);
  useLayoutEffect(
    () => () => {
      held.current?.upload.dispose();
      held.current = undefined;
    },
    []
  );
  return active && held.current?.root === root && held.current.view === view && held.current.use === use
    ? held.current.upload
    : undefined;
}
