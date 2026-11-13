/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useEffect, useMemo, useRef } from "react";
// nerve imports
import type { ModuleLink } from "@nerve/api-client";
// hooks
import { useModules } from "@/hooks/store/use-modules";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

/** A read of the links once a second, at most thirty of them, none thirty seconds after it began (M4's bounds). */
const EVERY = 1000;
const READS = 30;
const WINDOW = 30_000;

/** Whether nerve has yet to read a link's page: its metadata is empty (a failed read's carries its error). */
const unread = (link: ModuleLink) => Object.keys(link.metadata).length === 0;

/**
 * Reads a module's links again while nerve reads their pages (P6a spec 2.8, R10): nerve answers a link added, or one
 * whose address changed, with its metadata empty, and reads its page in the background. While one of the links the
 * caller shows has empty metadata, this reads the module's links again once a second, through the SWR read of them its
 * page makes (MODULE_LINKS, revalidated; not a fetch of its own), one read at a time, at most thirty reads and none
 * thirty seconds after it began; it stops once every link has metadata, once nerve refuses a read (SWR's error), and as
 * the caller goes. It reads nothing while no link wants metadata. The links of another session (its stores), a link
 * added or an address changed start it again; a read still out from before is waited for, never overlapped.
 */
export function useModuleLinkMetadataRefresh(moduleId: string): void {
  // store hooks
  const modules = useModules();
  const { mutate, error } = useSessionSWR(["MODULE_LINKS", moduleId], (id) => modules.fetchLinks(id), {
    revalidateOnMount: false,
  });
  const links = modules.getModuleLinks(moduleId) ?? [];
  const identity = JSON.stringify(links.map(({ id, url }) => [id, url]).toSorted());
  const waiting = links.some(unread);
  const refused = error !== undefined;
  // the reads of these links, of this session's stores, from the moment they first show
  const span = useMemo(() => ({ modules, identity, reads: 0, began: Date.now() }), [modules, identity]);
  // the read still out, of these links or of the links before
  const flight = useRef<Promise<unknown> | undefined>(undefined);

  useEffect(() => {
    if (!waiting || refused) return;
    let live = true;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const open = () => live && span.reads < READS && Date.now() - span.began < WINDOW;
    const schedule = () => {
      if (open()) timer = setTimeout(() => void read(), EVERY);
    };
    const read = async () => {
      if (!open()) return;
      if (flight.current) {
        schedule();
        return;
      }
      span.reads += 1;
      const request = mutate();
      flight.current = request;
      try {
        await request;
      } finally {
        if (flight.current === request) flight.current = undefined;
        schedule();
      }
    };
    schedule();
    return () => {
      live = false;
      clearTimeout(timer);
    };
  }, [span, waiting, refused, mutate]);
}
