import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { api } from "../../api/client";
import type { components } from "../../api/schema.gen";

export type Contact = components["schemas"]["Contact"];
export type ContactInput = components["schemas"]["ContactInput"];
export type ContactPatch = components["schemas"]["ContactPatch"];
type Problem = components["schemas"]["Problem"];

/** An API error. Keeps the problem details so forms can show field messages. */
export class ProblemError extends Error {
  readonly problem: Problem;

  constructor(problem: Problem) {
    super(problem.detail ?? problem.title);
    this.problem = problem;
  }
}

const contactsKey = ["people", "contacts"] as const;

/** Contact fields that may be empty. Empty ones are left out of requests, or sent as null in patches. */
export const optionalFields = [
  "nickname",
  "email",
  "phone",
  "birthday",
  "how_we_met",
  "notes",
] as const;

/** Lists contacts a page at a time, newest first. */
export function useContacts() {
  return useInfiniteQuery({
    queryKey: contactsKey,
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const { data, error } = await api.GET("/api/v1/people/contacts", {
        params: { query: pageParam ? { cursor: pageParam } : {} },
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    getNextPageParam: (last) => last.next_cursor,
  });
}

/** Adds a contact and refreshes the list. */
export function useCreateContact() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: ContactInput) => {
      const { data, error } = await api.POST("/api/v1/people/contacts", { body: input });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: contactsKey }),
  });
}

/**
 * Updates a contact. The version the form was opened with goes in If-Match, so
 * an edit made elsewhere in the meantime is refused rather than overwritten.
 */
export function useUpdateContact() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      version,
      patch,
    }: {
      id: string;
      version: number;
      patch: ContactPatch;
    }) => {
      const { data, error } = await api.PATCH("/api/v1/people/contacts/{id}", {
        params: { path: { id }, header: { "If-Match": `"${String(version)}"` } },
        body: patch,
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: contactsKey }),
  });
}

/** Moves a contact to the trash. */
export function useTrashContact() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE("/api/v1/people/contacts/{id}", {
        params: { path: { id } },
      });
      if (error) throw new ProblemError(error);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: contactsKey }),
  });
}
