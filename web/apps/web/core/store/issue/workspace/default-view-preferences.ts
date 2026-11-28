/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueListDisplayProperties } from "@nerve/api-client";
import type { WorkspacePageFilters } from "@/lib/issue-filters";
import { storage } from "@/lib/local-storage";
import { WORKSPACE_VIEW_CONTENT } from "@/store/views/view-content";
import type { PagePreferences, PagePreferencesService } from "../helpers/page-preferences-filter.store";

type Preferences = PagePreferences<WorkspacePageFilters>;
type Display = Pick<Preferences, "display_filters" | "display_properties">;

/** The browser's entry of the default views' displays: by account id, then by page id (M7/P6b spec 2.4). */
const KEY = "workspace_view_display";
/** What a default view shows before its page changes it: a new workspace's view's display (M7/P6b spec 2.11). */
const SHOWN: Display = {
  display_filters: WORKSPACE_VIEW_CONTENT.display_filters!,
  display_properties: WORKSPACE_VIEW_CONTENT.display_properties!,
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value);

/** The browser's entry, whole; none when it holds nothing, nothing readable, or refuses. */
function stored(): Record<string, unknown> {
  try {
    const raw = storage.get(KEY);
    const value: unknown = raw ? JSON.parse(raw) : {};
    return isRecord(value) ? value : {};
  } catch {
    return {};
  }
}

/**
 * A display read back from the browser, field by field: the spreadsheet without grouping, the calendar's defaults; an
 * order, the sub-issues, the empty groups and each property kept only with its type, its default otherwise. An order
 * nerve's list does not take falls back where it is used (show, workspaceListQuery).
 */
function readBack(entry: unknown): Display {
  const display = isRecord(entry) && isRecord(entry.display_filters) ? entry.display_filters : {};
  const shown = isRecord(entry) && isRecord(entry.display_properties) ? entry.display_properties : {};
  const base = SHOWN.display_filters;
  const properties = { ...SHOWN.display_properties };
  for (const key of Object.keys(properties) as (keyof IssueListDisplayProperties)[]) {
    const value = shown[key];
    if (typeof value === "boolean") properties[key] = value;
  }
  return {
    display_filters: {
      ...base,
      order_by: typeof display.order_by === "string" ? display.order_by : base.order_by,
      sub_issue: typeof display.sub_issue === "boolean" ? display.sub_issue : base.sub_issue,
      show_empty_groups:
        typeof display.show_empty_groups === "boolean" ? display.show_empty_groups : base.show_empty_groups,
    },
    display_properties: properties,
  };
}

/**
 * A default view's page's preferences (M7/P6b spec 2.4, R3; design §3.7), in nerve's words: its display and its
 * properties are the browser's, by account and page, so that entering the view again shows them; its filters are this
 * entry's alone, in memory, ended as the page leaves (discard). The browser's copy is not the handleIssuesLocalFilters
 * one, which checks the controls' words and would drop nerve's. Reads are answered at once; a browser that keeps
 * nothing or refuses fails nothing: the defaults are read, and a change is shown for the session.
 */
export class DefaultViewPreferences implements PagePreferencesService<WorkspacePageFilters> {
  /** This entry's filters, by page id. */
  private readonly filters = new Map<string, WorkspacePageFilters>();
  /** The displays changed in this session, by account and page id: the browser may not have kept them. */
  private readonly displays = new Map<string, Display>();

  /** account gives the caller's account id; without one the browser is neither read nor written. */
  constructor(private readonly account: () => string | undefined) {}

  private display(id: string): Display {
    const account = this.account();
    const held = this.displays.get(JSON.stringify([account ?? null, id]));
    if (held) return held;
    if (!account) return SHOWN;
    const entries = stored()[account];
    return readBack(isRecord(entries) ? entries[id] : undefined);
  }

  async get(id: string): Promise<Preferences> {
    return structuredClone({ filters: this.filters.get(id) ?? {}, ...this.display(id) });
  }

  update = async (id: string, body: Partial<Preferences>): Promise<Preferences> => {
    const { filters, ...change } = body;
    if (filters) this.filters.set(id, structuredClone(filters));
    if (change.display_filters || change.display_properties) {
      const display = structuredClone({ ...this.display(id), ...change });
      const account = this.account();
      this.displays.set(JSON.stringify([account ?? null, id]), display);
      if (account) this.keep(account, id, display);
    }
    return this.get(id);
  };

  /** Ends this entry's filters as the page leaves; its display stays (D-3). */
  discard(id: string): void {
    this.filters.delete(id);
  }

  private keep(account: string, id: string, display: Display): void {
    try {
      const entry = stored();
      const entries = isRecord(entry[account]) ? entry[account] : {};
      storage.set(KEY, JSON.stringify({ ...entry, [account]: { ...entries, [id]: display } }));
    } catch {
      // the browser keeps nothing: the session shows the change still
    }
  }
}
