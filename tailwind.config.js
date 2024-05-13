/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/**/*.{html,templ}"],
  darkMode: "class",
  theme: {
    extend: {},
  },
  safelist: ["outline-2", "outline-4", "lg:text-4xl"],
  daisyui: {
    themes: [
      {
        cmyk: {
          ...require("daisyui/src/theming/themes")["cmyk"],
          "base-200": "#f5f5f5",
          "base-300": "#d4d4d4",
          "base-400": "#737373",
        },
      },
      "dracula",
      "night",
    ],
  },
  // plugins: [require("@tailwindcss/forms")],
  plugins: [require("@tailwindcss/typography"), require("daisyui")],
};
