<script setup lang="ts">
import { onMounted } from "vue";
import { useRouter } from "vue-router";
import { NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NText, NSwitch } from "naive-ui";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore } from "@/stores/theme";
import { useEduPanelStore } from "@/stores/eduPanel";

const auth = useAuthStore();
const themeStore = useThemeStore();
const eduPanel = useEduPanelStore();
const router = useRouter();

function logout() {
  auth.logout();
  router.push({ name: "login" });
}

onMounted(() => {
  // Auto-aktifkan student mode untuk user dengan role 'student'
  if (auth.hasRole("student")) {
    eduPanel.enabled = true;
    eduPanel.loadLinks();
  }
});
</script>

<template>
  <NLayout>
    <NLayoutHeader bordered style="padding: 12px 24px">
      <NSpace justify="space-between" align="center">
        <NText strong>Aplikasi Surat Kecamatan</NText>
        <NSpace align="center">
          <NText depth="3">User: {{ auth.userID }} ({{ auth.roles.join(", ") }})</NText>
          <NSwitch v-model:value="themeStore.dark" size="small">
            <template #checked>Gelap</template>
            <template #unchecked>Terang</template>
          </NSwitch>
          <NButton
            v-if="eduPanel.enabled"
            size="small"
            @click="eduPanel.drawerOpen = !eduPanel.drawerOpen"
          >
            Student Mode
          </NButton>
          <NButton size="small" @click="logout">Keluar</NButton>
        </NSpace>
      </NSpace>
    </NLayoutHeader>
    <NLayoutContent style="padding: 24px">
      <NText>
        Fase 0 placeholder — Fase 1 akan menambahkan list surat, input form, dan detail view.
      </NText>
    </NLayoutContent>
  </NLayout>
</template>
