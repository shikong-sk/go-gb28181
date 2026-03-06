/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  prefix: 'tw-',
  corePlugins: {
    preflight: false,
  },
  theme: {
    extend: {
      colors: {
        primary: 'var(--el-color-primary)',
      },
    },
  },
  plugins: [],
}