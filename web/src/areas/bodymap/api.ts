import { useQuery } from "@tanstack/react-query";

import { api } from "../../api/client";
import type { components } from "../../api/schema.gen";
import { ProblemError } from "../heart/api";

export type BodyArea = components["schemas"]["BodyArea"];
export type BodyMap = components["schemas"]["BodyMap"];

/** The whole body map. Loaded once; mutations elsewhere refresh it. */
export function useBodyMap() {
  return useQuery({
    queryKey: ["bodymap"],
    queryFn: async (): Promise<BodyMap> => {
      const { data, error } = await api.GET("/api/v1/bodymap");
      if (error) throw new ProblemError(error);
      return data;
    },
  });
}
