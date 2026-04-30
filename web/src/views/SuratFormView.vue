<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  NLayout, NLayoutHeader, NLayoutContent,
  NSpace, NButton, NText, NCard, NForm, NFormItem,
  NInput, NSelect, NDatePicker, NUpload, NTag, NDivider,
  type FormInst, type UploadFileInfo, type SelectOption,
  useMessage,
} from "naive-ui";
import {
  suratApi, direktoriApi,
  type CreateSuratPayload, type AddReferencePayload,
  type LookupItem, type InstansiItem,
} from "@/api/surat";
import { useAuthStore } from "@/stores/auth";

const route = useRoute();
const router = useRouter();
const message = useMessage();
const auth = useAuthStore();

const isEditMode = computed(() => !!route.params.id);
const suratID = ref<string>("");

const formRef = ref<FormInst | null>(null);
const submitting = ref(false);

// Form state
const form = ref<CreateSuratPayload>({
  jenis: "masuk",
  nomor_surat: "",
  perihal: "",
  tanggal_surat: "",
  tanggal_terima: "",
  instansi_id: "",
  klasifikasi_id: undefined,
  sifat_id: undefined,
  access_level: "public",
});
const tanggalSuratPicker = ref<number | null>(null);
const tanggalTerimaPicker = ref<number | null>(null);

// Lookups
const klasifikasi = ref<LookupItem[]>([]);
const sifat = ref<LookupItem[]>([]);
const klasifikasiOptions = computed(() =>
  klasifikasi.value.map((k) => ({ label: `${k.kode} — ${k.nama}`, value: k.id })),
);
const sifatOptions = computed(() =>
  sifat.value.map((s) => ({ label: s.nama, value: s.id })),
);
const jenisOptions = [
  { label: "Surat Masuk", value: "masuk" },
  { label: "Surat Keluar", value: "keluar" },
];
const accessOptions = computed(() => {
  const opts = [
    { label: "Public", value: "public" },
    { label: "Restricted", value: "restricted" },
  ];
  if (auth.hasRole("camat") || auth.hasRole("admin")) {
    opts.push({ label: "Secret (camat/admin)", value: "secret" });
  }
  return opts;
});

// Instansi autocomplete
const instansiQuery = ref("");
const instansiResults = ref<InstansiItem[]>([]);
const instansiOptions = computed<SelectOption[]>(() =>
  instansiResults.value.map((i) => ({
    label: i.aliases.length > 0 ? `${i.nama_kanonik} (${i.aliases.join(", ")})` : i.nama_kanonik,
    value: i.id,
  })),
);
const selectedInstansiNama = ref("");

let instansiSearchTimeout: ReturnType<typeof setTimeout> | null = null;
function searchInstansi(q: string) {
  if (instansiSearchTimeout) clearTimeout(instansiSearchTimeout);
  instansiSearchTimeout = setTimeout(async () => {
    try {
      const resp = await direktoriApi.searchInstansi(q, 20);
      instansiResults.value = resp.items;
    } catch (e) {
      console.error(e);
    }
  }, 200);
}
watch(instansiQuery, (v) => searchInstansi(v));

// Quick-add instansi modal
const newInstansiNama = ref("");
const showAddInstansiForm = ref(false);
async function quickAddInstansi() {
  if (!newInstansiNama.value.trim()) {
    message.warning("Nama instansi wajib diisi");
    return;
  }
  try {
    const resp = await direktoriApi.createInstansi({ nama_kanonik: newInstansiNama.value.trim() });
    message.success("Instansi ditambahkan");
    form.value.instansi_id = resp.id;
    selectedInstansiNama.value = newInstansiNama.value.trim();
    instansiResults.value = [{ id: resp.id, nama_kanonik: newInstansiNama.value.trim(), aliases: [] }];
    showAddInstansiForm.value = false;
    newInstansiNama.value = "";
  } catch (e) {
    message.error("Gagal menambahkan instansi");
    console.error(e);
  }
}

// Attachment files (untuk create flow — di-upload setelah create surat sukses)
const primaryFile = ref<File | null>(null);
const lampiranFiles = ref<File[]>([]);

function onPrimaryFileChange(opt: { fileList: UploadFileInfo[] }) {
  primaryFile.value = opt.fileList[0]?.file ?? null;
}
function onLampiranFilesChange(opt: { fileList: UploadFileInfo[] }) {
  lampiranFiles.value = opt.fileList.map((f) => f.file).filter((f): f is File => f != null);
}

// References (untuk create — di-add setelah create surat sukses)
const newReferences = ref<AddReferencePayload[]>([]);
const newRefRelationship = ref<AddReferencePayload["relationship"]>("balasan");
const newRefMode = ref<"internal" | "external">("internal");
const newRefSearchQuery = ref("");
const newRefResults = ref<{ id: string; nomor_surat: string; perihal: string }[]>([]);
const newRefSelected = ref<{ id: string; nomor_surat: string; perihal: string } | null>(null);
const newRefExternal = ref("");
const newRefNote = ref("");

const relationshipOptions = [
  { label: "Membalas", value: "balasan" },
  { label: "Lanjutan dari", value: "lanjutan" },
  { label: "Hasil disposisi atas", value: "disposisi_hasil" },
  { label: "Revisi atas", value: "revisi" },
  { label: "Terkait", value: "terkait" },
];

let refSearchTimeout: ReturnType<typeof setTimeout> | null = null;
function searchSuratForRef(q: string) {
  if (refSearchTimeout) clearTimeout(refSearchTimeout);
  refSearchTimeout = setTimeout(async () => {
    if (!q.trim()) {
      newRefResults.value = [];
      return;
    }
    try {
      const resp = await suratApi.list({ search: q, limit: 10 });
      newRefResults.value = resp.items.map((it) => ({
        id: it.id, nomor_surat: it.nomor_surat, perihal: it.perihal,
      }));
    } catch (e) {
      console.error(e);
    }
  }, 200);
}
watch(newRefSearchQuery, (v) => searchSuratForRef(v));

function addReferenceLocal() {
  if (newRefMode.value === "internal") {
    if (!newRefSelected.value) {
      message.warning("Pilih surat target");
      return;
    }
    newReferences.value.push({
      to_surat_id: newRefSelected.value.id,
      relationship: newRefRelationship.value,
      note: newRefNote.value.trim() || undefined,
    });
  } else {
    if (!newRefExternal.value.trim()) {
      message.warning("External reference text wajib diisi");
      return;
    }
    newReferences.value.push({
      external_ref: newRefExternal.value.trim(),
      relationship: newRefRelationship.value,
      note: newRefNote.value.trim() || undefined,
    });
  }
  newRefSelected.value = null;
  newRefSearchQuery.value = "";
  newRefExternal.value = "";
  newRefNote.value = "";
}
function removeReferenceLocal(idx: number) {
  newReferences.value.splice(idx, 1);
}

// Sync date picker → form string
watch(tanggalSuratPicker, (v) => {
  form.value.tanggal_surat = v ? new Date(v).toISOString().slice(0, 10) : "";
});
watch(tanggalTerimaPicker, (v) => {
  form.value.tanggal_terima = v ? new Date(v).toISOString().slice(0, 10) : "";
});

// Load detail untuk edit mode
async function loadForEdit(id: string) {
  try {
    const detail = await suratApi.get(id);
    form.value = {
      jenis: detail.jenis,
      nomor_surat: detail.nomor_surat,
      perihal: detail.perihal,
      tanggal_surat: detail.tanggal_surat,
      tanggal_terima: detail.tanggal_terima ?? "",
      instansi_id: detail.instansi_id,
      klasifikasi_id: detail.klasifikasi_kode ? klasifikasiByKode(detail.klasifikasi_kode)?.id : undefined,
      sifat_id: detail.sifat_kode ? sifatByKode(detail.sifat_kode)?.id : undefined,
      access_level: detail.access_level,
    };
    tanggalSuratPicker.value = new Date(detail.tanggal_surat).getTime();
    if (detail.tanggal_terima) {
      tanggalTerimaPicker.value = new Date(detail.tanggal_terima).getTime();
    }
    selectedInstansiNama.value = detail.instansi_nama;
    instansiResults.value = [{
      id: detail.instansi_id, nama_kanonik: detail.instansi_nama, aliases: [],
    }];
  } catch (e) {
    message.error("Gagal memuat data surat");
    console.error(e);
    router.push({ name: "surat-list" });
  }
}

function klasifikasiByKode(kode: string): LookupItem | undefined {
  return klasifikasi.value.find((k) => k.kode === kode);
}
function sifatByKode(kode: string): LookupItem | undefined {
  return sifat.value.find((s) => s.kode === kode);
}

async function loadLookups() {
  try {
    const [k, s] = await Promise.all([
      direktoriApi.listKlasifikasi(),
      direktoriApi.listSifat(),
    ]);
    klasifikasi.value = k.items;
    sifat.value = s.items;
  } catch (e) {
    console.error(e);
  }
}

onMounted(async () => {
  await loadLookups();
  if (route.params.id) {
    suratID.value = route.params.id as string;
    await loadForEdit(suratID.value);
  }
});

async function handleSubmit() {
  // Validation
  if (!form.value.nomor_surat) return message.warning("Nomor surat wajib diisi");
  if (!form.value.perihal) return message.warning("Perihal wajib diisi");
  if (!form.value.tanggal_surat) return message.warning("Tanggal surat wajib diisi");
  if (!form.value.instansi_id) return message.warning("Pilih instansi");
  if (form.value.jenis === "masuk" && !form.value.tanggal_terima) {
    return message.warning("Tanggal terima wajib untuk surat masuk");
  }

  submitting.value = true;
  try {
    let id = suratID.value;
    if (isEditMode.value) {
      await suratApi.update(id, {
        nomor_surat: form.value.nomor_surat,
        perihal: form.value.perihal,
        tanggal_surat: form.value.tanggal_surat,
        tanggal_terima: form.value.jenis === "masuk" ? form.value.tanggal_terima : undefined,
        instansi_id: form.value.instansi_id,
        klasifikasi_id: form.value.klasifikasi_id || undefined,
        sifat_id: form.value.sifat_id || undefined,
        access_level: form.value.access_level,
      });
      message.success("Surat di-update");
    } else {
      const payload: CreateSuratPayload = {
        jenis: form.value.jenis,
        nomor_surat: form.value.nomor_surat,
        perihal: form.value.perihal,
        tanggal_surat: form.value.tanggal_surat,
        instansi_id: form.value.instansi_id,
        klasifikasi_id: form.value.klasifikasi_id || undefined,
        sifat_id: form.value.sifat_id || undefined,
        access_level: form.value.access_level,
      };
      if (form.value.jenis === "masuk") {
        payload.tanggal_terima = form.value.tanggal_terima;
      }
      const resp = await suratApi.create(payload);
      id = resp.id;
      message.success("Surat dibuat");

      // Upload attachments
      const filesToUpload: { file: File; role: "primary" | "lampiran" }[] = [];
      if (primaryFile.value) {
        filesToUpload.push({ file: primaryFile.value, role: "primary" });
      }
      for (const f of lampiranFiles.value) {
        filesToUpload.push({ file: f, role: "lampiran" });
      }
      if (filesToUpload.length > 0) {
        try {
          await suratApi.uploadAttachments(id, filesToUpload);
          message.success(`${filesToUpload.length} lampiran ter-upload`);
        } catch (e) {
          message.error("Lampiran gagal ter-upload — silakan tambah dari halaman detail");
          console.error(e);
        }
      }

      // Add references
      for (const ref of newReferences.value) {
        try {
          await suratApi.addReference(id, ref);
        } catch (e) {
          message.error("Gagal menambah reference");
          console.error(e);
        }
      }
    }

    router.push({ name: "surat-detail", params: { id } });
  } catch (e: unknown) {
    const status = (e as { status?: number }).status;
    if (status === 409) {
      message.error("Nomor surat sudah dipakai");
    } else if (status === 403) {
      message.error("Akses ditolak (mungkin set access_level=secret butuh role camat/admin)");
    } else {
      message.error("Gagal menyimpan surat");
    }
    console.error(e);
  } finally {
    submitting.value = false;
  }
}

function cancel() {
  if (isEditMode.value) {
    router.push({ name: "surat-detail", params: { id: suratID.value } });
  } else {
    router.push({ name: "surat-list" });
  }
}
</script>

<template>
  <NLayout style="height: 100vh">
    <NLayoutHeader bordered style="padding: 12px 24px">
      <NSpace align="center">
        <NButton text @click="cancel">← Batal</NButton>
        <NText strong>{{ isEditMode ? "Edit Surat" : "Surat Baru" }}</NText>
      </NSpace>
    </NLayoutHeader>

    <NLayoutContent style="padding: 24px; max-width: 900px; margin: 0 auto;">
      <NCard>
        <NForm ref="formRef" :model="form" label-placement="top">
          <NFormItem label="Jenis Surat" required>
            <NSelect
              v-model:value="form.jenis"
              :options="jenisOptions"
              :disabled="isEditMode"
              :placeholder="'Pilih jenis'"
            />
          </NFormItem>

          <NFormItem label="Nomor Surat" required>
            <NInput v-model:value="form.nomor_surat" placeholder="045/123/IV/2026" />
          </NFormItem>

          <NFormItem label="Perihal" required>
            <NInput
              v-model:value="form.perihal"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 4 }"
              placeholder="Subject surat"
            />
          </NFormItem>

          <NFormItem label="Tanggal Surat" required>
            <NDatePicker v-model:value="tanggalSuratPicker" type="date" clearable />
          </NFormItem>

          <NFormItem v-if="form.jenis === 'masuk'" label="Tanggal Terima" required>
            <NDatePicker v-model:value="tanggalTerimaPicker" type="date" clearable />
          </NFormItem>

          <NFormItem :label="form.jenis === 'masuk' ? 'Pengirim' : 'Tujuan'" required>
            <NSpace vertical style="width: 100%">
              <div data-testid="instansi-field">
                <NSelect
                  v-model:value="form.instansi_id"
                  :options="instansiOptions"
                  placeholder="Cari instansi..."
                  filterable
                  remote
                  :show-search="true"
                  clearable
                  @search="(q: string) => instansiQuery = q"
                />
              </div>
              <NSpace v-if="!showAddInstansiForm">
                <NButton size="tiny" tertiary @click="showAddInstansiForm = true" data-testid="show-add-instansi">
                  + Instansi baru
                </NButton>
              </NSpace>
              <NSpace v-else align="center">
                <NInput
                  v-model:value="newInstansiNama"
                  placeholder="Nama instansi baru"
                  style="width: 280px"
                  data-testid="new-instansi-input"
                />
                <NButton size="small" type="primary" @click="quickAddInstansi" data-testid="save-new-instansi">Simpan</NButton>
                <NButton size="small" @click="showAddInstansiForm = false; newInstansiNama = ''">Batal</NButton>
              </NSpace>
            </NSpace>
          </NFormItem>

          <NFormItem label="Klasifikasi">
            <NSelect
              v-model:value="form.klasifikasi_id"
              :options="klasifikasiOptions"
              placeholder="(Opsional)"
              clearable
            />
          </NFormItem>

          <NFormItem label="Sifat">
            <NSelect
              v-model:value="form.sifat_id"
              :options="sifatOptions"
              placeholder="(Opsional)"
              clearable
            />
          </NFormItem>

          <NFormItem label="Akses">
            <NSelect v-model:value="form.access_level" :options="accessOptions" />
          </NFormItem>

          <template v-if="!isEditMode">
            <NDivider>Lampiran</NDivider>
            <NFormItem label="PDF Utama (1 file)">
              <div data-testid="primary-upload">
                <NUpload
                  :max="1"
                  :multiple="false"
                  :default-upload="false"
                  @change="onPrimaryFileChange"
                >
                  <NButton>Pilih PDF Utama</NButton>
                </NUpload>
              </div>
            </NFormItem>

            <NFormItem label="Lampiran Pendukung (multiple)">
              <div data-testid="lampiran-upload">
                <NUpload :multiple="true" :default-upload="false" @change="onLampiranFilesChange">
                  <NButton>Pilih Lampiran</NButton>
                </NUpload>
              </div>
            </NFormItem>

            <NDivider>Riwayat Korespondensi</NDivider>
            <NFormItem label="Tambah referensi ke surat lain (opsional)">
              <NSpace vertical style="width: 100%">
                <NSpace>
                  <NSelect
                    v-model:value="newRefRelationship"
                    :options="relationshipOptions"
                    style="width: 200px"
                  />
                  <NSelect
                    v-model:value="newRefMode"
                    :options="[{label: 'Surat existing', value: 'internal'}, {label: 'External (text)', value: 'external'}]"
                    style="width: 180px"
                  />
                </NSpace>
                <template v-if="newRefMode === 'internal'">
                  <NSelect
                    :options="newRefResults.map(r => ({ label: `${r.nomor_surat} — ${r.perihal}`, value: r.id }))"
                    placeholder="Cari surat target..."
                    filterable
                    remote
                    @search="(q: string) => newRefSearchQuery = q"
                    @update:value="(id: string) => newRefSelected = newRefResults.find(r => r.id === id) ?? null"
                  />
                </template>
                <template v-else>
                  <NInput
                    v-model:value="newRefExternal"
                    placeholder="Mis. Surat Kemendagri No. 045/XX/2024 tanggal 12 Mar 2024"
                  />
                </template>
                <NInput v-model:value="newRefNote" placeholder="Catatan (opsional)" />
                <NButton size="small" tertiary @click="addReferenceLocal">+ Tambah ke daftar</NButton>
              </NSpace>
            </NFormItem>

            <div v-if="newReferences.length > 0">
              <NText strong>Akan ditambahkan:</NText>
              <NSpace vertical>
                <NSpace v-for="(ref, idx) in newReferences" :key="idx" align="center">
                  <NTag size="small">{{ relationshipOptions.find(o => o.value === ref.relationship)?.label }}</NTag>
                  <span v-if="ref.to_surat_id">
                    <code>{{ newRefResults.find(r => r.id === ref.to_surat_id)?.nomor_surat ?? ref.to_surat_id }}</code>
                  </span>
                  <span v-else>
                    <NTag size="tiny" type="warning">External</NTag>
                    {{ ref.external_ref }}
                  </span>
                  <NButton size="tiny" text @click="removeReferenceLocal(idx)">Hapus</NButton>
                </NSpace>
              </NSpace>
            </div>
          </template>

          <NDivider />
          <NSpace>
            <NButton type="primary" :loading="submitting" @click="handleSubmit">
              {{ isEditMode ? "Simpan Perubahan" : "Buat Surat" }}
            </NButton>
            <NButton @click="cancel">Batal</NButton>
          </NSpace>
        </NForm>
      </NCard>
    </NLayoutContent>
  </NLayout>
</template>
