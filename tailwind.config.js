/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./internal/**/*.{html,templ}"],
  darkMode: "class",
  theme: {
    extend: {},
  },
  plugins: [require("@tailwindcss/forms")],
};
