import type { QueryClient, QueryKey } from "@tanstack/react-query";

/**
 * Refreshes a resource's list and the body map. Every mutation that changes an
 * area's data calls this, since the map shows each area's status (CLAUDE.md).
 */
export async function invalidateWithBodyMap(
  queryClient: QueryClient,
  key: QueryKey,
): Promise<void> {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: key }),
    queryClient.invalidateQueries({ queryKey: ["bodymap"] }),
  ]);
}
