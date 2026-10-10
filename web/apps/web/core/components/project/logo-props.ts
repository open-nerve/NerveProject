/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { LogoProps } from "@nerve/api-client";
import type { TChangeHandlerProps } from "@nerve/propel/emoji-icon-picker";

/**
 * A project's icon as nerve keeps it (LogoProps, M3 design 3.19) from what the emoji picker picked: an emoji, by its
 * code, or an icon, by its name and colour. The creation's form and the general settings' both set it so.
 */
export function logoPropsOf(picked: TChangeHandlerProps): LogoProps {
  if (picked.type === "emoji") return { in_use: "emoji", emoji: { value: picked.value } };
  return { in_use: "icon", icon: { name: picked.value.name, color: picked.value.color } };
}
