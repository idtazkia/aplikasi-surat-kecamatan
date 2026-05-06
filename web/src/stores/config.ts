import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { loadedConfig, type TenantConfig } from "@/config";

// useConfigStore — reactive accessor ke tenant config yang sudah di-fetch
// di main.ts sebelum mount. Component pakai store ini, bukan langsung
// loadedConfig(), supaya theme switching atau hot-reload runtime ke depan
// lebih mudah.
export const useConfigStore = defineStore("config", () => {
  // Initialize dari sync getter yang throw kalau loadConfig() belum dijalankan.
  const config = ref<TenantConfig>(loadedConfig());

  const themeOverrides = computed(() => ({
    common: {
      primaryColor: config.value.branding.primary,
      primaryColorHover: config.value.branding.primaryHover,
      primaryColorPressed: config.value.branding.primaryHover,
      primaryColorSuppl: config.value.branding.accent,
    },
  }));

  return { config, themeOverrides };
});
