/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { View, ViewFilters } from "@nerve/api-client";
import type { ListPageFilters } from "@/lib/issue-filters";
import type { PagePreferences, PagePreferencesService } from "./page-preferences-filter.store";

/** A view's page's draft of the view: what the view lists and how, in nerve's words, the page's filters `F`. */
type Draft<F extends ListPageFilters> = PagePreferences<F>;

/**
 * The draft service's refusal of a read of a view that made no draft (P6a spec 2.6): its page left before nerve
 * answered, or the read gave no view. The page's read takes it as no failure (use-view-page-read.ts).
 */
export class NoViewDraftError extends Error {
  constructor(viewId: string, why: string) {
    super(`No draft of view ${viewId}: ${why}`);
    this.name = "NoViewDraftError";
  }
}

/**
 * A view's page's preferences (P6a spec 2.6, R2): an in-memory draft of the view, by its id, in nerve's words, which
 * the page changes without sending anything and "Update view" saves (§2.7). One read of the view (the view store's, so
 * that its answer is the store's too) seeds it; while it lasts nothing is read again. The page ends it as it leaves
 * (discard): each view's drafts have a generation of their own (re-96 I2), so that a seed answered after the page left
 * makes no draft, and nerve's answer to Update view replaces only a draft still held. The page-preferences base is
 * untouched: it writes the page's filters only from a draft this gives. A project's view's page and a workspace's
 * view's page each hold their own (M7/P6b spec 2.4): filtersOf gives the view's filters as the page holds them.
 */
export class ViewDraftService<F extends ListPageFilters> implements PagePreferencesService<F> {
  private readonly drafts = new Map<string, Draft<F>>();
  private readonly generations = new Map<string, number>();

  constructor(
    private readonly read: (viewId: string) => Promise<View | null | undefined>,
    private readonly filtersOf: (filters: ViewFilters) => F
  ) {}

  private draftOf(view: View): Draft<F> {
    return {
      filters: this.filtersOf(view.filters),
      display_filters: view.display_filters,
      display_properties: view.display_properties,
    };
  }

  private generation(viewId: string): number {
    return this.generations.get(viewId) ?? 0;
  }

  /**
   * The view's draft, a copy of it: the one held, or else one seeded by a read of the view, unless the page ended the
   * draft while the read was out, or the read gave no view (one deleted here meanwhile, or a read a newer one overtook).
   */
  async get(viewId: string): Promise<Draft<F>> {
    const held = this.drafts.get(viewId);
    if (held) return { ...held };
    const generation = this.generation(viewId);
    const view = await this.read(viewId);
    if (this.generation(viewId) !== generation)
      throw new NoViewDraftError(viewId, "its page left before nerve answered");
    if (!view) throw new NoViewDraftError(viewId, "the read gave no view");
    const draft = this.draftOf(view);
    this.drafts.set(viewId, draft);
    return { ...draft };
  }

  /** Replaces each field the change names, whole, in the draft held, and gives the whole draft; sends nothing. */
  update = async (viewId: string, body: Partial<Draft<F>>): Promise<Draft<F>> => {
    const held = this.drafts.get(viewId);
    if (!held) throw new NoViewDraftError(viewId, "none is held");
    const draft = { ...held, ...body };
    this.drafts.set(viewId, draft);
    return { ...draft };
  };

  /** Replaces the draft held with nerve's answer to Update view (§2.7); a draft the page ended stays ended. */
  adopt(viewId: string, view: View): void {
    if (this.drafts.has(viewId)) this.drafts.set(viewId, this.draftOf(view));
  }

  /** Ends the view's draft, and any seed of it out: the next get reads the view again. */
  discard(viewId: string): void {
    this.drafts.delete(viewId);
    this.generations.set(viewId, this.generation(viewId) + 1);
  }
}
