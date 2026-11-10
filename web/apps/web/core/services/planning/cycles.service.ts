/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type {
  ApiClient,
  Cycle,
  CycleCreate,
  CycleListItem,
  CycleProgress,
  CycleTransferResult,
  CycleUpdate,
} from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";

/** A project's cycles (M6 design 3.1–3.8): a span of days whose work items it plans, ended, frozen and archived. */
export class CyclesService {
  /** api: the client bound to the session of the stores that build this service (RootStore). */
  constructor(private readonly api: ApiClient) {}

  /** The project's cycles that are not archived, in nerve's order (sort order, then creation, then id), with counts. */
  async list(projectId: string): Promise<CycleListItem[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/cycles", { params: { path: { project_id: projectId } } })
    ).data;
  }

  /** The project's archived cycles, the latest archived first, with counts. */
  async listArchived(projectId: string): Promise<CycleListItem[]> {
    return unwrap(
      await this.api.GET("/api/v0/projects/{project_id}/cycles", {
        params: { path: { project_id: projectId }, query: { archived: true } },
      })
    ).data;
  }

  /** The cycle as nerve holds it, an archived one's too. */
  async get(cycleId: string): Promise<Cycle> {
    return unwrap(await this.api.GET("/api/v0/cycles/{cycle_id}", { params: { path: { cycle_id: cycleId } } }));
  }

  /** Creates a cycle, first of its project's; a draft names no dates. */
  async create(projectId: string, body: CycleCreate): Promise<Cycle> {
    return unwrap(
      await this.api.POST("/api/v0/projects/{project_id}/cycles", { params: { path: { project_id: projectId } }, body })
    );
  }

  /** Changes the fields body names; the answer is the cycle as nerve holds it. */
  async update(cycleId: string, body: CycleUpdate): Promise<Cycle> {
    return unwrap(await this.api.PATCH("/api/v0/cycles/{cycle_id}", { params: { path: { cycle_id: cycleId } }, body }));
  }

  /** Deletes a cycle. */
  async remove(cycleId: string): Promise<void> {
    unwrap(await this.api.DELETE("/api/v0/cycles/{cycle_id}", { params: { path: { cycle_id: cycleId } } }));
  }

  /** Archives a completed cycle; one archived already is answered as it is. */
  async archive(cycleId: string): Promise<Cycle> {
    return unwrap(
      await this.api.POST("/api/v0/cycles/{cycle_id}/archive", { params: { path: { cycle_id: cycleId } } })
    );
  }

  /** Unarchives a cycle; one not archived is answered as it is. */
  async unarchive(cycleId: string): Promise<Cycle> {
    return unwrap(
      await this.api.POST("/api/v0/cycles/{cycle_id}/unarchive", { params: { path: { cycle_id: cycleId } } })
    );
  }

  /** The cycle's status, its live progress, and the progress it was frozen with once it ended. */
  async progress(cycleId: string): Promise<CycleProgress> {
    return unwrap(
      await this.api.GET("/api/v0/cycles/{cycle_id}/progress", { params: { path: { cycle_id: cycleId } } })
    );
  }

  /** Moves a completed cycle's unfinished work items to the target cycle; the answer says how many moved. */
  async transfer(cycleId: string, targetCycleId: string): Promise<CycleTransferResult> {
    return unwrap(
      await this.api.POST("/api/v0/cycles/{cycle_id}/transfer-issues", {
        params: { path: { cycle_id: cycleId } },
        body: { target_cycle_id: targetCycleId },
      })
    );
  }
}
