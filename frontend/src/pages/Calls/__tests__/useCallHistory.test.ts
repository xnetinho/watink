import { describe, expect, it } from "vitest";
import { buildHistoryParams } from "../hooks/useCallHistory";

describe("buildHistoryParams", () => {
  it("só envia os filtros preenchidos", () => {
    expect(buildHistoryParams({ status: "", direction: "", page: 1, pageSize: 20 })).toEqual({ page: 1, pageSize: 20 });
    expect(buildHistoryParams({ status: "missed", direction: "", page: 2, pageSize: 20 })).toEqual({ page: 2, pageSize: 20, status: "missed" });
    expect(buildHistoryParams({ status: "ended", direction: "incoming", page: 1, pageSize: 50 })).toEqual({
      page: 1, pageSize: 50, status: "ended", direction: "incoming",
    });
  });
});
