/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// Exact legacy M7 transport; remove when the intake API is implemented.
type TIntakeDescriptionVersion = {
  created_at: string;
  created_by: string | null;
  id: string;
  last_saved_at: string;
  owned_by: string;
  project: string;
  updated_at: string;
  updated_by: string | null;
};

export type TIntakeDescriptionVersionDetails = TIntakeDescriptionVersion & {
  description_html: string | null;
  description_json: object | null;
  description_stripped: string | null;
};

export type TIntakeDescriptionVersionsListResponse = {
  cursor: string;
  next_cursor: string | null;
  next_page_results: boolean;
  page_count: number;
  prev_cursor: string | null;
  prev_page_results: boolean;
  results: TIntakeDescriptionVersion[];
  total_pages: number;
  total_results: number;
};
