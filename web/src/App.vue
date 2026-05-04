<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NDialogProvider, NLoadingBarProvider, idID, dateIdID } from "naive-ui";
import { darkTheme } from "naive-ui";
import { computed } from "vue";
import { useThemeStore } from "@/stores/theme";
import { useEduPanelStore } from "@/stores/eduPanel";
import { useOfflineStore } from "@/stores/offline";
import StudentDrawer from "@/components/StudentDrawer.vue";

const themeStore = useThemeStore();
const eduPanel = useEduPanelStore();
const offline = useOfflineStore();
const theme = computed(() => (themeStore.dark ? darkTheme : null));
</script>

<template>
  <NConfigProvider :theme="theme" :locale="idID" :date-locale="dateIdID">
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
