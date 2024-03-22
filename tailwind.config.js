/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/**/*.{html,templ}"],
  darkMode: "class",
  theme: {
    extend: {},
  },
  daisyui: {
    themes: ["cmyk", "dracula"],
  },
  // plugins: [require("@tailwindcss/forms")],
  plugins: [require("daisyui")],
};
