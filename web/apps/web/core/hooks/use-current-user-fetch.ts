/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useUser } from "@/hooks/store/user";
import { useSessionSWR } from "@/lib/use-session-swr";
/** Account pages and their images share the session's existing account request. */
export function useCurrentUserFetch(wanted = true) {
  const { fetchCurrentUser } = useUser();
  return useSessionSWR(wanted ? ["CURRENT_USER"] : null, () => fetchCurrentUser(), {
    dedupingInterval: 0,
    revalidateOnReconnect: false,
  });
}
