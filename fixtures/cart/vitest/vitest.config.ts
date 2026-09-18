import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    exclude: ["**/node_modules/**", "**/dist/**", "test/integration/**"],
    reporters: ["junit"],
    outputFile: "results/junit.xml",
  },
});
