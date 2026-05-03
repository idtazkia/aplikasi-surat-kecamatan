<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NLayout, NLayoutHeader, NLayoutContent, NSpace, NButton, NText, NCard, NTag,
  NDescriptions, NDescriptionsItem, NList, NListItem, NThing, NSpin, NEmpty, NIcon,
  NPopconfirm, NModal, NSelect, NInput, NUpload, NDatePicker,
  type UploadFileInfo,
  useMessage,
} from "naive-ui";
import {
  suratApi, direktoriApi, disposisiApi,
  type SuratDetail, type SuratReference, type AddReferencePayload, type AddTembusanPayload,
  type Disposisi, type DisposisiStatus, type AssignableUser, type CreateDisposisiPayload,
} from "@/api/surat";
import { useAuthStore } from "@/stores/auth";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const authStore = useAuthStore();

const detail = ref<SuratDetail | null>(null);
const loading = ref(true);
const previewURL = ref<string | null>(null);
const previewName = ref<string>("");
const previewMime = ref<string>("");

function canPreview(mimeType: string): boolean {
  return mimeType === "application/pdf" || mimeType.startsWith("image/");
}

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
    await fetchDisposisi();
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

async function handleDelete() {
  if (!detail.value) return;
  try {
    await suratApi.remove(detail.value.id);
    message.success("Surat dihapus");
    router.push({ name: "surat-list" });
  } catch (e) {
    message.error("Gagal menghapus surat");
    console.error(e);
  }
}

function handleEdit() {
  if (!detail.value) return;
  router.push({ name: "surat-edit", params: { id: detail.value.id } });
}

async function previewAttachment(attID: string, fileName: string, mimeType: string) {
  if (!detail.value) return;
  if (previewURL.value) {
    URL.revokeObjectURL(previewURL.value);
    previewURL.value = null;
  }
  try {
    const blobURL = await suratApi.fetchAttachmentPreviewBlobURL(detail.value.id, attID);
    previewURL.value = blobURL;
    previewName.value = fileName;
    previewMime.value = mimeType;
  } catch (e) {
    message.error("Gagal memuat preview");
    console.error(e);
  }
}

function closePreview() {
  if (previewURL.value) {
    URL.revokeObjectURL(previewURL.value);
    previewURL.value = null;
  }
  previewName.value = "";
  previewMime.value = "";
}

// =============================================================================
// Add reference dialog
// =============================================================================
const showAddRefDialog = ref(false);
const newRefMode = ref<"internal" | "external">("internal");
const newRefRelationship = ref<AddReferencePayload["relationship"]>("balasan");
const newRefSearchQuery = ref("");
const newRefResults = ref<{ id: string; nomor_surat: string; perihal: string }[]>([]);
const newRefSelectedID = ref<string>("");
const newRefExternal = ref("");
const newRefNote = ref("");
const submittingRef = ref(false);

const relationshipOptions = [
  { label: "Membalas", value: "balasan" },
  { label: "Lanjutan dari", value: "lanjutan" },
  { label: "Hasil disposisi atas", value: "disposisi_hasil" },
  { label: "Revisi atas", value: "revisi" },
  { label: "Terkait", value: "terkait" },
];
const refModeOptions = [
  { label: "Surat existing (search)", value: "internal" },
  { label: "External (text bebas)", value: "external" },
];

let refSearchTimeout: ReturnType<typeof setTimeout> | null = null;
watch(newRefSearchQuery, (q) => {
  if (refSearchTimeout) clearTimeout(refSearchTimeout);
  refSearchTimeout = setTimeout(async () => {
    if (!q.trim()) {
      newRefResults.value = [];
      return;
    }
    try {
      const resp = await suratApi.list({ search: q, limit: 10 });
      newRefResults.value = resp.items.map((it) => ({
        id: it.id,
        nomor_surat: it.nomor_surat,
        perihal: it.perihal,
      }));
    } catch (e) {
      console.error(e);
    }
  }, 200);
});

function openAddRefDialog() {
  newRefMode.value = "internal";
  newRefRelationship.value = "balasan";
  newRefSearchQuery.value = "";
  newRefResults.value = [];
  newRefSelectedID.value = "";
  newRefExternal.value = "";
  newRefNote.value = "";
  showAddRefDialog.value = true;
}

async function submitAddReference() {
  if (!detail.value) return;
  if (newRefMode.value === "internal" && !newRefSelectedID.value) {
    return message.warning("Pilih surat target");
  }
  if (newRefMode.value === "external" && !newRefExternal.value.trim()) {
    return message.warning("External reference text wajib diisi");
  }
  submittingRef.value = true;
  try {
    const payload: AddReferencePayload = {
      relationship: newRefRelationship.value,
      note: newRefNote.value.trim() || undefined,
    };
    if (newRefMode.value === "internal") {
      payload.to_surat_id = newRefSelectedID.value;
    } else {
      payload.external_ref = newRefExternal.value.trim();
    }
    await suratApi.addReference(detail.value.id, payload);
    message.success("Referensi ditambahkan");
    showAddRefDialog.value = false;
    await fetchDetail();
  } catch (e) {
    message.error("Gagal menambah referensi");
    console.error(e);
  } finally {
    submittingRef.value = false;
  }
}

async function deleteReference(refID: string) {
  if (!detail.value) return;
  try {
    await suratApi.removeReference(detail.value.id, refID);
    message.success("Referensi dihapus");
    await fetchDetail();
  } catch (e) {
    message.error("Gagal menghapus referensi");
    console.error(e);
  }
}

// =============================================================================
// Add attachment dialog
// =============================================================================
const showAddAttDialog = ref(false);
const newAttRole = ref<"primary" | "lampiran">("lampiran");
const newAttFiles = ref<File[]>([]);
const submittingAtt = ref(false);

function openAddAttDialog() {
  newAttRole.value = "lampiran";
  newAttFiles.value = [];
  showAddAttDialog.value = true;
}
function onAttFilesChange(opt: { fileList: UploadFileInfo[] }) {
  newAttFiles.value = opt.fileList.map((f) => f.file).filter((f): f is File => f != null);
}
async function submitAddAttachment() {
  if (!detail.value) return;
  if (newAttFiles.value.length === 0) {
    return message.warning("Pilih file dulu");
  }
  submittingAtt.value = true;
  try {
    const files = newAttFiles.value.map((file) => ({ file, role: newAttRole.value }));
    await suratApi.uploadAttachments(detail.value.id, files);
    message.success(`${files.length} file ter-upload`);
    showAddAttDialog.value = false;
    await fetchDetail();
  } catch (e) {
    message.error("Gagal upload (cek MIME type / size limit 25MB)");
    console.error(e);
  } finally {
    submittingAtt.value = false;
  }
}

// =============================================================================
// Add tembusan dialog
// =============================================================================
const showAddTembusanDialog = ref(false);
const newTembusanMode = ref<"internal" | "external">("internal");
const newTembusanInstansiQuery = ref("");
const newTembusanInstansiResults = ref<{ label: string; value: string }[]>([]);
const newTembusanInstansiID = ref<string>("");
const newTembusanExternal = ref("");
const submittingTembusan = ref(false);

const tembusanModeOptions = [
  { label: "Instansi (search direktori)", value: "internal" },
  { label: "External (text bebas)", value: "external" },
];

let tembusanSearchTimeout: ReturnType<typeof setTimeout> | null = null;
watch(newTembusanInstansiQuery, (q) => {
  if (tembusanSearchTimeout) clearTimeout(tembusanSearchTimeout);
  tembusanSearchTimeout = setTimeout(async () => {
    if (!q.trim()) {
      newTembusanInstansiResults.value = [];
      return;
    }
    try {
      const resp = await direktoriApi.searchInstansi(q, 10);
      newTembusanInstansiResults.value = resp.items.map((it) => ({
        label: it.nama_kanonik,
        value: it.id,
      }));
    } catch (e) {
      console.error(e);
    }
  }, 200);
});

function openAddTembusanDialog() {
  newTembusanMode.value = "internal";
  newTembusanInstansiQuery.value = "";
  newTembusanInstansiResults.value = [];
  newTembusanInstansiID.value = "";
  newTembusanExternal.value = "";
  showAddTembusanDialog.value = true;
}

async function submitAddTembusan() {
  if (!detail.value) return;
  if (newTembusanMode.value === "internal" && !newTembusanInstansiID.value) {
    return message.warning("Pilih instansi target");
  }
  if (newTembusanMode.value === "external" && !newTembusanExternal.value.trim()) {
    return message.warning("External text wajib diisi");
  }
  submittingTembusan.value = true;
  try {
    const payload: AddTembusanPayload = {};
    if (newTembusanMode.value === "internal") {
      payload.instansi_id = newTembusanInstansiID.value;
    } else {
      payload.external_text = newTembusanExternal.value.trim();
    }
    await suratApi.addTembusan(detail.value.id, payload);
    message.success("Tembusan ditambahkan");
    showAddTembusanDialog.value = false;
    await fetchDetail();
  } catch (e) {
    message.error("Gagal menambah tembusan");
    console.error(e);
  } finally {
    submittingTembusan.value = false;
  }
}

async function deleteTembusan(tembusanID: string) {
  if (!detail.value) return;
  try {
    await suratApi.removeTembusan(detail.value.id, tembusanID);
    message.success("Tembusan dihapus");
    await fetchDetail();
  } catch (e) {
    message.error("Gagal menghapus tembusan");
    console.error(e);
  }
}

// =============================================================================
// Disposisi
// =============================================================================
const disposisiList = ref<Disposisi[]>([]);
const assignableUsers = ref<AssignableUser[]>([]);

const showAddDisposisiDialog = ref(false);
const newDispAssignee = ref<string>("");
const newDispInstruksi = ref("");
const newDispNomor = ref("");
const newDispDeadline = ref<number | null>(null); // unix ms — NDatePicker model
const submittingDisp = ref(false);

const disposisiStatusOptions: { label: string; value: DisposisiStatus }[] = [
  { label: "Pending", value: "pending" },
  { label: "Sedang dikerjakan", value: "in_progress" },
  { label: "Selesai", value: "done" },
  { label: "Dibatalkan", value: "cancelled" },
];

const disposisiStatusTagType: Record<DisposisiStatus, "default" | "warning" | "info" | "success" | "error"> = {
  pending: "warning",
  in_progress: "info",
  done: "success",
  cancelled: "error",
};

function disposisiStatusLabel(s: DisposisiStatus): string {
  const m = disposisiStatusOptions.find((o) => o.value === s);
  return m ? m.label : s;
}

async function fetchDisposisi() {
  if (!detail.value) return;
  try {
    const resp = await disposisiApi.list({ surat_id: detail.value.id });
    disposisiList.value = resp.items;
  } catch (e) {
    console.error(e);
  }
}

async function fetchAssignable() {
  if (assignableUsers.value.length > 0) return; // cache: hanya 1x per session
  try {
    const resp = await disposisiApi.listAssignableUsers();
    assignableUsers.value = resp.items;
  } catch (e) {
    console.error(e);
  }
}

async function openAddDisposisiDialog() {
  newDispAssignee.value = "";
  newDispInstruksi.value = "";
  newDispNomor.value = "";
  newDispDeadline.value = null;
  await fetchAssignable();
  showAddDisposisiDialog.value = true;
}

async function submitAddDisposisi() {
  if (!detail.value) return;
  if (!newDispAssignee.value) return message.warning("Pilih assignee");
  if (!newDispInstruksi.value.trim()) return message.warning("Instruksi wajib diisi");

  submittingDisp.value = true;
  try {
    const payload: CreateDisposisiPayload = {
      surat_id: detail.value.id,
      assigned_to: newDispAssignee.value,
      instruksi: newDispInstruksi.value.trim(),
    };
    if (newDispNomor.value.trim()) payload.nomor_disposisi = newDispNomor.value.trim();
    if (newDispDeadline.value) payload.deadline = new Date(newDispDeadline.value).toISOString();

    await disposisiApi.create(payload);
    message.success("Disposisi dibuat");
    showAddDisposisiDialog.value = false;
    await fetchDisposisi();
  } catch (e) {
    message.error("Gagal membuat disposisi");
    console.error(e);
  } finally {
    submittingDisp.value = false;
  }
}

async function updateDisposisiStatus(d: Disposisi, status: DisposisiStatus) {
  try {
    await disposisiApi.update(d.id, { status });
    message.success(`Status diubah ke ${disposisiStatusLabel(status)}`);
    await fetchDisposisi();
  } catch (e) {
    message.error("Gagal update status");
    console.error(e);
  }
}

function canUpdateDisposisi(d: Disposisi): boolean {
  if (d.assigned_to === authStore.userID) return true;
  if (d.created_by === authStore.userID) return true;
  if (authStore.hasRole("camat") || authStore.hasRole("admin")) return true;
  return false;
}

function canCreateDisposisi(): boolean {
  return authStore.hasRole("staf") || authStore.hasRole("camat") || authStore.hasRole("admin");
}

function formatDeadline(iso: string): string {
  return new Date(iso).toLocaleString("id-ID", {
    year: "numeric", month: "short", day: "numeric",
    hour: "2-digit", minute: "2-digit",
  });
}

function isOverdue(d: Disposisi): boolean {
  if (!d.deadline) return false;
  if (d.status === "done" || d.status === "cancelled") return false;
  return new Date(d.deadline).getTime() < Date.now();
}

function downloadAttachment(attID: string) {
  if (!detail.value) return;
  const url = suratApi.attachmentDownloadURL(detail.value.id, attID);
  // Trigger download dengan auth header — pakai fetch + blob URL untuk include bearer token
  const authRaw = localStorage.getItem("surat-kec-auth");
  let token = "";
  if (authRaw) {
    try {
      token = JSON.parse(authRaw).accessToken ?? "";
    } catch { /* ignore */ }
  }
  fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {} })
    .then((resp) => {
      if (!resp.ok) throw new Error(`Download gagal: ${resp.status}`);
      return resp.blob().then((blob) => ({ blob, dispo: resp.headers.get("Content-Disposition") }));
    })
    .then(({ blob, dispo }) => {
      const blobURL = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = blobURL;
      // Extract filename from Content-Disposition kalau ada
      const m = dispo?.match(/filename="?([^"]+)"?/);
      a.download = m ? m[1] : "lampiran";
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(blobURL);
    })
    .catch((e) => {
      message.error("Gagal mengunduh lampiran");
      console.error(e);
    });
}

onMounted(fetchDetail);
</script>

<template>
  <NLayout style="height: 100vh">
    <NLayoutHeader bordered style="padding: 12px 24px">
      <NSpace justify="space-between" align="center">
        <NSpace align="center">
          <NButton text @click="router.push({ name: 'surat-list' })">← Daftar</NButton>
          <NText strong>Detail Surat</NText>
        </NSpace>
        <NSpace v-if="detail" align="center">
          <NButton size="small" @click="handleEdit">Edit</NButton>
          <NPopconfirm @positive-click="handleDelete">
            <template #trigger>
              <NButton size="small" type="error" tertiary>Hapus</NButton>
            </template>
            Yakin hapus surat ini? (Soft delete — bisa di-restore admin.)
          </NPopconfirm>
        </NSpace>
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

          <!-- Disposisi -->
          <NCard title="Disposisi" data-testid="disposisi-card">
            <template #header-extra>
              <NButton
                v-if="canCreateDisposisi()"
                size="small"
                type="primary"
                tertiary
                @click="openAddDisposisiDialog"
                data-testid="add-disposisi-btn"
              >
                + Buat Disposisi
              </NButton>
            </template>
            <NEmpty v-if="disposisiList.length === 0" description="Belum ada disposisi" size="small" />
            <NList v-else>
              <NListItem v-for="d in disposisiList" :key="d.id" :data-testid="`disposisi-item-${d.id}`">
                <NThing>
                  <template #header>
                    <NSpace align="center" :size="8">
                      <NTag size="small" :type="disposisiStatusTagType[d.status]">
                        {{ disposisiStatusLabel(d.status) }}
                      </NTag>
                      <NTag v-if="isOverdue(d)" size="tiny" type="error">Overdue</NTag>
                      <NText strong>→ {{ d.assignee_name }}</NText>
                    </NSpace>
                  </template>
                  <template #header-extra>
                    <NSpace v-if="canUpdateDisposisi(d)" :size="4">
                      <NButton
                        v-if="d.status === 'pending'"
                        size="tiny"
                        @click="updateDisposisiStatus(d, 'in_progress')"
                        :data-testid="`disposisi-start-${d.id}`"
                      >
                        Mulai
                      </NButton>
                      <NButton
                        v-if="d.status === 'in_progress'"
                        size="tiny"
                        type="success"
                        @click="updateDisposisiStatus(d, 'done')"
                        :data-testid="`disposisi-done-${d.id}`"
                      >
                        Selesai
                      </NButton>
                      <NButton
                        v-if="d.status !== 'done' && d.status !== 'cancelled'"
                        size="tiny"
                        type="error"
                        tertiary
                        @click="updateDisposisiStatus(d, 'cancelled')"
                        :data-testid="`disposisi-cancel-${d.id}`"
                      >
                        Batal
                      </NButton>
                    </NSpace>
                  </template>
                  <template #description>
                    <div>{{ d.instruksi }}</div>
                    <div style="margin-top: 4px; font-size: 12px; opacity: 0.7">
                      Oleh {{ d.creator_name }}
                      <span v-if="d.deadline"> · Deadline: {{ formatDeadline(d.deadline) }}</span>
                      <span v-if="d.completed_at"> · Selesai: {{ formatDeadline(d.completed_at) }}</span>
                    </div>
                  </template>
                </NThing>
              </NListItem>
            </NList>
          </NCard>

          <!-- Lampiran -->
          <NCard title="Lampiran">
            <template #header-extra>
              <NButton size="small" type="primary" tertiary @click="openAddAttDialog" data-testid="add-attachment-btn">
                + Tambah Lampiran
              </NButton>
            </template>
            <NEmpty v-if="detail.attachments.length === 0" description="Belum ada lampiran" size="small" />
            <NList v-else>
              <NListItem v-for="att in detail.attachments" :key="att.id">
                <NThing>
                  <template #header>
                    <NIcon size="16" style="vertical-align: middle">📄</NIcon>
                    {{ att.file_name }}
                  </template>
                  <template #header-extra>
                    <NSpace>
                      <NTag size="small" :type="att.role === 'primary' ? 'info' : 'default'">
                        {{ att.role === "primary" ? "Utama" : "Lampiran" }}
                      </NTag>
                      <NButton
                        v-if="canPreview(att.mime_type)"
                        size="tiny"
                        type="primary"
                        tertiary
                        @click="previewAttachment(att.id, att.file_name, att.mime_type)"
                        data-testid="attachment-preview-btn"
                      >
                        Preview
                      </NButton>
                      <NButton size="tiny" @click="downloadAttachment(att.id)">Unduh</NButton>
                    </NSpace>
                  </template>
                  <template #description>
                    {{ formatBytes(att.file_size) }} · {{ att.mime_type }}
                  </template>
                </NThing>
              </NListItem>
            </NList>
          </NCard>

          <!-- Inline preview iframe -->
          <NCard v-if="previewURL" :title="`Preview: ${previewName}`" data-testid="preview-card">
            <template #header-extra>
              <NButton size="small" @click="closePreview">Tutup</NButton>
            </template>
            <iframe
              :src="previewURL"
              data-testid="preview-iframe"
              :title="previewName"
              style="width: 100%; height: 600px; border: 1px solid #ddd; border-radius: 4px;"
            />
          </NCard>

          <!-- Tembusan -->
          <NCard title="Tembusan" data-testid="tembusan-card">
            <template #header-extra>
              <NButton size="small" type="primary" tertiary @click="openAddTembusanDialog" data-testid="add-tembusan-btn">
                + Tambah Tembusan
              </NButton>
            </template>
            <NEmpty v-if="detail.tembusan.length === 0" description="Belum ada tembusan" size="small" />
            <NList v-else>
              <NListItem v-for="t in detail.tembusan" :key="t.id">
                <NThing>
                  <template #header>
                    <NTag size="small" style="margin-right: 8px">{{ t.urutan }}</NTag>
                    <span v-if="t.instansi_id">{{ t.instansi_nama }}</span>
                    <span v-else>
                      <NTag size="tiny" type="warning">External</NTag>
                      <em style="margin-left: 4px">{{ t.external_text }}</em>
                    </span>
                  </template>
                  <template #header-extra>
                    <NPopconfirm @positive-click="deleteTembusan(t.id)">
                      <template #trigger>
                        <NButton size="tiny" tertiary type="error" data-testid="delete-tembusan-btn">Hapus</NButton>
                      </template>
                      Hapus tembusan ini?
                    </NPopconfirm>
                  </template>
                </NThing>
              </NListItem>
            </NList>
          </NCard>

          <!-- Riwayat korespondensi -->
          <NCard title="Riwayat Korespondensi">
            <template #header-extra>
              <NButton size="small" type="primary" tertiary @click="openAddRefDialog" data-testid="add-reference-btn">
                + Tambah Referensi
              </NButton>
            </template>
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
                      <template #header-extra>
                        <NPopconfirm @positive-click="deleteReference(ref.id)">
                          <template #trigger>
                            <NButton size="tiny" tertiary type="error" data-testid="delete-reference-btn">Hapus</NButton>
                          </template>
                          Hapus referensi ini?
                        </NPopconfirm>
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

    <!-- Add Reference Dialog -->
    <NModal v-model:show="showAddRefDialog" preset="dialog" title="Tambah Referensi" style="width: 500px">
      <NSpace vertical size="medium" style="margin-top: 12px">
        <div>
          <NText strong>Tipe relasi</NText>
          <NSelect v-model:value="newRefRelationship" :options="relationshipOptions" />
        </div>
        <div>
          <NText strong>Sumber</NText>
          <NSelect v-model:value="newRefMode" :options="refModeOptions" />
        </div>
        <div v-if="newRefMode === 'internal'">
          <NText strong>Surat target</NText>
          <NInput v-model:value="newRefSearchQuery" placeholder="Cari berdasarkan perihal..." />
          <div v-if="newRefResults.length > 0" style="margin-top: 8px; max-height: 200px; overflow-y: auto; border: 1px solid #eee; border-radius: 4px;">
            <div
              v-for="r in newRefResults"
              :key="r.id"
              :data-testid="`ref-option-${r.id}`"
              style="padding: 8px; cursor: pointer; border-bottom: 1px solid #f0f0f0;"
              :style="{ backgroundColor: newRefSelectedID === r.id ? '#e6f7ff' : 'transparent' }"
              @click="newRefSelectedID = r.id"
            >
              <code>{{ r.nomor_surat }}</code> — {{ r.perihal }}
            </div>
          </div>
        </div>
        <div v-else>
          <NText strong>Reference text</NText>
          <NInput
            v-model:value="newRefExternal"
            placeholder="Mis. Surat Kemendagri No. 045/XX/2024 tanggal 12 Mar 2024"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
          />
        </div>
        <div>
          <NText strong>Catatan (opsional)</NText>
          <NInput v-model:value="newRefNote" placeholder="Catatan tentang relasi ini" />
        </div>
      </NSpace>
      <template #action>
        <NSpace>
          <NButton @click="showAddRefDialog = false">Batal</NButton>
          <NButton type="primary" :loading="submittingRef" @click="submitAddReference" data-testid="submit-add-reference">
            Tambah
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Add Disposisi Dialog -->
    <NModal v-model:show="showAddDisposisiDialog" preset="dialog" title="Buat Disposisi" style="width: 500px">
      <NSpace vertical size="medium" style="margin-top: 12px">
        <div>
          <NText strong>Assignee</NText>
          <NSelect
            v-model:value="newDispAssignee"
            :options="assignableUsers.map((u) => ({ label: `${u.full_name} (${u.username})`, value: u.id }))"
            placeholder="Pilih user..."
            filterable
            data-testid="disposisi-assignee-select"
          />
        </div>
        <div>
          <NText strong>Instruksi</NText>
          <NInput
            v-model:value="newDispInstruksi"
            type="textarea"
            placeholder="Tindakan yang diharapkan dari assignee"
            :autosize="{ minRows: 3, maxRows: 6 }"
            data-testid="disposisi-instruksi-input"
          />
        </div>
        <div>
          <NText strong>Nomor Disposisi (opsional)</NText>
          <NInput v-model:value="newDispNomor" placeholder="Mis. DISP/045/IV/2026" />
        </div>
        <div>
          <NText strong>Deadline (opsional)</NText>
          <NDatePicker
            v-model:value="newDispDeadline"
            type="datetime"
            clearable
            style="width: 100%"
          />
        </div>
      </NSpace>
      <template #action>
        <NSpace>
          <NButton @click="showAddDisposisiDialog = false">Batal</NButton>
          <NButton
            type="primary"
            :loading="submittingDisp"
            @click="submitAddDisposisi"
            data-testid="submit-add-disposisi"
          >
            Buat
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Add Tembusan Dialog -->
    <NModal v-model:show="showAddTembusanDialog" preset="dialog" title="Tambah Tembusan" style="width: 500px">
      <NSpace vertical size="medium" style="margin-top: 12px">
        <div>
          <NText strong>Tipe</NText>
          <NSelect v-model:value="newTembusanMode" :options="tembusanModeOptions" />
        </div>
        <div v-if="newTembusanMode === 'internal'">
          <NText strong>Instansi</NText>
          <NInput
            v-model:value="newTembusanInstansiQuery"
            placeholder="Cari nama instansi..."
            data-testid="tembusan-instansi-search"
          />
          <div v-if="newTembusanInstansiResults.length > 0" style="margin-top: 8px; max-height: 200px; overflow-y: auto; border: 1px solid #eee; border-radius: 4px;">
            <div
              v-for="r in newTembusanInstansiResults"
              :key="r.value"
              :data-testid="`tembusan-option-${r.value}`"
              style="padding: 8px; cursor: pointer; border-bottom: 1px solid #f0f0f0;"
              :style="{ backgroundColor: newTembusanInstansiID === r.value ? '#e6f7ff' : 'transparent' }"
              @click="newTembusanInstansiID = r.value"
            >
              {{ r.label }}
            </div>
          </div>
        </div>
        <div v-else>
          <NText strong>Tujuan tembusan (text bebas)</NText>
          <NInput
            v-model:value="newTembusanExternal"
            placeholder="Mis. Kepala Bidang XYZ, Pemerhati Lingkungan"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
          />
        </div>
      </NSpace>
      <template #action>
        <NSpace>
          <NButton @click="showAddTembusanDialog = false">Batal</NButton>
          <NButton type="primary" :loading="submittingTembusan" @click="submitAddTembusan" data-testid="submit-add-tembusan">
            Tambah
          </NButton>
        </NSpace>
      </template>
    </NModal>

    <!-- Add Attachment Dialog -->
    <NModal v-model:show="showAddAttDialog" preset="dialog" title="Tambah Lampiran" style="width: 500px">
      <NSpace vertical size="medium" style="margin-top: 12px">
        <div>
          <NText strong>Tipe</NText>
          <NSelect
            v-model:value="newAttRole"
            :options="[{ label: 'PDF Utama', value: 'primary' }, { label: 'Lampiran', value: 'lampiran' }]"
          />
        </div>
        <div data-testid="add-attachment-upload">
          <NUpload :multiple="newAttRole === 'lampiran'" :default-upload="false" @change="onAttFilesChange">
            <NButton>Pilih File</NButton>
          </NUpload>
        </div>
      </NSpace>
      <template #action>
        <NSpace>
          <NButton @click="showAddAttDialog = false">Batal</NButton>
          <NButton type="primary" :loading="submittingAtt" @click="submitAddAttachment" data-testid="submit-add-attachment">
            Upload
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </NLayout>
</template>
