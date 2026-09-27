/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { FormEvent } from "react";
import { useMemo, useRef, useState } from "react";
import { observer } from "mobx-react";
// icons
import { CloseCircleOutline, HideOutline, ShowOutline, WarningCircleOutline } from "@makeplane/propel/icons";
// nerve imports
import { Banner } from "@makeplane/propel/components/banner";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { E_PASSWORD_STRENGTH } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { PasswordStrengthIndicator, Spinner } from "@nerve/ui";
import { checkEmailValidity, getPasswordStrength } from "@nerve/utils";
// helpers
import { EAuthModes, errorMessageKey, fieldErrorKeys, needsErrorBanner } from "@/helpers/authentication.helper";
// hooks
import { useUser } from "@/hooks/store/user";
import { usePasswordStrengthLabels } from "@/hooks/use-password-strength-labels";

type Props = {
  mode: EAuthModes;
};

type TPasswordFormValues = {
  email: string;
  password: string;
  confirm_password: string;
};

const defaultValues: TPasswordFormValues = {
  email: "",
  password: "",
  confirm_password: "",
};

/** The fields whose messages from nerve show under them; a message for any other field shows above the form. */
const NERVE_FIELDS = ["email", "password"];

/**
 * The sign-in and sign-up form (M2 design 7.3). It calls nerve through the store; on success the token
 * manager keeps the session and AuthenticationWrapper moves on. A refusal shows here, the email kept: a
 * field's message under the field, and the problem's message above the form unless every field it names is
 * one of the form's.
 */
export const AuthPasswordForm = observer(function AuthPasswordForm(props: Props) {
  const { mode } = props;
  const { t } = useTranslation();
  const passwordStrengthLabels = usePasswordStrengthLabels();
  const { signIn, signUp } = useUser();
  // ref
  const emailInputRef = useRef<HTMLInputElement>(null);
  // states
  const [passwordFormData, setPasswordFormData] = useState<TPasswordFormValues>(defaultValues);
  const [showPassword, setShowPassword] = useState({
    password: false,
    retypePassword: false,
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isRetryPasswordInputFocused, setIsRetryPasswordInputFocused] = useState(false);
  const [isBannerMessage, setBannerMessage] = useState(false);
  const [isEmailInputFocused, setIsEmailInputFocused] = useState(false);
  // the last refusal of nerve, until the form changes
  const [submitError, setSubmitError] = useState<unknown>(undefined);

  const handleShowPassword = (key: keyof typeof showPassword) =>
    setShowPassword((prev) => ({ ...prev, [key]: !prev[key] }));

  // an edit answers the last message, whichever it was
  const handleFormChange = (key: keyof TPasswordFormValues, value: string) => {
    setSubmitError(undefined);
    setBannerMessage(false);
    setPasswordFormData((prev) => ({ ...prev, [key]: value }));
  };

  const isEmailInvalid = passwordFormData.email.length > 0 && !checkEmailValidity(passwordFormData.email);

  const isButtonDisabled = useMemo(
    () =>
      isSubmitting ||
      passwordFormData.email.length === 0 ||
      isEmailInvalid ||
      !passwordFormData.password ||
      (mode === EAuthModes.SIGN_UP && passwordFormData.password !== passwordFormData.confirm_password),
    [
      isSubmitting,
      isEmailInvalid,
      mode,
      passwordFormData.confirm_password,
      passwordFormData.email,
      passwordFormData.password,
    ]
  );

  const password = passwordFormData.password;
  const confirmPassword = passwordFormData.confirm_password;
  const renderPasswordMatchError = !isRetryPasswordInputFocused || confirmPassword.length >= password.length;
  const fieldErrors = fieldErrorKeys(submitError);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (mode === EAuthModes.SIGN_UP && getPasswordStrength(password) !== E_PASSWORD_STRENGTH.STRENGTH_VALID) {
      setBannerMessage(true);
      return;
    }
    setIsSubmitting(true);
    setSubmitError(undefined);
    const credentials = { email: passwordFormData.email, password };
    try {
      await (mode === EAuthModes.SIGN_IN ? signIn(credentials) : signUp(credentials));
    } catch (error) {
      setSubmitError(error);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <>
      {isBannerMessage && mode === EAuthModes.SIGN_UP && (
        <Banner
          placement="inline"
          variant="danger"
          description={t("auth.sign_up.errors.password.strength")}
          onDismiss={() => setBannerMessage(false)}
        />
      )}
      {submitError !== undefined && needsErrorBanner(submitError, NERVE_FIELDS) && (
        <Banner
          placement="inline"
          variant="danger"
          description={t(errorMessageKey(submitError))}
          onDismiss={() => setSubmitError(undefined)}
        />
      )}
      <form className="space-y-4" noValidate onSubmit={(event) => void handleSubmit(event)}>
        <div className="space-y-1">
          <label htmlFor="email" className="text-13 font-medium text-tertiary">
            {t("auth.common.email.label")}
          </label>
          <Field name="email" invalid={(!isEmailInputFocused && isEmailInvalid) || Boolean(fieldErrors.email)}>
            <InputGroup
              size="2xl"
              onFocus={() => setIsEmailInputFocused(true)}
              onBlur={() => setIsEmailInputFocused(false)}
            >
              <Input
                size="2xl"
                id="email"
                name="email"
                type="email"
                value={passwordFormData.email}
                onChange={(e) => handleFormChange("email", e.target.value)}
                placeholder={t("auth.common.email.placeholder")}
                autoComplete="email"
                ref={emailInputRef}
              />
              {passwordFormData.email.length > 0 && (
                <button
                  type="button"
                  className="grid size-5 place-items-center"
                  onClick={() => {
                    handleFormChange("email", "");
                    emailInputRef.current?.focus();
                  }}
                  aria-label={t("aria_labels.auth_forms.clear_email")}
                  tabIndex={-1}
                >
                  <CloseCircleOutline className="size-5 text-placeholder" />
                </button>
              )}
            </InputGroup>
          </Field>
          {isEmailInvalid && !isEmailInputFocused && (
            <p className="flex items-center gap-1 px-0.5 text-11 text-danger-primary">
              <WarningCircleOutline height={12} width={12} />
              {t("auth.common.email.errors.invalid")}
            </p>
          )}
          {fieldErrors.email && <p className="px-0.5 text-11 text-danger-primary">{t(fieldErrors.email)}</p>}
        </div>

        <div className="space-y-1">
          <label htmlFor="password" className="text-13 font-medium text-tertiary">
            {mode === EAuthModes.SIGN_IN ? t("auth.common.password.label") : t("auth.common.password.set_password")}
          </label>
          <Field name="password" invalid={Boolean(fieldErrors.password)}>
            <InputGroup size="2xl">
              <Input
                size="2xl"
                type={showPassword.password ? "text" : "password"}
                id="password"
                name="password"
                value={passwordFormData.password}
                onChange={(e) => handleFormChange("password", e.target.value)}
                placeholder={t("auth.common.password.placeholder")}
                autoComplete={mode === EAuthModes.SIGN_IN ? "current-password" : "new-password"}
              />
              <button
                type="button"
                onClick={() => handleShowPassword("password")}
                className="grid size-5 place-items-center"
                aria-label={t(
                  showPassword.password
                    ? "aria_labels.auth_forms.hide_password"
                    : "aria_labels.auth_forms.show_password"
                )}
              >
                {showPassword.password ? (
                  <HideOutline className="size-5 text-placeholder" />
                ) : (
                  <ShowOutline className="size-5 text-placeholder" />
                )}
              </button>
            </InputGroup>
          </Field>
          {fieldErrors.password && <p className="px-0.5 text-11 text-danger-primary">{t(fieldErrors.password)}</p>}
          {mode === EAuthModes.SIGN_UP && (
            <PasswordStrengthIndicator password={password} labels={passwordStrengthLabels} />
          )}
        </div>

        {mode === EAuthModes.SIGN_UP && (
          <div className="space-y-1">
            <label htmlFor="confirm-password" className="text-13 font-medium text-tertiary">
              {t("auth.common.password.confirm_password.label")}
            </label>
            <InputGroup size="2xl">
              <Input
                size="2xl"
                type={showPassword.retypePassword ? "text" : "password"}
                id="confirm-password"
                name="confirm_password"
                value={passwordFormData.confirm_password}
                onChange={(e) => handleFormChange("confirm_password", e.target.value)}
                placeholder={t("auth.common.password.confirm_password.placeholder")}
                onFocus={() => setIsRetryPasswordInputFocused(true)}
                onBlur={() => setIsRetryPasswordInputFocused(false)}
                autoComplete="new-password"
              />
              <button
                type="button"
                className="grid size-5 place-items-center"
                aria-label={t(
                  showPassword.retypePassword
                    ? "aria_labels.auth_forms.hide_password"
                    : "aria_labels.auth_forms.show_password"
                )}
                onClick={() => handleShowPassword("retypePassword")}
              >
                {showPassword.retypePassword ? (
                  <HideOutline className="size-5 text-placeholder" />
                ) : (
                  <ShowOutline className="size-5 text-placeholder" />
                )}
              </button>
            </InputGroup>
            {!!confirmPassword && password !== confirmPassword && renderPasswordMatchError && (
              <span className="text-13 text-danger-primary">{t("auth.common.password.errors.match")}</span>
            )}
          </div>
        )}

        <Button type="submit" variant="primary" className="w-full" size="xl" disabled={isButtonDisabled}>
          {isSubmitting ? (
            <Spinner height="20px" width="20px" />
          ) : mode === EAuthModes.SIGN_IN ? (
            t("common.go_to_workspace")
          ) : (
            t("auth.sign_up.submit")
          )}
        </Button>
      </form>
    </>
  );
});
