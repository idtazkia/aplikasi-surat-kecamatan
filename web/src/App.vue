<script setup lang="ts">
import { NConfigProvider, NMessageProvider, NDialogProvider, NLoadingBarProvider, idID, dateIdID } from "naive-ui";
import { darkTheme } from "naive-ui";
import { computed } from "vue";
import { useThemeStore } from "@/stores/theme";
import { useEduPanelStore } from "@/stores/eduPanel";
import StudentDrawer from "@/components/StudentDrawer.vue";

const themeStore = useThemeStore();
const eduPanel = useEduPanelStore();
const theme = computed(() => (themeStore.dark ? darkTheme : null));
</script>

<template>
  <NConfigProvider :theme="theme" :locale="idID" :date-locale="dateIdID">
    <NLoadingBarProvider>
      <NDialogProvider>
        <NMessageProvider>
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
