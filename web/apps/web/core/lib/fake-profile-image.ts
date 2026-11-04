/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ProfileImageView } from "@/hooks/use-profile-image";

/** Presentation-only tests keep initials; actual scoped image reads are exercised with a real RootStore. */
export const useProfileImage = (): ProfileImageView => ({
  url: undefined,
  loading: false,
  error: undefined,
  retry: async () => {},
  onError: () => {},
  onLoad: () => {},
});
