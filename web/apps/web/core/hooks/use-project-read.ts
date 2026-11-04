/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useProject } from "@/hooks/store/use-project";
import { useSessionSWR } from "@/lib/use-session-swr";
/** Read only the project; the page's access decision owns any additional member-only queries. */
export function useProjectRead(projectID: string | undefined) {
  const { fetchProject } = useProject();
  return useSessionSWR(projectID ? ["PROJECT", projectID] : null, (id) => fetchProject(id), {
    revalidateOnReconnect: false,
  });
}
