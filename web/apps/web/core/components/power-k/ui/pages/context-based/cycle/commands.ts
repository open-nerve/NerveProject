/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "react-router";
import { LinkOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import type { TPowerKCommandConfig } from "@/components/power-k/core/types";
// hooks
import { useCopyLink } from "@/hooks/use-copy-link";

/**
 * The commands on a cycle's page (P5a spec 2.7, R2): the copy of its link, the route's cycle's, through the web app's
 * one copy, which says whether it could. The cycles' favorites are gone until M7 brings them back (M6 design 15 #13).
 */
export const usePowerKCycleContextBasedActions = (): TPowerKCommandConfig[] => {
  // navigation
  const { workspaceSlug, projectId, cycleId } = useParams();
  // hooks
  const copyLink = useCopyLink();
  // translation
  const { t } = useTranslation();

  return [
    {
      id: "copy_cycle_url",
      i18n_title: "power_k.contextual_actions.cycle.copy_url",
      icon: LinkOutline,
      group: "contextual",
      contextType: "cycle",
      type: "action",
      action: () =>
        copyLink(
          `/${workspaceSlug}/projects/${projectId}/cycles/${cycleId}`,
          t("power_k.contextual_actions.cycle.copy_url_toast_success")
        ),
      modifierShortcut: "cmd+shift+,",
      isEnabled: () => true,
      isVisible: () => true,
      closeOnSelect: true,
    },
  ];
};
