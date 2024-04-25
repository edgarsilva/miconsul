/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/**/*.{html,templ}"],
  darkMode: "class",
  theme: {
    extend: {},
  },
  safelist: ["outline-2", "outline-4", "lg:text-4xl"],
  daisyui: {
    themes: ["cmyk", "dracula", "night"],
  },
  // plugins: [require("@tailwindcss/forms")],
  plugins: [require("@tailwindcss/typography"), require("daisyui")],
};
