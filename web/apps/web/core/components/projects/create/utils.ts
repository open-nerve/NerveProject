/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { RANDOM_EMOJI_CODES } from "@nerve/constants";
// local imports
import type { ProjectCreationForm } from "./use-create-project";

/** A new project's form as it opens: public, no lead, a random emoji its icon (M3 design 3.19). */
export const getProjectFormValues = (): ProjectCreationForm => ({
  description: "",
  logo_props: {
    in_use: "emoji",
    emoji: {
      value: RANDOM_EMOJI_CODES[Math.floor(Math.random() * RANDOM_EMOJI_CODES.length)],
    },
  },
  identifier: "",
  name: "",
  network: 2,
  project_lead_id: null,
});
