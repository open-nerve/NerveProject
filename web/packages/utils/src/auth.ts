/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { E_PASSWORD_STRENGTH } from "@nerve/constants";

// The server's password lengths, in UTF-16 code units like password.length
// (server/internal/modules/identity/domain/password.go).
const MIN_PASSWORD_LENGTH = 8;
const MAX_PASSWORD_LENGTH = 128;

const isLengthValid = (password: string) =>
  password.length >= MIN_PASSWORD_LENGTH && password.length <= MAX_PASSWORD_LENGTH;

/**
 * Calculate password strength based on various criteria
 */
export const getPasswordStrength = (password: string): E_PASSWORD_STRENGTH => {
  if (!password || password === "" || password.length <= 0) {
    return E_PASSWORD_STRENGTH.EMPTY;
  }

  if (!isLengthValid(password)) {
    return E_PASSWORD_STRENGTH.LENGTH_NOT_VALID;
  }

  // Check all criteria
  const hasUpperCase = /[A-Z]/.test(password);
  const hasLowerCase = /[a-z]/.test(password);
  const hasDigit = /[0-9]/.test(password);
  const hasSpecialChar = /[!@#$%^&*()\-_+=[\]{}|;:'",.<>?/]/.test(password);

  if (hasUpperCase && hasLowerCase && hasDigit && hasSpecialChar) {
    return E_PASSWORD_STRENGTH.STRENGTH_VALID;
  }

  return E_PASSWORD_STRENGTH.STRENGTH_NOT_VALID;
};

type PasswordCriteria = {
  key: string;
  label: string;
  isValid: boolean;
};

/**
 * Get password criteria for validation display
 */
export const getPasswordCriteria = (password: string): PasswordCriteria[] => [
  {
    key: "length",
    label: "8–128 characters",
    isValid: isLengthValid(password),
  },
  {
    key: "uppercase",
    label: "Min 1 upper-case letter",
    isValid: /[A-Z]/.test(password),
  },
  {
    key: "lowercase",
    label: "Min 1 lower-case letter",
    isValid: /[a-z]/.test(password),
  },
  {
    key: "number",
    label: "Min 1 number",
    isValid: /[0-9]/.test(password),
  },
  {
    key: "special",
    label: "Min 1 special character",
    isValid: /[!@#$%^&*()\-_+=[\]{}|;:'",.<>?/]/.test(password),
  },
];
