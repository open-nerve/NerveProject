/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observable, reaction } from "mobx";
import type { RootStore } from "@/store/root.store";

type Fact = {
  workspaceID: string;
  kind: "workspace" | "member" | "project";
  projectID?: string;
  signature: string | undefined;
};
const key = (workspaceID: string, kind: Fact["kind"], projectID?: string) =>
  JSON.stringify([workspaceID, kind, projectID ?? null]);

/** Metadata for one issue root only: observe semantic permissions, never duplicate entities or directories. */
export class PermissionReads {
  private serial = 0;
  private readonly watermarks = observable.map<string, number>();
  private readonly workspaceProjects = observable.map<string, number>();
  private facts = new Map<string, Fact>();
  private readonly disposeObservation: () => void;
  constructor(
    private readonly stores: RootStore,
    private readonly changed: (workspaceID: string, projectID?: string, memberOnly?: boolean) => void
  ) {
    this.disposeObservation = reaction(
      () => this.snapshot(),
      (facts) => this.observe(facts),
      { fireImmediately: true }
    );
  }
  private snapshot(): Fact[] {
    const facts: Fact[] = [];
    for (const workspace of this.stores.workspaceRoot.workspaces ?? []) {
      const workspaceFact: Fact = { workspaceID: workspace.id, kind: "workspace", signature: workspace.slug };
      const memberFact: Fact = {
        workspaceID: workspace.id,
        kind: "member",
        signature: workspace.role === 5 ? undefined : String(workspace.role),
      };
      const projects = this.stores.projectRoot.project.knownProjectIds(workspace.id).map((projectID): Fact => {
        const project = this.stores.projectRoot.project.getProjectById(projectID);
        const effective = this.stores.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(
          workspace.slug,
          projectID
        );
        return {
          workspaceID: workspace.id,
          kind: "project",
          projectID,
          signature:
            project && effective !== undefined
              ? JSON.stringify([
                  effective,
                  project.member_role,
                  project.member_role === 5 ? project.guest_view_all_features : null,
                ])
              : undefined,
        };
      });
      facts.push(workspaceFact, memberFact, ...projects);
    }
    return facts;
  }
  private observe(facts: Fact[]): void {
    const next = new Map(facts.map((fact) => [key(fact.workspaceID, fact.kind, fact.projectID), fact]));
    for (const [id, previous] of this.facts) {
      const now = next.get(id);
      if (previous.signature !== undefined && previous.signature !== now?.signature) {
        const serial = ++this.serial;
        this.watermarks.set(id, serial);
        if (previous.projectID) this.workspaceProjects.set(previous.workspaceID, serial);
        this.changed(previous.workspaceID, previous.projectID, previous.kind === "member");
      }
      if (!now) next.set(id, { ...previous, signature: undefined });
    }
    this.facts = next;
  }
  capture(): number {
    return this.serial;
  }
  current(serial: number, workspaceID: string | undefined, projectIDs: readonly string[], member = false): boolean {
    if (!workspaceID && !projectIDs.length) return serial >= this.serial;
    if (workspaceID && (this.watermarks.get(key(workspaceID, "workspace")) ?? 0) > serial) return false;
    if (member && workspaceID && (this.watermarks.get(key(workspaceID, "member")) ?? 0) > serial) return false;
    if (projectIDs.length)
      return projectIDs.every((id) => {
        const workspace = workspaceID ?? [...this.facts.values()].find((fact) => fact.projectID === id)?.workspaceID;
        return (
          !workspace ||
          ((this.watermarks.get(key(workspace, "workspace")) ?? 0) <= serial &&
            (this.watermarks.get(key(workspace, "project", id)) ?? 0) <= serial)
        );
      });
    return !workspaceID || (this.workspaceProjects.get(workspaceID) ?? 0) <= serial;
  }
  identity(workspaceID: string | undefined, projectIDs: readonly string[], member = false): string {
    if (!workspaceID) return String(this.serial);
    const marks = [this.watermarks.get(key(workspaceID, "workspace")) ?? 0];
    if (member) marks.push(this.watermarks.get(key(workspaceID, "member")) ?? 0);
    if (projectIDs.length)
      for (const id of projectIDs) marks.push(this.watermarks.get(key(workspaceID, "project", id)) ?? 0);
    else marks.push(this.workspaceProjects.get(workspaceID) ?? 0);
    return String(Math.max(...marks));
  }
  retire(): void {
    this.disposeObservation();
  }
}
