/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { observer } from "mobx-react";
import { useTheme } from "next-themes";
// nerve imports
import type { I_THEME_OPTION } from "@nerve/constants";
import { THEME_OPTIONS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { setPromiseToast } from "@nerve/propel/toast";
// components
import { ThemeSwitch } from "@/components/core/theme/theme-switch";
import { SettingsControlItem } from "@/components/settings/control-item";
// helpers
import { errorMessageKey } from "@/helpers/authentication.helper";
// hooks
import { useUserProfile } from "@/hooks/store/user";

export const ThemeSwitcher = observer(function ThemeSwitcher(props: {
  option: {
    id: string;
    title: string;
    description: string;
  };
}) {
  // store hooks
  const { data: userProfile, updateUserTheme } = useUserProfile();
  // theme
  const { setTheme } = useTheme();
  // translation
  const { t } = useTranslation();
  // derived values
  const currentTheme = useMemo(() => {
    const userThemeOption = THEME_OPTIONS.find((option) => option.value === userProfile?.theme);
    return userThemeOption || null;
  }, [userProfile?.theme]);

  const handleThemeChange = useCallback(
    async (themeOption: I_THEME_OPTION) => {
      try {
        setTheme(themeOption.value);

        const updatePromise = updateUserTheme(themeOption.value);
        setPromiseToast(updatePromise, {
          loading: t("power_k.preferences_actions.toast.theme.updating"),
          success: {
            title: t("power_k.preferences_actions.toast.theme.updated"),
            message: () => t("power_k.preferences_actions.toast.theme.reloading"),
          },
          error: {
            title: t("toast.error"),
            message: (error) => t(errorMessageKey(error)),
          },
        });
        // Wait for the promise to resolve, then reload after showing toast
        await updatePromise;
        window.location.reload();
      } catch (error) {
        console.error("Error updating theme:", error);
      }
    },
    [setTheme, updateUserTheme, t]
  );

  if (!userProfile) return null;

  return (
    <SettingsControlItem
      title={t(props.option.title)}
      description={t(props.option.description)}
      control={
        <ThemeSwitch
          value={currentTheme}
          onChange={(themeOption) => {
            void handleThemeChange(themeOption);
          }}
        />
      }
    />
  );
});
