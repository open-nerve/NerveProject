/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { observer } from "mobx-react";
// icons
import { CloseCircleOutline, HideOutline, ShowOutline, WarningCircleOutline } from "@makeplane/propel/icons";
// plane imports
import { Banner } from "@makeplane/propel/components/banner";
import { Field } from "@makeplane/propel/components/field";
import { Input, InputGroup } from "@makeplane/propel/components/input";
import { E_PASSWORD_STRENGTH } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { PasswordStrengthIndicator, Spinner } from "@nerve/ui";
import { checkEmailValidity, getPasswordStrength } from "@nerve/utils";
// helpers
import { EAuthModes } from "@/helpers/authentication.helper";
// services
import { AuthService } from "@/services/auth.service";

type Props = {
  email: string;
  mode: EAuthModes;
  nextPath: string | undefined;
};

type TPasswordFormValues = {
  email: string;
  password: string;
  confirm_password?: string;
};

const defaultValues: TPasswordFormValues = {
  email: "",
  password: "",
};

const authService = new AuthService();

export const AuthPasswordForm = observer(function AuthPasswordForm(props: Props) {
  const { email, mode, nextPath } = props;
  // plane imports
  const { t } = useTranslation();
  // ref
  const formRef = useRef<HTMLFormElement>(null);
  const emailInputRef = useRef<HTMLInputElement>(null);
  // states
  const [csrfPromise, setCsrfPromise] = useState<Promise<{ csrf_token: string }> | undefined>(undefined);
  const [passwordFormData, setPasswordFormData] = useState<TPasswordFormValues>({ ...defaultValues, email });
  const [showPassword, setShowPassword] = useState({
    password: false,
    retypePassword: false,
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isPasswordInputFocused, setIsPasswordInputFocused] = useState(false);
  const [isRetryPasswordInputFocused, setIsRetryPasswordInputFocused] = useState(false);
  const [isBannerMessage, setBannerMessage] = useState(false);
  const [isEmailInputFocused, setIsEmailInputFocused] = useState(false);

  const handleShowPassword = (key: keyof typeof showPassword) =>
    setShowPassword((prev) => ({ ...prev, [key]: !prev[key] }));

  const handleFormChange = (key: keyof TPasswordFormValues, value: string) =>
    setPasswordFormData((prev) => ({ ...prev, [key]: value }));

  useEffect(() => {
    if (csrfPromise === undefined) {
      const promise = authService.requestCSRFToken();
      setCsrfPromise(promise);
    }
  }, [csrfPromise]);

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

  const password = passwordFormData?.password ?? "";
  const confirmPassword = passwordFormData?.confirm_password ?? "";
  const renderPasswordMatchError = !isRetryPasswordInputFocused || confirmPassword.length >= password.length;

  const handleCSRFToken = async () => {
    if (!formRef || !formRef.current) return;
    const token = await csrfPromise;
    if (!token?.csrf_token) return;
    const csrfElement = formRef.current.querySelector("input[name=csrfmiddlewaretoken]");
    csrfElement?.setAttribute("value", token?.csrf_token);
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
      <form
        ref={formRef}
        className="space-y-4"
        method="POST"
        action={`/auth/${mode === EAuthModes.SIGN_IN ? "sign-in" : "sign-up"}/`}
        onSubmit={async (event) => {
          event.preventDefault(); // Prevent form from submitting by default
          await handleCSRFToken();
          const isPasswordValid =
            mode === EAuthModes.SIGN_UP
              ? getPasswordStrength(passwordFormData.password) === E_PASSWORD_STRENGTH.STRENGTH_VALID
              : true;
          if (isPasswordValid) {
            setIsSubmitting(true);
            if (formRef.current) formRef.current.submit(); // Manually submit the form if the condition is met
          } else {
            setBannerMessage(true);
          }
        }}
        onError={() => {
          setIsSubmitting(false);
        }}
      >
        <input type="hidden" name="csrfmiddlewaretoken" />
        {nextPath && <input type="hidden" value={nextPath} name="next_path" />}
        <div className="space-y-1">
          <label htmlFor="email" className="text-13 font-medium text-tertiary">
            {t("auth.common.email.label")}
          </label>
          <Field name="email" invalid={!isEmailInputFocused && isEmailInvalid}>
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
                autoComplete="off"
                autoFocus={email.length === 0}
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
        </div>

        <div className="space-y-1">
          <label htmlFor="password" className="text-13 font-medium text-tertiary">
            {mode === EAuthModes.SIGN_IN ? t("auth.common.password.label") : t("auth.common.password.set_password")}
          </label>
          <InputGroup size="2xl">
            <Input
              size="2xl"
              type={showPassword?.password ? "text" : "password"}
              id="password"
              name="password"
              value={passwordFormData.password}
              onChange={(e) => handleFormChange("password", e.target.value)}
              placeholder={t("auth.common.password.placeholder")}
              onFocus={() => setIsPasswordInputFocused(true)}
              onBlur={() => setIsPasswordInputFocused(false)}
              autoComplete="off"
              autoFocus={email.length > 0}
            />
            <button
              type="button"
              onClick={() => handleShowPassword("password")}
              className="grid size-5 place-items-center"
              aria-label={t(
                showPassword?.password ? "aria_labels.auth_forms.hide_password" : "aria_labels.auth_forms.show_password"
              )}
            >
              {showPassword?.password ? (
                <HideOutline className="size-5 text-placeholder" />
              ) : (
                <ShowOutline className="size-5 text-placeholder" />
              )}
            </button>
          </InputGroup>
          {mode === EAuthModes.SIGN_UP &&
            passwordFormData.password.length > 0 &&
            getPasswordStrength(passwordFormData.password) != E_PASSWORD_STRENGTH.STRENGTH_VALID && (
              <PasswordStrengthIndicator password={passwordFormData.password} isFocused={isPasswordInputFocused} />
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
                type={showPassword?.retypePassword ? "text" : "password"}
                id="confirm-password"
                name="confirm_password"
                value={passwordFormData.confirm_password}
                onChange={(e) => handleFormChange("confirm_password", e.target.value)}
                placeholder={t("auth.common.password.confirm_password.placeholder")}
                onFocus={() => setIsRetryPasswordInputFocused(true)}
                onBlur={() => setIsRetryPasswordInputFocused(false)}
                autoComplete="off"
              />
              <button
                type="button"
                className="grid size-5 place-items-center"
                aria-label={t(
                  showPassword?.retypePassword
                    ? "aria_labels.auth_forms.hide_password"
                    : "aria_labels.auth_forms.show_password"
                )}
                onClick={() => handleShowPassword("retypePassword")}
              >
                {showPassword?.retypePassword ? (
                  <HideOutline className="size-5 text-placeholder" />
                ) : (
                  <ShowOutline className="size-5 text-placeholder" />
                )}
              </button>
            </InputGroup>
            {!!passwordFormData.confirm_password &&
              passwordFormData.password !== passwordFormData.confirm_password &&
              renderPasswordMatchError && (
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
            "Create account"
          )}
        </Button>
      </form>
    </>
  );
});
