import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { api } from "../../api/client";
import { invalidateWithBodyMap } from "../../api/invalidate";
import type { components } from "../../api/schema.gen";
import { ProblemError } from "../heart/api";

export type Account = components["schemas"]["Account"];
export type AccountInput = components["schemas"]["AccountInput"];
export type Category = components["schemas"]["Category"];
export type CategoryInput = components["schemas"]["CategoryInput"];
export type Transaction = components["schemas"]["Transaction"];
export type TransactionInput = components["schemas"]["TransactionInput"];
export type BudgetLine = components["schemas"]["BudgetLine"];
export type BudgetMonth = components["schemas"]["BudgetMonth"];
export type Rate = components["schemas"]["Rate"];
export type MoneySettings = components["schemas"]["MoneySettings"];
export type MoneySummary = components["schemas"]["MoneySummary"];
export type MergePatch = components["schemas"]["MergePatch"];

const root = ["money"] as const;

export function useCurrencies() {
  return useQuery({
    queryKey: [...root, "currencies"],
    queryFn: async () => {
      const { data, error } = await api.GET("/api/v1/money/currencies");
      if (error) throw new ProblemError(error);
      return data.items;
    },
    staleTime: Infinity,
  });
}

export function useMoneySettings() {
  return useQuery({
    queryKey: [...root, "settings"],
    queryFn: async () => {
      const { data, error } = await api.GET("/api/v1/money/settings");
      if (error) throw new ProblemError(error);
      return data;
    },
  });
}

export function useUpdateMoneySettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ version, patch }: { version: number; patch: MergePatch }) => {
      const { data, error } = await api.PATCH("/api/v1/money/settings", {
        params: { header: { "If-Match": `"${String(version)}"` } },
        body: patch,
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateAllMoney(queryClient),
  });
}

export function useAccounts() {
  return useQuery({
    queryKey: [...root, "accounts"],
    queryFn: async () => {
      const { data, error } = await api.GET("/api/v1/money/accounts");
      if (error) throw new ProblemError(error);
      return data.items;
    },
  });
}

export function useCreateAccount() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: AccountInput) => {
      const { data, error } = await api.POST("/api/v1/money/accounts", { body: input });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateAllMoney(queryClient),
  });
}

export function useUpdateAccount() {
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
      const { data, error } = await api.PATCH("/api/v1/money/accounts/{id}", {
        params: { path: { id }, header: { "If-Match": `"${String(version)}"` } },
        body: patch,
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateAllMoney(queryClient),
  });
}

export function useDeleteAccount() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE("/api/v1/money/accounts/{id}", {
        params: { path: { id } },
      });
      if (error) throw new ProblemError(error);
    },
    onSuccess: () => invalidateAllMoney(queryClient),
  });
}

export function useCategories() {
  return useQuery({
    queryKey: [...root, "categories"],
    queryFn: async () => {
      const { data, error } = await api.GET("/api/v1/money/categories");
      if (error) throw new ProblemError(error);
      return data.items;
    },
  });
}

export function useTransactions() {
  return useInfiniteQuery({
    queryKey: [...root, "transactions"],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) => {
      const { data, error } = await api.GET("/api/v1/money/transactions", {
        params: { query: pageParam ? { cursor: pageParam } : {} },
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    getNextPageParam: (last) => last.next_cursor,
  });
}

export function useRecordTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: TransactionInput) => {
      const { data, error } = await api.POST("/api/v1/money/transactions", { body: input });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateAllMoney(queryClient),
  });
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      const { error } = await api.DELETE("/api/v1/money/transactions/{id}", {
        params: { path: { id } },
      });
      if (error) throw new ProblemError(error);
    },
    onSuccess: () => invalidateAllMoney(queryClient),
  });
}

export function useBudgets(month: string) {
  return useQuery({
    queryKey: [...root, "budgets", month],
    queryFn: async (): Promise<BudgetMonth> => {
      const { data, error } = await api.GET("/api/v1/money/budgets", {
        params: { query: { month } },
      });
      if (error) throw new ProblemError(error);
      return data;
    },
  });
}

export function useSetBudget(month: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({
      categoryId,
      amountMinor,
    }: {
      categoryId: string;
      amountMinor: number;
    }) => {
      const { data, error } = await api.PUT("/api/v1/money/budgets", {
        body: { category_id: categoryId, month, amount_minor: amountMinor },
      });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateAllMoney(queryClient),
  });
}

export function useRates() {
  return useQuery({
    queryKey: [...root, "rates"],
    queryFn: async () => {
      const { data, error } = await api.GET("/api/v1/money/rates");
      if (error) throw new ProblemError(error);
      return data.items;
    },
  });
}

export function useSetRate() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: Rate) => {
      const { data, error } = await api.PUT("/api/v1/money/rates", { body: input });
      if (error) throw new ProblemError(error);
      return data;
    },
    onSuccess: () => invalidateAllMoney(queryClient),
  });
}

export function useMoneySummary(month: string) {
  return useQuery({
    queryKey: [...root, "summary", month],
    queryFn: async (): Promise<MoneySummary> => {
      const { data, error } = await api.GET("/api/v1/money/summary", {
        params: { query: { month } },
      });
      if (error) throw new ProblemError(error);
      return data;
    },
  });
}

/** Money changes move the balances, totals, budgets and the map's status. */
function invalidateAllMoney(queryClient: ReturnType<typeof useQueryClient>) {
  return invalidateWithBodyMap(queryClient, root);
}
