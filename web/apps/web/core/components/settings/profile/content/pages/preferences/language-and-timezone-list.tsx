/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import type { Language } from "@nerve/api-client";
import { SUPPORTED_LANGUAGES, toSupportedLanguage, useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { CustomSelect } from "@nerve/ui";
// components
import { TimezoneSelect } from "@/components/global";
import { StartOfWeekPreference } from "@/components/profile/start-of-week-preference";
import { SettingsControlItem } from "@/components/settings/control-item";
// hooks
import { useUser, useUserProfile } from "@/hooks/store/user";
import { useRefusalToast } from "@/hooks/use-refusal-toast";

export const ProfileSettingsLanguageAndTimezonePreferencesList = observer(
  function ProfileSettingsLanguageAndTimezonePreferencesList() {
    // store hooks
    const {
      data: user,
      updateCurrentUser,
      userProfile: { data: profile },
    } = useUser();
    const { updateUserProfile } = useUserProfile();
    // translation
    const { t } = useTranslation();
    const toastRefusal = useRefusalToast();

    const handleTimezoneChange = async (value: string) => {
      try {
        await updateCurrentUser({ user_timezone: value });
        setToast({
          title: t("toast.success"),
          message: t("power_k.preferences_actions.toast.timezone.success"),
          type: TOAST_TYPE.SUCCESS,
        });
      } catch (error) {
        toastRefusal(error);
      }
    };

    const handleLanguageChange = async (value: Language) => {
      try {
        await updateUserProfile({ language: value });
        setToast({
          title: t("toast.success"),
          message: t("power_k.preferences_actions.toast.generic.success"),
          type: TOAST_TYPE.SUCCESS,
        });
      } catch (error) {
        toastRefusal(error);
      }
    };

    // a language that is no longer supported shows as the fallback language, which is what the app uses
    const language = toSupportedLanguage(profile?.language);
    const languageLabel = SUPPORTED_LANGUAGES.find((l) => l.value === language)?.label;

    return (
      <div className="flex flex-col gap-y-1">
        <SettingsControlItem
          title={t("timezone")}
          description={t("timezone_setting")}
          control={<TimezoneSelect value={user?.user_timezone} onChange={handleTimezoneChange} />}
        />
        <SettingsControlItem
          title={t("language")}
          description={t("language_setting")}
          control={
            <CustomSelect
              value={language}
              label={languageLabel}
              onChange={handleLanguageChange}
              buttonClassName="border border-subtle-1"
              className="rounded-md"
              input
              placement="bottom-end"
            >
              {SUPPORTED_LANGUAGES.map((item) => (
                <CustomSelect.Option key={item.value} value={item.value}>
                  {item.label}
                </CustomSelect.Option>
              ))}
            </CustomSelect>
          }
        />
        <StartOfWeekPreference
          option={{
            title: t("account_settings.preferences.start_of_week.title"),
            description: t("account_settings.preferences.start_of_week.description"),
          }}
        />
      </div>
    );
  }
);
