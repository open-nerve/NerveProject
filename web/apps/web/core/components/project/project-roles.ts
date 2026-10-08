/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { ProjectRole } from "@nerve/api-client";

/**
 * A project's roles, the lowest first, as its role selects offer them, each labelled by ROLE: a select gives the
 * picked role's number, the ProjectRole nerve's ProjectMemberNew and ProjectMemberUpdate take (a string is refused).
 */
export const PROJECT_ROLES: ProjectRole[] = [5, 15, 20];
