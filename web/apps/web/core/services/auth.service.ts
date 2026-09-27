/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { AuthTokens, LoginRequest, RegisterRequest } from "@nerve/api-client";
import { unwrap } from "@/lib/api-error";
import { publicClient } from "@/lib/auth/api-client";

// Sign-in and sign-up; the token manager refreshes and signs out itself (M2 design 7.5).
export class AuthService {
  async register(body: RegisterRequest): Promise<AuthTokens> {
    return unwrap(await publicClient.POST("/api/v0/auth/register", { body }));
  }

  async login(body: LoginRequest): Promise<AuthTokens> {
    return unwrap(await publicClient.POST("/api/v0/auth/login", { body }));
  }
}
