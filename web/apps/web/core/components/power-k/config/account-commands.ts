/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { LogOutOutline } from "@makeplane/propel/icons";
// components
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
// hooks
import { useSignOut } from "@/hooks/use-sign-out";

/**
 * Account commands - Account related commands
 */
export const usePowerKAccountCommands = (): TPowerKCommandConfig[] => {
  const signOut = useSignOut();

  return [
    {
      id: "sign_out",
      type: "action",
      group: "account",
      i18n_title: "power_k.account_actions.sign_out",
      icon: LogOutOutline,
      action: () => void signOut(),
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
  ];
};
