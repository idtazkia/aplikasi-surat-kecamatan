<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import {
  NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NText,
  NCard, NSpin, NGrid, NGridItem, useMessage,
} from "naive-ui";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore } from "@/stores/theme";
import { dashboardApi, type DashboardCamatStats } from "@/api/surat";

const router = useRouter();
const auth = useAuthStore();
const themeStore = useThemeStore();
const message = useMessage();

const stats = ref<DashboardCamatStats | null>(null);
const loading = ref(false);

async function fetch() {
  loading.value = true;
  try {
    stats.value = await dashboardApi.camat();
  } catch (e) {
    message.error("Gagal memuat dashboard");
    console.error(e);
  } finally {
    loading.value = false;
  }
}

function logout() {
  auth.logout();
  router.push({ name: "login" });
}

onMounted(fetch);
</script>

<template>
  <NLayout style="height: 100vh">
    <NLayoutHeader bordered style="padding: 12px 24px">
      <NSpace justify="space-between" align="center">
        <NSpace align="center">
          <NButton text @click="router.push({ name: 'surat-list' })">← Daftar Surat</NButton>
          <NText strong>Dashboard Camat</NText>
        </NSpace>
        <NSpace align="center">
          <NText depth="3">{{ auth.userID }} ({{ auth.roles.join(", ") }})</NText>
          <NButton size="small" @click="themeStore.toggle()">
            {{ themeStore.dark ? "☀" : "☾" }}
          </NButton>
          <NButton size="small" @click="logout">Keluar</NButton>
        </NSpace>
      </NSpace>
    </NLayoutHeader>

    <NLayoutContent style="padding: 24px">
      <NSpin :show="loading">
        <NGrid v-if="stats" :cols="4" :x-gap="16" :y-gap="16">
          <NGridItem>
            <NCard
              hoverable
              style="cursor: pointer"
              data-testid="card-surat-masuk-hari-ini"
              @click="router.push({ name: 'surat-list' })"
            >
              <NText depth="3" style="font-size: 12px">Surat Masuk Hari Ini</NText>
              <div style="font-size: 32px; font-weight: 600; margin-top: 8px">
                {{ stats.surat_masuk_hari_ini }}
              </div>
            </NCard>
          </NGridItem>
          <NGridItem>
            <NCard
              hoverable
              style="cursor: pointer"
              data-testid="card-disposisi-belum-assign"
              @click="router.push({ name: 'surat-list' })"
            >
              <NText depth="3" style="font-size: 12px">Belum Diassign</NText>
              <div style="font-size: 32px; font-weight: 600; margin-top: 8px">
                {{ stats.disposisi_belum_assign }}
              </div>
              <NText depth="3" style="font-size: 11px">surat masuk tanpa disposisi</NText>
            </NCard>
          </NGridItem>
          <NGridItem>
            <NCard
              hoverable
              style="cursor: pointer"
              data-testid="card-disposisi-overdue"
              @click="router.push({ name: 'inbox' })"
            >
              <NText depth="3" style="font-size: 12px">Overdue</NText>
              <div style="font-size: 32px; font-weight: 600; margin-top: 8px; color: #d03050">
                {{ stats.disposisi_overdue }}
              </div>
              <NText depth="3" style="font-size: 11px">disposisi melebihi deadline</NText>
            </NCard>
          </NGridItem>
          <NGridItem>
            <NCard
              hoverable
              style="cursor: pointer"
              data-testid="card-disposisi-mine"
              @click="router.push({ name: 'inbox' })"
            >
              <NText depth="3" style="font-size: 12px">Disposisi Saya</NText>
              <div style="font-size: 32px; font-weight: 600; margin-top: 8px">
                {{ stats.disposisi_assigned_to_me }}
              </div>
              <NText depth="3" style="font-size: 11px">aktif (pending/dikerjakan)</NText>
            </NCard>
          </NGridItem>
        </NGrid>
      </NSpin>
    </NLayoutContent>
  </NLayout>
</template>
