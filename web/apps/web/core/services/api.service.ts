/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

/* eslint-disable @typescript-eslint/no-explicit-any */
import type { AxiosInstance, AxiosRequestConfig } from "axios";
import { create } from "axios";
import { normalizeAPIRequestURL } from "@nerve/services";

// The domains that do not use nerve's API yet (M3–M8) still call their old addresses through this class, and
// nerve answers them 404 problem; each domain moves to the generated client when it gets its API (M2 design
// 7.2). It sends no token, and never navigates: sending the user to sign in is AuthenticationWrapper's.
export abstract class APIService {
  private axiosInstance: AxiosInstance;

  constructor() {
    this.axiosInstance = create();

    this.setupInterceptors();
  }

  private setupInterceptors() {
    this.axiosInstance.interceptors.request.use((config) => {
      try {
        if (config.url) {
          config.url = normalizeAPIRequestURL(config.url);
        }
      } catch (error) {
        // Never block a request because of slash normalization — fall back to the
        // original URL and let the call proceed.
        console.warn("[APIService] Failed to normalize trailing slash:", config.url, error);
      }
      return config;
    });
  }

  get(url: string, params = {}, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.get(url, {
      ...params,
      ...config,
    });
  }

  post(url: string, data = {}, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.post(url, data, config);
  }

  put(url: string, data = {}, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.put(url, data, config);
  }

  patch(url: string, data = {}, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.patch(url, data, config);
  }

  delete(url: string, data?: any, config: AxiosRequestConfig = {}) {
    return this.axiosInstance.delete(url, { data, ...config });
  }

  request(config = {}) {
    return this.axiosInstance(config);
  }
}
