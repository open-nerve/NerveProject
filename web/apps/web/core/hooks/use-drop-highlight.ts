/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useLayoutEffect, useMemo } from "react";
import scrollIntoView from "smooth-scroll-into-view-if-needed";
import { HIGHLIGHT_CLASS } from "@/components/issues/issue-layouts/utils";

/** A host owns the delayed feedback and every animation frame, including smooth scrolling after confirmation. */
export function useDropHighlight(scope: string) {
  const owner = useMemo(() => ({ scope, active: false, cancel: new Map<string, () => void>() }), [scope]);
  useLayoutEffect(() => {
    owner.active = true;
    return () => {
      owner.active = false;
      for (const cancel of owner.cancel.values()) cancel();
      owner.cancel.clear();
    };
  }, [owner]);
  return useCallback(
    (elementId: string | undefined, scroll = true, current: () => boolean = () => true) => {
      if (!elementId || !owner.active || !current()) return;
      owner.cancel.get(elementId)?.();
      let element: HTMLElement | null = null;
      let frame = 0;
      const cancel = () => {
        clearTimeout(timer);
        cancelAnimationFrame(frame);
        element?.classList.remove(HIGHLIGHT_CLASS);
        owner.cancel.delete(elementId);
      };
      const timer = setTimeout(() => {
        if (!owner.active || !current()) {
          cancel();
          return;
        }
        element = document.getElementById(elementId);
        if (!element) {
          cancel();
          return;
        }
        element.classList.add(HIGHLIGHT_CLASS);
        if (!scroll) return;
        // The library computes all scroll containers; this host owns cancellable animation scheduling.
        void scrollIntoView(element, {
          block: "center",
          behavior: (actions) => {
            const initial = actions.map(({ el }) => ({ left: el.scrollLeft, top: el.scrollTop }));
            const start = performance.now();
            const animate = (now: number) => {
              if (!owner.active || !current() || !element?.isConnected) {
                cancel();
                return;
              }
              const elapsed = Math.min((now - start) / 1500, 1);
              const eased = 1 - (1 - elapsed) ** 5;
              actions.forEach(({ el, left, top }, index) => {
                el.scrollLeft = initial[index].left + (left - initial[index].left) * eased;
                el.scrollTop = initial[index].top + (top - initial[index].top) * eased;
              });
              if (elapsed < 1) frame = requestAnimationFrame(animate);
            };
            frame = requestAnimationFrame(animate);
          },
        });
      }, 200);
      owner.cancel.set(elementId, cancel);
    },
    [owner]
  );
}
