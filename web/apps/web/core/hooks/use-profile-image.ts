/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useContext, useEffect, useLayoutEffect, useRef, useState } from "react";
import { StoreContext } from "@/lib/store-context";
import { useSessionSWR } from "@/lib/use-session-swr";
import { ImageSource } from "@/store/asset/image-source";
import type { ProfileImageLease, ProfileImageSource } from "@/store/asset/profile-images";
import type { RootStore } from "@/store/root.store";

export type ProfileImageView = {
  url: string | undefined;
  loading: boolean;
  error: unknown;
  retry(): Promise<void>;
  onError(): void;
  onLoad(): void;
};

/** Used in observers: canonical owner fields and the lease's metadata drive rendering together. */
export function useProfileImage(source: ProfileImageSource): ProfileImageView {
  const root = useContext(StoreContext);
  const projection = new ImageSource(root, source);
  const identity = JSON.stringify(projection.source);
  const enabled = projection.allowed;
  const held = useRef<{ root: RootStore; identity: string; lease: ProfileImageLease } | undefined>(undefined);
  const [, redraw] = useState(0);
  const current = held.current?.lease.current;
  // Acquisition belongs to the committed view, including StrictMode's mount/cleanup replay.
  useLayoutEffect(() => {
    const previous = held.current;
    if (previous?.root === root && previous.identity === identity && previous.lease.current && enabled) return;
    previous?.lease.dispose();
    held.current = undefined;
    if (enabled) {
      held.current = { root, identity, lease: root.asset.images.acquire(source) };
      redraw((value) => value + 1);
    }
  }, [root, identity, enabled, source, current]);
  useLayoutEffect(
    () => () => {
      held.current?.lease.dispose();
      held.current = undefined;
    },
    []
  );
  const lease =
    held.current?.root === root && held.current.identity === identity && enabled && held.current.lease.current
      ? held.current.lease
      : undefined;
  const { mutate } = useSessionSWR(lease?.canRead ? ["profile-image", lease.key] : null, async () => lease?.read(), {
    dedupingInterval: 0,
  });
  const stamp = lease?.stamp;
  const url = lease?.url;
  const expiry = lease?.expiry;
  const refresh = useCallback(() => {
    void mutate().catch(() => {
      /* The current lease owns the displayed error. */
    });
  }, [mutate]);
  useEffect(() => {
    if (!lease || !stamp) return;
    if (lease.expire(stamp)) {
      redraw((value) => value + 1);
      refresh();
      return;
    }
    if (!url || expiry === undefined) return;
    lease.arm();
    let timer: ReturnType<typeof setTimeout>;
    const check = () => {
      clearTimeout(timer);
      if (lease.expire(stamp)) {
        redraw((value) => value + 1);
        refresh();
      } else if (lease.url && lease.stamp === stamp)
        timer = setTimeout(check, Math.min(Math.max(1, expiry - Date.now()), 2_147_483_647));
    };
    check();
    document.addEventListener("visibilitychange", check);
    window.addEventListener("focus", check);
    window.addEventListener("online", check);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", check);
      window.removeEventListener("focus", check);
      window.removeEventListener("online", check);
    };
  }, [lease, stamp, url, expiry, refresh]);
  return {
    url,
    loading: lease?.loading ?? false,
    error: lease?.error,
    retry: async () => {
      lease?.requestRetry();
      await mutate();
    },
    onError: () => {
      if (lease && stamp && lease.fail(stamp)) refresh();
    },
    onLoad: () => {
      if (lease && stamp) lease.loaded(stamp);
    },
  };
}
