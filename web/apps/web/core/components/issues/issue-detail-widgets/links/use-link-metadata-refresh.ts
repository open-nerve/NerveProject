/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useDetailScope } from "../use-detail-scope";

/** A mounted link widget schedules bounded reads; the resource remains the only owner of link data. */
export function useLinkMetadataRefresh({
  workspaceSlug,
  projectId,
  issueId,
  active,
}: {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  active: boolean;
}) {
  const { link, rootIssueStore: root } = useIssueDetail();
  const capture = useDetailScope(workspaceSlug, projectId, issueId);
  const workspace = root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug);
  const permission = root.permissionReads.identity(workspace?.id, [projectId]);
  const links = useCallback(
    () =>
      (link.getLinksByIssueId(issueId) ?? []).flatMap((id) => {
        const item = link.getLinkById(issueId, id);
        return item ? [item] : [];
      }),
    [link, issueId]
  );
  const identity = useCallback(
    () =>
      JSON.stringify(
        links()
          .map(({ id, url }) => [id, url])
          .toSorted()
      ),
    [links]
  );
  const needsMetadata = useCallback(() => links().some((item) => Object.keys(item.metadata).length === 0), [links]);
  const key = identity();
  const waiting = needsMetadata();
  const error = link.getError(issueId);
  const [attempt, setAttempt] = useState(0);
  const scopeKey = JSON.stringify([workspaceSlug, projectId, issueId, permission, key, active, attempt]);
  const owner = useMemo(
    () => ({ root, scopeKey, active: false, expired: false, count: 0, started: Date.now() }),
    [root, scopeKey]
  );
  const [expiredOwner, setExpiredOwner] = useState<typeof owner>();
  // This component also waits for an older URL's read to settle before starting the next URL's loop.
  const flight = useRef<Promise<unknown> | undefined>(undefined);
  const [busyOwner, setBusyOwner] = useState<typeof owner>();

  useEffect(() => {
    if (!active || !waiting || error !== undefined) return;
    owner.active = true;
    const scope = capture();
    const current = () => owner.active && scope() && identity() === key;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const expire = () => {
      if (!current() || !needsMetadata()) return;
      owner.expired = true;
      setExpiredOwner(owner);
      clearTimeout(timer);
    };
    const deadline = setTimeout(expire, Math.max(0, 30000 - (Date.now() - owner.started)));
    const schedule = () => {
      if (!current() || owner.expired || !needsMetadata() || link.getError(issueId) !== undefined) return;
      if (owner.count >= 30 || Date.now() - owner.started >= 30000) {
        expire();
        return;
      }
      timer = setTimeout(() => void read(), 1000);
    };
    const read = async () => {
      if (!current() || owner.expired || !needsMetadata()) return;
      if (flight.current) {
        schedule();
        return;
      }
      owner.count += 1;
      const request = link.fetchLinks(workspaceSlug, projectId, issueId);
      flight.current = request;
      setBusyOwner(owner);
      try {
        await request;
      } catch {
        /* The resource exposes a current refusal; a retired read has no UI followup. */
      } finally {
        if (flight.current === request) flight.current = undefined;
        if (current()) {
          setBusyOwner(undefined);
          schedule();
        }
      }
    };
    schedule();
    return () => {
      owner.active = false;
      clearTimeout(timer);
      clearTimeout(deadline);
    };
  }, [owner, active, waiting, error, capture, link, workspaceSlug, projectId, issueId, key, needsMetadata, identity]);

  const retry = async () => {
    const current = capture();
    if (!active || !current() || flight.current) return;
    setAttempt((value) => value + 1);
    const request = link.retry(workspaceSlug, projectId, issueId);
    flight.current = request;
    try {
      await request;
    } catch {
      /* Keep the resource's error and its read-only retry. */
    } finally {
      if (flight.current === request) flight.current = undefined;
    }
  };
  return {
    waiting: active && waiting && error === undefined,
    expired: waiting && expiredOwner === owner,
    busy: busyOwner === owner,
    retry,
  };
}
