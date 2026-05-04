<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { useRouter } from "vue-router";
import {
  NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NText, NCard,
  NList, NListItem, NThing, NTag, NEmpty, NSpin, NCheckbox,
  NDescriptions, NDescriptionsItem, NPopconfirm, useMessage,
} from "naive-ui";
import { useAuthStore } from "@/stores/auth";
import { useThemeStore } from "@/stores/theme";
import {
  reconciliationApi,
  type ReconciliationGroup,
  type ReconciliationDetail,
  type SuratDetail,
} from "@/api/surat";
import NotificationBell from "@/components/NotificationBell.vue";
import PendingSyncIndicator from "@/components/PendingSyncIndicator.vue";

const router = useRouter();
const auth = useAuthStore();
const themeStore = useThemeStore();
const message = useMessage();

const groups = ref<ReconciliationGroup[]>([]);
const loading = ref(false);
const includeResolved = ref(false);

// Detail merge state
const selectedDetail = ref<ReconciliationDetail | null>(null);
const detailLoading = ref(false);
const selectedCanonicalID = ref<string>("");
const submitting = ref(false);

async function fetchList() {
  loading.value = true;
  try {
    const resp = await reconciliationApi.list(includeResolved.value);
    groups.value = resp.items;
  } catch (e) {
    message.error("Gagal memuat antrian rekonsiliasi");
    console.error(e);
  } finally {
    loading.value = false;
  }
}

async function openDetail(groupID: string) {
  detailLoading.value = true;
  try {
    selectedDetail.value = await reconciliationApi.get(groupID);
    selectedCanonicalID.value = selectedDetail.value.surats[0]?.id ?? "";
  } catch (e) {
    message.error("Gagal memuat detail group");
    console.error(e);
  } finally {
    detailLoading.value = false;
  }
}

function closeDetail() {
  selectedDetail.value = null;
  selectedCanonicalID.value = "";
}

async function submitMerge() {
  if (!selectedDetail.value || !selectedCanonicalID.value) return;
  submitting.value = true;
  try {
    await reconciliationApi.merge(selectedDetail.value.group_id, selectedCanonicalID.value);
    message.success("Surat di-merge");
    closeDetail();
    await fetchList();
  } catch (e) {
    message.error("Gagal merge");
    console.error(e);
  } finally {
    submitting.value = false;
  }
}

async function submitKeepBoth() {
  if (!selectedDetail.value) return;
  submitting.value = true;
  try {
    await reconciliationApi.keepBoth(selectedDetail.value.group_id);
    message.success("Kedua surat di-tandai bukan duplikat");
    closeDetail();
    await fetchList();
  } catch (e) {
    message.error("Gagal mark keep-both");
    console.error(e);
  } finally {
    submitting.value = false;
  }
}

function logout() {
  auth.logout();
  router.push({ name: "login" });
}

const statusTag: Record<string, "warning" | "success" | "default"> = {
  pending: "warning",
  merged: "success",
  kept_both: "default",
};

const statusLabel: Record<string, string> = {
  pending: "Pending",
  merged: "Sudah di-merge",
  kept_both: "Disimpan kedua",
};

// Field-level diff highlighter — mark "berbeda" untuk row yang values
// antar surats tidak semua sama. Helper untuk visual cue di Descriptions.
function isFieldDiff(field: keyof SuratDetail): boolean {
  if (!selectedDetail.value || selectedDetail.value.surats.length < 2) return false;
  const values = selectedDetail.value.surats.map((s) => JSON.stringify(s[field]));
  return new Set(values).size > 1;
}

const pendingCount = computed(() => groups.value.filter((g) => g.status === "pending").length);

onMounted(fetchList);
</script>

<template>
  <NLayout style="height: 100vh">
    <NLayoutHeader bordered style="padding: 12px 24px">
      <NSpace justify="space-between" align="center">
        <NSpace align="center">
          <NButton text @click="router.push({ name: 'surat-list' })">← Daftar Surat</NButton>
          <NText strong>Antrian Rekonsiliasi</NText>
          <NTag v-if="pendingCount > 0" type="warning" size="small">
            {{ pendingCount }} pending
          </NTag>
        </NSpace>
        <NSpace align="center">
          <PendingSyncIndicator />
          <NotificationBell />
          <NText depth="3">{{ auth.userID }} ({{ auth.roles.join(", ") }})</NText>
          <NButton size="small" @click="themeStore.toggle()">
            {{ themeStore.dark ? "☀" : "☾" }}
          </NButton>
          <NButton size="small" @click="logout">Keluar</NButton>
        </NSpace>
      </NSpace>
    </NLayoutHeader>

    <NLayoutContent style="padding: 24px">
      <!-- Detail merge view (kalau group dipilih) -->
      <div v-if="selectedDetail" data-testid="recon-detail-view">
        <NSpace vertical size="large">
          <NSpace justify="space-between">
            <NText strong>Merge Group: <code>{{ selectedDetail.dedup_key }}</code></NText>
            <NButton size="small" @click="closeDetail">← Kembali ke list</NButton>
          </NSpace>

          <NSpin :show="detailLoading">
            <NSpace vertical size="medium">
              <NText>
                Pilih surat yang akan dijadikan kanonik. Surat lain akan di-soft-delete.
                Kalau ternyata bukan duplikat, klik "Bukan duplikat" untuk pertahankan keduanya.
              </NText>

              <NSpace>
                <NCard
                  v-for="surat in selectedDetail.surats"
                  :key="surat.id"
                  :title="surat.perihal"
                  size="small"
                  :data-testid="`recon-surat-card-${surat.id}`"
                  :style="{
                    minWidth: '320px',
                    border: selectedCanonicalID === surat.id
                      ? '2px solid #2080f0' : '1px solid #ddd',
                  }"
                  @click="selectedCanonicalID = surat.id"
                >
                  <template #header-extra>
                    <NTag
                      v-if="selectedCanonicalID === surat.id"
                      size="tiny"
                      type="info"
                    >
                      Kanonik
                    </NTag>
                  </template>
                  <NDescriptions :column="1" size="small" bordered>
                    <NDescriptionsItem label="Nomor">
                      <span :style="{ color: isFieldDiff('nomor_surat') ? '#d48806' : 'inherit' }">
                        <code>{{ surat.nomor_surat }}</code>
                      </span>
                    </NDescriptionsItem>
                    <NDescriptionsItem label="Tanggal Surat">
                      <span :style="{ color: isFieldDiff('tanggal_surat') ? '#d48806' : 'inherit' }">
                        {{ surat.tanggal_surat }}
                      </span>
                    </NDescriptionsItem>
                    <NDescriptionsItem v-if="surat.tanggal_terima" label="Tanggal Terima">
                      <span :style="{ color: isFieldDiff('tanggal_terima') ? '#d48806' : 'inherit' }">
                        {{ surat.tanggal_terima }}
                      </span>
                    </NDescriptionsItem>
                    <NDescriptionsItem label="Instansi">{{ surat.instansi_nama }}</NDescriptionsItem>
                    <NDescriptionsItem label="Klasifikasi">
                      {{ surat.klasifikasi_kode ?? "—" }}
                    </NDescriptionsItem>
                    <NDescriptionsItem label="Sifat">
                      {{ surat.sifat_kode ?? "—" }}
                    </NDescriptionsItem>
                    <NDescriptionsItem label="Akses">{{ surat.access_level }}</NDescriptionsItem>
                    <NDescriptionsItem label="Lampiran">
                      {{ surat.attachments.length }} file
                    </NDescriptionsItem>
                  </NDescriptions>
                </NCard>
              </NSpace>

              <NSpace v-if="selectedDetail.status === 'pending'" justify="end">
                <NPopconfirm @positive-click="submitKeepBoth">
                  <template #trigger>
                    <NButton :loading="submitting" data-testid="recon-keep-both-btn">
                      Bukan duplikat — pertahankan keduanya
                    </NButton>
                  </template>
                  Yakin? Status group akan di-set "kept_both".
                </NPopconfirm>
                <NButton
                  type="primary"
                  :loading="submitting"
                  :disabled="!selectedCanonicalID"
                  @click="submitMerge"
                  data-testid="recon-merge-btn"
                >
                  Merge — pakai surat kanonik, soft-delete lainnya
                </NButton>
              </NSpace>
              <NSpace v-else>
                <NText depth="3">
                  Group sudah resolved (status: {{ statusLabel[selectedDetail.status] }})
                </NText>
              </NSpace>
            </NSpace>
          </NSpin>
        </NSpace>
      </div>

      <!-- List view -->
      <div v-else>
        <NSpace align="center" style="margin-bottom: 16px">
          <NCheckbox v-model:checked="includeResolved" @update:checked="fetchList">
            Tampilkan group yang sudah resolved
          </NCheckbox>
        </NSpace>

        <NSpin :show="loading">
          <NEmpty
            v-if="!loading && groups.length === 0"
            description="Tidak ada antrian rekonsiliasi"
          />
          <NList v-else>
            <NListItem
              v-for="g in groups"
              :key="g.group_id"
              style="cursor: pointer"
              :data-testid="`recon-group-${g.group_id}`"
              @click="openDetail(g.group_id)"
            >
              <NThing>
                <template #header>
                  <NSpace align="center" :size="6">
                    <NTag size="small" :type="statusTag[g.status] ?? 'default'">
                      {{ statusLabel[g.status] ?? g.status }}
                    </NTag>
                    <NText strong>{{ g.instansi_nama }}</NText>
                    <NText depth="3">·</NText>
                    <code>{{ g.nomor_surat }}</code>
                  </NSpace>
                </template>
                <template #header-extra>
                  <NText depth="3" style="font-size: 12px">
                    {{ g.surat_count }} surat
                  </NText>
                </template>
                <template #description>
                  <NText depth="3" style="font-size: 12px">
                    Tanggal terima: {{ g.tanggal_terima ?? "—" }} ·
                    Created: {{ new Date(g.created_at).toLocaleString("id-ID") }}
                  </NText>
                </template>
              </NThing>
            </NListItem>
          </NList>
        </NSpin>
      </div>
    </NLayoutContent>
  </NLayout>
</template>
