/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { RANDOM_EMOJI_CODES } from "@nerve/constants";
import type { ProjectCreate } from "@nerve/api-client";

export const getProjectFormValues = (): ProjectCreate => ({
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
});
