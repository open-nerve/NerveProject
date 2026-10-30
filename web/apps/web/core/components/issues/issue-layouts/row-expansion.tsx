/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext, useCallback, useContext, useLayoutEffect, useMemo, useRef, useState } from "react";
import type { Dispatch, ReactNode, SetStateAction } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { EIssuesStoreType } from "@nerve/types";
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssueStoreType } from "@/hooks/use-issue-layout-store";

type Expansion = { ids: ReadonlySet<string>; set: (id: string, update: SetStateAction<boolean>) => void };
const Context = createContext<Expansion | undefined>(undefined);
const EMPTY: ReadonlySet<string> = new Set();

/** UI intent survives a verification skeleton, never a different route, permission scope or layout host. */
export const RowExpansionProvider = observer(function RowExpansionProvider({
  children,
  viewId,
}: {
  children: ReactNode;
  viewId?: string;
}) {
  const { rootIssueStore: root } = useIssueDetail();
  const type = useIssueStoreType();
  const { workspaceSlug, projectId, userId } = useParams();
  const workspace = workspaceSlug ? root.rootStore.workspaceRoot.getWorkspaceBySlug(workspaceSlug) : undefined;
  const identity = root.permissionReads.identity(
    workspace?.id,
    projectId ? [projectId] : [],
    type === EIssuesStoreType.PROFILE
  );
  const scope = useMemo(
    () => ({ root, type, workspaceSlug, projectId, userId, viewId, identity }),
    [root, type, workspaceSlug, projectId, userId, viewId, identity]
  );
  const current = useRef(scope);
  current.current = scope;
  const [state, setState] = useState({ scope, ids: EMPTY });
  useLayoutEffect(() => {
    setState((previous) => (previous.scope === scope ? previous : { scope, ids: EMPTY }));
  }, [scope]);
  const ids = state.scope === scope ? state.ids : EMPTY;
  const set = useCallback(
    (id: string, update: SetStateAction<boolean>) => {
      if (current.current !== scope) return;
      setState((previous) => {
        const held = previous.scope === scope ? previous.ids : EMPTY;
        const expanded = typeof update === "function" ? update(held.has(id)) : update;
        if (expanded === held.has(id) && previous.scope === scope) return previous;
        const next = new Set(held);
        if (expanded) next.add(id);
        else next.delete(id);
        return { scope, ids: next };
      });
    },
    [scope]
  );
  const value = useMemo(() => ({ ids, set }), [ids, set]);
  return <Context.Provider value={value}>{children}</Context.Provider>;
});

export function useRowExpansion(id: string): [boolean, Dispatch<SetStateAction<boolean>>] {
  const owner = useContext(Context);
  // Standalone rows can still own their state; real layout roots supply the longer-lived owner.
  const [local, setLocal] = useState(false);
  const set = useCallback<Dispatch<SetStateAction<boolean>>>(
    (update) => {
      if (owner) owner.set(id, update);
      else setLocal(update);
    },
    [owner, id]
  );
  return [owner ? owner.ids.has(id) : local, set];
}
