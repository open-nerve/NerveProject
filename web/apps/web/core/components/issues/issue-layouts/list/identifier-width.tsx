/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { createContext, useContext, useLayoutEffect, useMemo, useState, type ReactNode } from "react";
import { makeAutoObservable } from "mobx";
import type { Issue, IssueDetail } from "@nerve/api-client";
import { useProject } from "@/hooks/store/use-project";
import { calculateIdentifierWidth } from "../utils";

// Only presentation measurements live here. Each mounted List owns its own registry.
class IdentifierWidths {
  maximum = 0;
  rows = new Map<object, number>();
  constructor() {
    makeAutoObservable(this, {}, { autoBind: true });
  }
  get width() {
    return this.rows.size ? this.maximum : 0;
  }
  register(owner: object, width: number) {
    if (width > 0) {
      this.rows.set(owner, width);
      this.maximum = Math.max(this.maximum, width);
    }
    return () => this.unregister(owner);
  }
  unregister(owner: object) {
    this.rows.delete(owner);
  }
  clear() {
    this.rows.clear();
    this.maximum = 0;
  }
}
const WidthContext = createContext<IdentifierWidths | undefined>(undefined);
export function IdentifierWidthScope({ children, owner }: { children: ReactNode; owner: object }) {
  const { widths } = useMemo(() => ({ owner, widths: new IdentifierWidths() }), [owner]);
  useLayoutEffect(() => widths.clear, [widths]);
  return <WidthContext.Provider value={widths}>{children}</WidthContext.Provider>;
}
export function useIdentifierWidth(enabled: boolean | undefined): number {
  const widths = useContext(WidthContext);
  return enabled ? (widths?.width ?? 0) : 0;
}
/** Register outside RenderIfVisible, so an already loaded row contributes even outside the viewport. */
export function useRegisterIdentifierWidth(issue: Issue | IssueDetail | undefined) {
  const widths = useContext(WidthContext);
  const { getProjectIdentifierById } = useProject();
  const prefix = issue ? getProjectIdentifierById(issue.project_id) : undefined;
  const sequence = issue?.sequence_id;
  const width =
    prefix && sequence != null && Number.isSafeInteger(sequence) && sequence > 0
      ? calculateIdentifierWidth(prefix.length, sequence)
      : 0;
  const [owner] = useState(() => ({}));
  useLayoutEffect(() => widths?.register(owner, width), [widths, owner, width]);
}
