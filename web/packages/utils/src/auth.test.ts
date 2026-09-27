/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it } from "vitest";
import { E_PASSWORD_STRENGTH } from "@nerve/constants";
import { getPasswordCriteria, getPasswordStrength } from "./auth";

// The strength hint follows the server's composition rules (M2 design 3.8, 7.3): 8–128 UTF-16 code units,
// an upper-case letter, a lower-case letter, a digit and a special character. The common-password list is
// the server's alone.

const strong = (length: number) => "Aa1!" + "x".repeat(length - 4);

// The server's special characters (server/internal/modules/identity/domain/password.go, passwordSpecials).
const specials = `!@#$%^&*()-_+=[]{}|;:'",.<>?/`;

describe("getPasswordStrength", () => {
  it("takes 8 to 128 characters", () => {
    expect(getPasswordStrength(strong(8))).toBe(E_PASSWORD_STRENGTH.STRENGTH_VALID);
    expect(getPasswordStrength(strong(128))).toBe(E_PASSWORD_STRENGTH.STRENGTH_VALID);
  });

  it("refuses fewer than 8 or more than 128 characters, as the server does", () => {
    expect(getPasswordStrength(strong(7))).toBe(E_PASSWORD_STRENGTH.LENGTH_NOT_VALID);
    expect(getPasswordStrength(strong(129))).toBe(E_PASSWORD_STRENGTH.LENGTH_NOT_VALID);
  });

  it("counts UTF-16 code units: 'Aa1!' and 62 emoji are 128 of them, one character more is too long", () => {
    expect(getPasswordStrength("Aa1!" + "😀".repeat(62))).toBe(E_PASSWORD_STRENGTH.STRENGTH_VALID);
    expect(getPasswordStrength("Aa1!x" + "😀".repeat(62))).toBe(E_PASSWORD_STRENGTH.LENGTH_NOT_VALID);
  });

  it("needs every class of character", () => {
    expect(getPasswordStrength("aa1!xxxx")).toBe(E_PASSWORD_STRENGTH.STRENGTH_NOT_VALID);
    expect(getPasswordStrength("")).toBe(E_PASSWORD_STRENGTH.EMPTY);
  });

  it("takes each of the server's special characters, and nothing else as one", () => {
    for (const special of specials)
      expect(getPasswordStrength(`Aa1${special}xxxx`), special).toBe(E_PASSWORD_STRENGTH.STRENGTH_VALID);
    for (const other of " ~`\\")
      expect(getPasswordStrength(`Aa1${other}xxxx`), other).toBe(E_PASSWORD_STRENGTH.STRENGTH_NOT_VALID);
  });
});

describe("getPasswordCriteria", () => {
  it("shows the length rule as unmet above 128 characters", () => {
    const length = (password: string) => getPasswordCriteria(password).find((c) => c.key === "length");
    expect(length(strong(128))).toMatchObject({ label: "8–128 characters", isValid: true });
    expect(length(strong(129))?.isValid).toBe(false);
    expect(length(strong(7))?.isValid).toBe(false);
  });

  it("shows the special-character rule as met by each of the server's special characters alone", () => {
    const special = (password: string) => getPasswordCriteria(password).find((c) => c.key === "special")?.isValid;
    for (const character of specials) expect(special(`a${character}`), character).toBe(true);
    for (const other of " ~`\\") expect(special(`a${other}`), other).toBe(false);
  });
});
