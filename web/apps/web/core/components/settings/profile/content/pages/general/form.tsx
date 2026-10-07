/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { Controller, useForm } from "react-hook-form";
import { UserOutline } from "@makeplane/propel/icons";
// nerve imports
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import type { User, UserUpdate } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";

import { getFileURL } from "@nerve/utils";
// components
import { DeactivateAccountModal } from "@/components/account/deactivate-account-modal";
import { CoverImage } from "@/components/common/cover-image";
import { SettingsBoxedControlItem } from "@/components/settings/boxed-control-item";
// hooks
import { useUser } from "@/hooks/store/user";
// lib
import { errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/lib/error-messages";

type TUserProfileForm = {
  first_name: string;
  last_name: string;
  display_name: string;
  email: string;
};

type Props = {
  user: User;
};

/**
 * The fields of UserUpdate the form has, whose errors show under them. The form checks only that the required
 * ones are filled: the rules of the names are nerve's (M2 design 4.2), and its refusal shows under the field.
 */
const FIELDS = ["first_name", "last_name", "display_name"] as const;

export const GeneralProfileSettingsForm = observer(function GeneralProfileSettingsForm(props: Props) {
  const { user } = props;
  // states
  const [isLoading, setIsLoading] = useState(false);
  const [deactivateAccountModal, setDeactivateAccountModal] = useState(false);
  // language support
  const { t } = useTranslation();
  // form info
  const {
    handleSubmit,
    watch,
    control,
    setError,
    formState: { errors },
  } = useForm<TUserProfileForm>({
    defaultValues: {
      first_name: user.first_name || "",
      last_name: user.last_name || "",
      display_name: user.display_name || "",
      email: user.email || "",
    },
  });
  // store hooks
  const { data: currentUser, updateCurrentUser } = useUser();

  const onSubmit = async (formData: TUserProfileForm) => {
    setIsLoading(true);
    const userPayload: UserUpdate = {
      first_name: formData.first_name,
      last_name: formData.last_name,
      display_name: formData?.display_name,
    };

    try {
      await updateCurrentUser(userPayload);
      setToast({ type: TOAST_TYPE.SUCCESS, title: t("toast.success"), message: t("profile_updated") });
    } catch (error) {
      // A refusal shows under the field it is about, anything else in a toast (M2 design 7.3).
      const fields = fieldErrorKeys(error);
      for (const field of FIELDS) {
        const key = fields[field];
        if (key !== undefined) setError(field, { type: "manual", message: t(key) });
      }
      if (needsErrorBanner(error, FIELDS))
        setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t(errorMessageKey(error)) });
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <DeactivateAccountModal isOpen={deactivateAccountModal} onClose={() => setDeactivateAccountModal(false)} />
      <form onSubmit={handleSubmit(onSubmit)} className="w-full">
        <div className="flex w-full flex-col gap-7">
          {/* The picture and the cover are shown, not uploaded, until the file storage (M5, M2 design 3.2) */}
          <div className="relative h-44 w-full">
            <CoverImage
              src={user.cover_image_url ?? undefined}
              className="h-44 w-full rounded-lg"
              alt={currentUser?.first_name ?? t("cover_image_alt")}
            />
            <div className="absolute -bottom-6 left-6 flex items-end justify-between">
              <div className="flex gap-3">
                <div className="flex h-16 w-16 items-center justify-center rounded-lg bg-surface-2">
                  {user.avatar_url ? (
                    <div className="relative h-16 w-16 overflow-hidden">
                      <img
                        src={getFileURL(user.avatar_url)}
                        className="absolute top-0 left-0 h-full w-full rounded-lg object-cover"
                        alt={currentUser?.display_name}
                      />
                    </div>
                  ) : (
                    <div className="h-16 w-16 rounded-md bg-layer-1 p-2">
                      <UserOutline className="h-full w-full text-secondary" />
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
          <div className="item-center mt-6 flex justify-between">
            <div className="flex flex-col">
              <div className="item-center flex text-16 font-medium text-secondary">
                <span>{`${watch("first_name")} ${watch("last_name")}`}</span>
              </div>
              <span className="text-13 tracking-tight text-tertiary">{watch("email")}</span>
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <div className="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-2 xl:grid-cols-3">
              <div className="flex flex-col gap-1">
                <h4 className="text-13 font-medium text-secondary">
                  {t("first_name")}&nbsp;
                  <span className="text-danger-primary">*</span>
                </h4>
                <Controller
                  control={control}
                  name="first_name"
                  rules={{
                    required: t("common.errors.required"),
                  }}
                  render={({ field: { value, onChange, ref } }) => (
                    <Field name="first_name" invalid={Boolean(errors.first_name)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="first_name"
                          name="first_name"
                          type="text"
                          value={value}
                          onChange={onChange}
                          ref={ref}
                          placeholder={t("enter_your_first_name")}
                          autoComplete="on"
                        />
                      </InputGroup>
                    </Field>
                  )}
                />
                {errors.first_name && <span className="text-11 text-danger-primary">{errors.first_name.message}</span>}
              </div>
              <div className="flex flex-col gap-1">
                <h4 className="text-13 font-medium text-secondary">{t("last_name")}</h4>
                <Controller
                  control={control}
                  name="last_name"
                  render={({ field: { value, onChange, ref } }) => (
                    <Field name="last_name" invalid={Boolean(errors.last_name)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="last_name"
                          name="last_name"
                          type="text"
                          value={value}
                          onChange={onChange}
                          ref={ref}
                          placeholder={t("enter_your_last_name")}
                          autoComplete="on"
                        />
                      </InputGroup>
                    </Field>
                  )}
                />
                {errors.last_name && <span className="text-11 text-danger-primary">{errors.last_name.message}</span>}
              </div>
              <div className="flex flex-col gap-1">
                <h4 className="text-13 font-medium text-secondary">
                  {t("display_name")}&nbsp;
                  <span className="text-danger-primary">*</span>
                </h4>
                <Controller
                  control={control}
                  name="display_name"
                  rules={{
                    required: t("common.errors.required"),
                  }}
                  render={({ field: { value, onChange, ref } }) => (
                    <Field name="display_name" invalid={Boolean(errors?.display_name)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="display_name"
                          name="display_name"
                          type="text"
                          value={value}
                          onChange={onChange}
                          ref={ref}
                          placeholder={t("enter_your_display_name")}
                        />
                      </InputGroup>
                    </Field>
                  )}
                />
                {errors?.display_name && (
                  <span className="text-11 text-danger-primary">{errors?.display_name?.message}</span>
                )}
              </div>
              <div className="flex flex-col gap-1">
                <h4 className="text-13 font-medium text-secondary">
                  {t("auth.common.email.label")}&nbsp;
                  <span className="text-danger-primary">*</span>
                </h4>
                <Controller
                  control={control}
                  name="email"
                  rules={{
                    required: t("common.errors.required"),
                  }}
                  render={({ field: { value, ref } }) => (
                    <Field name="email" invalid={Boolean(errors.email)}>
                      <InputGroup size="2xl">
                        <Input
                          size="2xl"
                          id="email"
                          name="email"
                          type="email"
                          value={value}
                          ref={ref}
                          placeholder={t("auth.common.email.placeholder")}
                          autoComplete="on"
                          disabled
                        />
                      </InputGroup>
                    </Field>
                  )}
                />
              </div>
            </div>
          </div>
          <div>
            <Button variant="primary" type="submit" loading={isLoading}>
              {isLoading ? t("saving") : t("save_changes")}
            </Button>
          </div>
        </div>
      </form>
      <div className="mt-10">
        <SettingsBoxedControlItem
          title={t("deactivate_account")}
          description={t("deactivate_account_description")}
          control={
            <Button variant="error-outline" onClick={() => setDeactivateAccountModal(true)}>
              {t("deactivate_account")}
            </Button>
          }
        />
      </div>
    </>
  );
});
