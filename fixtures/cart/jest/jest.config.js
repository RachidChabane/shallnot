module.exports = {
  testPathIgnorePatterns: ["/node_modules/", "<rootDir>/test/integration/"],
  reporters: [
    "default",
    [
      "jest-junit",
      {
        outputDirectory: "results",
        outputName: "junit.xml",
      },
    ],
  ],
};
