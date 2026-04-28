import { defineStore } from "pinia";
import { ref, watch } from "vue";

const STORAGE_KEY = "surat-kec-theme";

export const useThemeStore = defineStore("theme", () => {
  const dark = ref<boolean>(localStorage.getItem(STORAGE_KEY) === "dark");

  watch(dark, (v) => {
    localStorage.setItem(STORAGE_KEY, v ? "dark" : "light");
  });

  function toggle() {
    dark.value = !dark.value;
  }

  return { dark, toggle };
});
