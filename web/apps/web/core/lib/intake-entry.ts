/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { Project } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";

/**
 * Whether the caller is offered the ways into the project's intake (M7 design 3.11; P5a spec R21; M6-M7-07): the
 * sidebar's and the tab bar's intake item, and the command palette's command that goes to it. The project's intake is
 * turned on (intake_view) and the caller has a role in it, an admin's, a member's or a guest's, as the permission
 * store gives it by the address's workspace and project (never the router's; the workspace's admin who is its member
 * is its admin). Unlike planningEntryOn (planning-switch.ts), a guest is offered it, as he submits to the intake too,
 * and an archived project is not left out: its intake is read as ever, its page read-only (R27). A store-free module
 * of lib, as planning-switch.ts is.
 */
export function intakeEntryOn(project: Project | undefined, role: EUserPermissions | undefined): boolean {
  return (
    project !== undefined &&
    project.intake_view &&
    (role === EUserPermissions.ADMIN || role === EUserPermissions.MEMBER || role === EUserPermissions.GUEST)
  );
}
