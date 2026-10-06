import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { invalidateWithBodyMap } from "../../api/invalidate";
import { api } from "../../api/client";
import type { components } from "../../api/schema.gen";
import { ProblemError } from "../heart/api";

export type SelfProfile = components["schemas"]["SelfProfile"];
export type MergePatch = components["schemas"]["MergePatch"];

const profileKey = ["self", "profile"] as const;

export function useSelfProfile() {
  return useQuery({
    queryKey: profileKey,
    queryFn: async () => {
      const { data, error } = await api.GET("/api/v1/self");
      if (error) throw new ProblemError(error);
      return data;
    },
  });
}

export function useUpdateSelfProfile() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ version, patch }: { version: number; patch: MergePatch }) => {
      const { data, error } = await api.PATCH("/api/v1/self", {
        params: { header: { "If-Match": `"${String(version)}"` } },
        body: patch,
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateWithBodyMap(queryClient, profileKey),
  });
}
