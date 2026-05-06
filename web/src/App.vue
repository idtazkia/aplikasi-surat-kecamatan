<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NDialogProvider, NLoadingBarProvider, idID, dateIdID } from "naive-ui";
import { darkTheme } from "naive-ui";
import { computed, watch, onMounted } from "vue";
import { useThemeStore } from "@/stores/theme";
import { useEduPanelStore } from "@/stores/eduPanel";
import { useAuthStore } from "@/stores/auth";
import { useOfflineStore } from "@/stores/offline";
import { useConfigStore } from "@/stores/config";
import StudentDrawer from "@/components/StudentDrawer.vue";

const themeStore = useThemeStore();
const eduPanel = useEduPanelStore();
const auth = useAuthStore();
const offline = useOfflineStore();
const configStore = useConfigStore();
const theme = computed(() => (themeStore.dark ? darkTheme : null));

onMounted(() => {
  // Load concept-links.json sekali untuk dereferences concept_id → permalink.
  void eduPanel.loadLinks();
});

// Auto-enable student mode saat user dengan role 'student' login.
// Watch karena auth.roles bisa berubah lewat login event (App mount happens
// at /login dengan empty roles, hydrate setelah login).
watch(
  () => auth.roles,
  (roles) => {
    if (roles.includes("student")) {
      eduPanel.enabled = true;
    }
  },
  { immediate: true },
);

// Auto-pop drawer saat ada payload baru dari API response.
// User bisa close manual; drawer reopen kalau payload berubah lagi.
watch(
  () => eduPanel.lastPayload,
  (val) => {
    if (eduPanel.enabled && val) {
      eduPanel.drawerOpen = true;
    }
  },
);
</script>

<template>
  <NConfigProvider :theme="theme" :theme-overrides="configStore.themeOverrides" :locale="idID" :date-locale="dateIdID">
    <NLoadingBarProvider>
      <NDialogProvider>
        <NMessageProvider>
          <div
            v-if="!offline.online"
            data-testid="offline-banner"
            style="
              padding: 6px 16px;
              background: #fff7e6;
              color: #ad6800;
              border-bottom: 1px solid #ffd591;
              font-size: 13px;
              text-align: center;
            "
          >
            ⚠ Anda offline — data terakhir disinkron {{ offline.lastSyncRelative }}
          </div>
          <RouterView />
          <StudentDrawer
            v-if="eduPanel.enabled"
            :open="eduPanel.drawerOpen"
            @close="eduPanel.drawerOpen = false"
          />
        </NMessageProvider>
      </NDialogProvider>
    </NLoadingBarProvider>
  </NConfigProvider>
</template>
