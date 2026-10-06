import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";

import { invalidateWithBodyMap } from "../../api/invalidate";
import { api } from "../../api/client";
import type { components } from "../../api/schema.gen";
import { ProblemError } from "../heart/api";

export type Task = components["schemas"]["Task"];
export type TaskInput = components["schemas"]["TaskInput"];
export type MergePatch = components["schemas"]["MergePatch"];

const tasksKey = ["tasks"] as const;

/** Lists tasks a page at a time, newest saved first. The page sorts them by due date. */
export function useTasks() {
  return useInfiniteQuery({
    queryKey: tasksKey,
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const { data, error } = await api.GET("/api/v1/tasks", {
        params: { query: pageParam ? { cursor: pageParam } : {} },
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    getNextPageParam: (last) => last.next_cursor,
  });
}

export function useCreateTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: TaskInput) => {
      const { data, error } = await api.POST("/api/v1/tasks", { body: input });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateWithBodyMap(queryClient, tasksKey),
  });
}

export function useUpdateTask() {
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
      const { data, error } = await api.PATCH("/api/v1/tasks/{id}", {
        params: { path: { id }, header: { "If-Match": `"${String(version)}"` } },
        body: patch,
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateWithBodyMap(queryClient, tasksKey),
  });
}

/** Marks a task done, or open again when `done` is false. */
export function useSetTaskDone() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, done }: { id: string; done: boolean }) => {
      const result = done
        ? await api.POST("/api/v1/tasks/{id}/complete", { params: { path: { id } } })
        : await api.POST("/api/v1/tasks/{id}/reopen", { params: { path: { id } } });
      if (result.error) throw new ProblemError(result.error);
      return result.data;
    },
    onSuccess: () => invalidateWithBodyMap(queryClient, tasksKey),
  });
}

export function useDeleteTask() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE("/api/v1/tasks/{id}", { params: { path: { id } } });
      if (error) throw new ProblemError(error);
    },
    onSuccess: () => invalidateWithBodyMap(queryClient, tasksKey),
  });
}
