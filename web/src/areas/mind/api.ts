import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { invalidateWithBodyMap } from "../../api/invalidate";
import { api } from "../../api/client";
import type { components } from "../../api/schema.gen";
import { ProblemError } from "../heart/api";

export type JournalEntry = components["schemas"]["JournalEntry"];
export type JournalEntryInput = components["schemas"]["JournalEntryInput"];
export type MergePatch = components["schemas"]["MergePatch"];

const entriesKey = ["journal", "entries"] as const;

/** Lists the owner's entries a page at a time, newest saved first. */
export function useJournalEntries() {
  return useInfiniteQuery({
    queryKey: entriesKey,
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const { data, error } = await api.GET("/api/v1/journal/entries", {
        params: { query: pageParam ? { cursor: pageParam } : {} },
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    getNextPageParam: (last) => last.next_cursor,
  });
}

export function useCreateJournalEntry() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: JournalEntryInput) => {
      const { data, error } = await api.POST("/api/v1/journal/entries", { body: input });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateWithBodyMap(queryClient, entriesKey),
  });
}

export function useUpdateJournalEntry() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      version,
      patch,
    }: {
      id: string;
      version: number;
      patch: MergePatch;
    }) => {
      const { data, error } = await api.PATCH("/api/v1/journal/entries/{id}", {
        params: { path: { id }, header: { "If-Match": `"${String(version)}"` } },
        body: patch,
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateWithBodyMap(queryClient, entriesKey),
  });
}

export function useDeleteJournalEntry() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE("/api/v1/journal/entries/{id}", {
        params: { path: { id } },
      });
      if (error) throw new ProblemError(error);
    },
    onSuccess: () => invalidateWithBodyMap(queryClient, entriesKey),
  });
}
