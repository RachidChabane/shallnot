import { describe, it, test } from "vitest";

describe("discount codes [verifies REQ-2~2]", () => {
  it('applies a code [verifies REQ-3~1]', () => {});
  test.each([10, 20])("takes %i%% off the total [verifies REQ-2~2] every time", () => {});
  it(`rounds ${mode} totals [verifies REQ-4~1]`, () => {});
});

// [verifies REQ-5~1]
it("isn't tagged in its title", () => {});
it("has no revision [verifies REQ-6]", () => {});
