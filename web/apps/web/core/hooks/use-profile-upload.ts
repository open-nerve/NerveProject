/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { UploadSession } from "@/store/asset/upload-session";
import type { ProfileUploadTarget } from "@/store/asset/upload-policy";
import { useContext, useLayoutEffect, useRef, useState } from "react";
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";

/** A view owns one target's upload, created after commit and released on close, target change or unmount. */
export function useProfileUpload(target: ProfileUploadTarget, enabled: boolean): UploadSession | undefined {
  const root = useContext(StoreContext);
  const identity = JSON.stringify(target);
  const active = enabled && root.issue.active;
  const held = useRef<{ root: RootStore; identity: string; upload: UploadSession } | undefined>(undefined);
  const [, redraw] = useState(0);
  useLayoutEffect(() => {
    const previous = held.current;
    if (active && previous?.root === root && previous.identity === identity) return;
    previous?.upload.dispose();
    held.current = undefined;
    if (active) {
      held.current = { root, identity, upload: root.asset.createUpload(target) };
      redraw((value) => value + 1);
    }
  }, [root, identity, active, target]);
  useLayoutEffect(
    () => () => {
      held.current?.upload.dispose();
      held.current = undefined;
    },
    []
  );
  return active && held.current?.root === root && held.current.identity === identity ? held.current.upload : undefined;
}
