/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observable, runInAction } from "mobx";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueRootStore } from "../root.store";
import { DetailResource } from "./resource";

/** Issue IDs are globally unique; each resource still binds its original workspace/project proof. */
export class IssueResources<V> {
  private readonly entries = observable.map<string, DetailResource<V>>({}, { deep: false });
  constructor(private readonly root: IssueRootStore) {}
  get(id: string): DetailResource<V> | undefined {
    return this.entries.get(id);
  }
  entry(workspaceSlug: string, projectID: string, id: string): DetailResource<V> {
    this.root.assertProjectReadable(workspaceSlug, projectID);
    const scope = this.root.scope(workspaceSlug, projectID);
    const known = this.root.issues.held(id);
    if (known === null || (known && (known.project_id !== projectID || known.workspace_id !== scope.workspaceID)))
      throw new SessionChangedError();
    const held = this.entries.get(id);
    if (held) {
      if (held.scope.workspaceID !== scope.workspaceID || !held.scope.projectIDs.includes(projectID))
        throw new SessionChangedError();
      return held;
    }
    const created = new DetailResource<V>(this.root, { workspaceID: scope.workspaceID, projectIDs: [projectID] });
    runInAction(() => this.entries.set(id, created));
    return created;
  }
}
