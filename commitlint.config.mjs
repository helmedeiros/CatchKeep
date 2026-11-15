export default {
  extends: ["@commitlint/config-conventional"],
  plugins: [
    {
      rules: {
        "no-ai-attribution": ({ raw }) => [
          !/co-authored-by:.*(claude|anthropic|copilot|openai|gpt)|generated with|noreply@anthropic\.com/i.test(
            raw,
          ),
          "commit message must not contain AI attribution",
        ],
      },
    },
  ],
  rules: {
    "no-ai-attribution": [2, "always"],
  },
};
