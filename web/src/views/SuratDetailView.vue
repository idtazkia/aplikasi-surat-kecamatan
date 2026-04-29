<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NText, NCard, NTag,
  NDescriptions, NDescriptionsItem, NList, NListItem, NThing, NSpin, NEmpty, NIcon,
  useMessage,
} from "naive-ui";
import { suratApi, type SuratDetail, type SuratReference } from "@/api/surat";

const route = useRoute();
const router = useRouter();
const message = useMessage();

const detail = ref<SuratDetail | null>(null);
const loading = ref(true);

const sifatTagType: Record<string, "default" | "info" | "warning" | "error"> = {
  biasa: "default",
  segera: "warning",
  penting: "info",
  rahasia: "error",
};

const relationshipLabel: Record<string, string> = {
  balasan: "Membalas",
  lanjutan: "Lanjutan dari",
  disposisi_hasil: "Hasil disposisi atas",
  revisi: "Revisi atas",
  terkait: "Terkait",
};

const relationshipReverseLabel: Record<string, string> = {
  balasan: "Dibalas oleh",
  lanjutan: "Dilanjut oleh",
  disposisi_hasil: "Menghasilkan",
  revisi: "Direvisi oleh",
  terkait: "Terkait dengan",
};

async function fetchDetail() {
  loading.value = true;
  try {
    detail.value = await suratApi.get(route.params.id as string);
  } catch (e: unknown) {
    if (e instanceof Error && (e as { status?: number }).status === 404) {
      message.error("Surat tidak ditemukan");
      router.push({ name: "surat-list" });
    } else if (e instanceof Error && (e as { status?: number }).status === 403) {
      message.error("Akses surat ini ditolak");
      router.push({ name: "surat-list" });
    } else {
      message.error("Gagal memuat detail");
      console.error(e);
    }
  } finally {
    loading.value = false;
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function refLabel(ref: SuratReference, direction: "predecessor" | "successor"): string {
  return direction === "predecessor"
    ? relationshipLabel[ref.relationship] ?? ref.relationship
    : relationshipReverseLabel[ref.relationship] ?? ref.relationship;
}

onMounted(fetchDetail);
</script>

<template>
  <NLayout style="height: 100vh">
    <NLayoutHeader bordered style="padding: 12px 24px">
      <NSpace align="center">
        <NButton text @click="router.push({ name: 'surat-list' })">← Daftar</NButton>
        <NText strong>Detail Surat</NText>
      </NSpace>
    </NLayoutHeader>

    <NLayoutContent style="padding: 24px">
      <NSpin :show="loading">
        <NEmpty v-if="!loading && !detail" description="Detail tidak tersedia" />
        <NSpace v-else-if="detail" vertical size="large">
          <!-- Header card -->
          <NCard :title="detail.perihal">
            <template #header-extra>
              <NSpace>
                <NTag :type="detail.jenis === 'masuk' ? 'success' : 'info'">
                  {{ detail.jenis === "masuk" ? "Surat Masuk" : "Surat Keluar" }}
                </NTag>
                <NTag
                  v-if="detail.sifat_kode"
                  :type="sifatTagType[detail.sifat_kode] ?? 'default'"
                >
                  {{ detail.sifat_kode }}
                </NTag>
              </NSpace>
            </template>
            <NDescriptions :column="2" bordered>
              <NDescriptionsItem label="Nomor Surat">
                <code>{{ detail.nomor_surat }}</code>
              </NDescriptionsItem>
              <NDescriptionsItem label="Tanggal Surat">{{ detail.tanggal_surat }}</NDescriptionsItem>
              <NDescriptionsItem v-if="detail.tanggal_terima" label="Tanggal Terima">
                {{ detail.tanggal_terima }}
              </NDescriptionsItem>
              <NDescriptionsItem label="Instansi">{{ detail.instansi_nama }}</NDescriptionsItem>
              <NDescriptionsItem v-if="detail.klasifikasi_kode" label="Klasifikasi">
                <code>{{ detail.klasifikasi_kode }}</code>
                <span v-if="detail.deskripsi_klasifikasi" style="margin-left: 8px; opacity: 0.7">
                  ({{ detail.deskripsi_klasifikasi }})
                </span>
              </NDescriptionsItem>
              <NDescriptionsItem label="Akses">
                <NTag
                  size="small"
                  :type="detail.access_level === 'public' ? 'default' : detail.access_level === 'restricted' ? 'warning' : 'error'"
                >
                  {{ detail.access_level }}
                </NTag>
              </NDescriptionsItem>
            </NDescriptions>
          </NCard>

          <!-- Lampiran -->
          <NCard title="Lampiran">
            <NEmpty v-if="detail.attachments.length === 0" description="Belum ada lampiran" size="small" />
            <NList v-else>
              <NListItem v-for="att in detail.attachments" :key="att.id">
                <NThing>
                  <template #header>
                    <NIcon size="16" style="vertical-align: middle">📄</NIcon>
                    {{ att.file_name }}
                  </template>
                  <template #header-extra>
                    <NTag size="small" :type="att.role === 'primary' ? 'info' : 'default'">
                      {{ att.role === "primary" ? "Utama" : "Lampiran" }}
                    </NTag>
                  </template>
                  <template #description>
                    {{ formatBytes(att.file_size) }} · {{ att.mime_type }}
                  </template>
                </NThing>
              </NListItem>
            </NList>
          </NCard>

          <!-- Riwayat korespondensi -->
          <NCard title="Riwayat Korespondensi">
            <div v-if="detail.predecessors.length === 0 && detail.successors.length === 0">
              <NEmpty description="Tidak ada surat yang berelasi" size="small" />
            </div>
            <NSpace v-else vertical size="large">
              <div v-if="detail.predecessors.length > 0">
                <NText strong>Predecessor (surat ini merujuk):</NText>
                <NList>
                  <NListItem v-for="ref in detail.predecessors" :key="ref.id">
                    <NThing>
                      <template #header>
                        <NTag size="small">{{ refLabel(ref, "predecessor") }}</NTag>
                        <span style="margin-left: 8px">
                          <RouterLink
                            v-if="ref.to_surat_id"
                            :to="{ name: 'surat-detail', params: { id: ref.to_surat_id } }"
                          >
                            <code>{{ ref.to_nomor_surat }}</code> — {{ ref.to_perihal }}
                          </RouterLink>
                          <span v-else>
                            <NTag size="tiny" type="warning">External</NTag>
                            <em style="margin-left: 4px">{{ ref.external_ref }}</em>
                          </span>
                        </span>
                      </template>
                      <template #description>
                        <em v-if="ref.note">{{ ref.note }}</em>
                      </template>
                    </NThing>
                  </NListItem>
                </NList>
              </div>
              <div v-if="detail.successors.length > 0">
                <NText strong>Successor (surat lain merujuk ini):</NText>
                <NList>
                  <NListItem v-for="ref in detail.successors" :key="ref.id">
                    <NThing>
                      <template #header>
                        <NTag size="small">{{ refLabel(ref, "successor") }}</NTag>
                        <span style="margin-left: 8px">
                          <RouterLink
                            v-if="ref.to_surat_id"
                            :to="{ name: 'surat-detail', params: { id: ref.to_surat_id } }"
                          >
                            <code>{{ ref.to_nomor_surat }}</code> — {{ ref.to_perihal }}
                          </RouterLink>
                        </span>
                      </template>
                      <template #description>
                        <em v-if="ref.note">{{ ref.note }}</em>
                      </template>
                    </NThing>
                  </NListItem>
                </NList>
              </div>
            </NSpace>
          </NCard>
        </NSpace>
      </NSpin>
    </NLayoutContent>
  </NLayout>
</template>
