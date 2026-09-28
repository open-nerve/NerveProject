/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { EmptyStateCompact } from "@nerve/propel/empty-state";
// components
import { CreateApiTokenModal } from "@/components/api-token/modal/create-token-modal";
import { ApiTokenList } from "@/components/api-token/token-list";
import { ProfileSettingsHeading } from "@/components/settings/profile/heading";

export function APITokensProfileSettings() {
  // states
  const [isCreateTokenModalOpen, setIsCreateTokenModalOpen] = useState(false);
  // translation
  const { t } = useTranslation();

  return (
    <div className="size-full">
      <CreateApiTokenModal isOpen={isCreateTokenModalOpen} onClose={() => setIsCreateTokenModalOpen(false)} />
      <ProfileSettingsHeading
        title={t("account_settings.api_tokens.title")}
        description={t("account_settings.api_tokens.description")}
        control={
          <Button variant="primary" size="lg" onClick={() => setIsCreateTokenModalOpen(true)}>
            {t("workspace_settings.settings.api_tokens.add_token")}
          </Button>
        }
      />
      <div className="mt-7">
        <ApiTokenList
          empty={
            <EmptyStateCompact
              assetKey="token"
              assetClassName="size-20"
              title={t("settings_empty_state.tokens.title")}
              description={t("settings_empty_state.tokens.description")}
              actions={[
                {
                  label: t("settings_empty_state.tokens.cta_primary"),
                  onClick: () => {
                    setIsCreateTokenModalOpen(true);
                  },
                },
              ]}
              align="start"
              rootClassName="py-20"
            />
          }
        />
      </div>
    </div>
  );
}
