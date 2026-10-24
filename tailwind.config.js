/** Tailwind-Konfiguration für index.html und assets/datenschutz.html.
 *  Neu bauen (nur nötig, wenn dort neue Tailwind-Klassen dazukommen):
 *  tailwindcss -c tailwind.config.js -i assets/css/tailwind.src.css -o assets/css/tailwind.css --minify
 *  (Standalone-CLI: https://github.com/tailwindlabs/tailwindcss/releases, Version 3.4.x) */
module.exports = {
  darkMode: 'class',
  content: ['./index.html', './assets/datenschutz.html'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      colors: {
        brand: { 50: '#f0f6ff', 500: '#3b82f6', 600: '#2563eb', 700: '#1d4ed8', accent: '#a855f7', neon: '#06b6d4' },
        dark: { bg: '#090d16', card: '#111726', border: '#1f293d', hover: '#1a2336' },
      },
    },
  },
};
