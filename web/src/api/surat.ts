import { apiClient } from "./client";

export interface SuratListItem {
  id: string;
  jenis: "masuk" | "keluar";
  nomor_surat: string;
  perihal: string;
  tanggal_surat: string;
  tanggal_terima?: string;
  instansi_id: string;
  instansi_nama: string;
  klasifikasi_kode?: string;
  sifat_kode?: string;
  access_level: "public" | "restricted" | "secret";
  created_at: string;
}

export interface SuratCursor {
  created_at: string;
  id: string;
}

export interface SuratListResponse {
  items: SuratListItem[];
  next_cursor?: SuratCursor;
}

export interface SuratAttachment {
  id: string;
  role: "primary" | "lampiran";
  file_name: string;
  file_size: number;
  mime_type: string;
  uploaded_at: string;
}

export interface SuratReference {
  id: string;
  to_surat_id?: string;
  to_nomor_surat?: string;
  to_perihal?: string;
  relationship: "balasan" | "lanjutan" | "disposisi_hasil" | "revisi" | "terkait";
  external_ref?: string;
  note?: string;
  created_at: string;
}

export interface SuratTembusan {
  id: string;
  instansi_id?: string;
  instansi_nama?: string;
  external_text?: string;
  urutan: number;
}

export interface SuratDetail extends SuratListItem {
  deskripsi_klasifikasi?: string;
  nama_sifat?: string;
  attachments: SuratAttachment[];
  predecessors: SuratReference[];
  successors: SuratReference[];
  tembusan: SuratTembusan[];
}

export interface ListSuratParams {
  jenis?: "masuk" | "keluar";
  tanggal_dari?: string;
  tanggal_sampai?: string;
  instansi_id?: string;
  klasifikasi_id?: string;
  sifat_id?: string;
  search?: string;
  limit?: number;
  after_id?: string;
  after_created_at?: string;
}

export interface CreateSuratPayload {
  jenis: "masuk" | "keluar";
  nomor_surat: string;
  perihal: string;
  tanggal_surat: string;
  tanggal_terima?: string;
  instansi_id: string;
  klasifikasi_id?: string;
  sifat_id?: string;
  access_level: "public" | "restricted" | "secret";
}

export type UpdateSuratPayload = Partial<CreateSuratPayload>;

export interface AddReferencePayload {
  to_surat_id?: string;
  external_ref?: string;
  relationship: "balasan" | "lanjutan" | "disposisi_hasil" | "revisi" | "terkait";
  note?: string;
}

export interface AddTembusanPayload {
  instansi_id?: string;
  external_text?: string;
  urutan?: number;
}

export type DisposisiStatus = "pending" | "in_progress" | "done" | "cancelled";

export interface Disposisi {
  id: string;
  surat_id: string;
  surat_nomor: string;
  surat_perihal: string;
  assigned_to: string;
  assignee_name: string;
  nomor_disposisi?: string;
  instruksi: string;
  deadline?: string;
  status: DisposisiStatus;
  completed_at?: string;
  created_by: string;
  creator_name: string;
  created_at: string;
  updated_at: string;
}

export interface CreateDisposisiPayload {
  surat_id: string;
  assigned_to: string;
  nomor_disposisi?: string;
  instruksi: string;
  deadline?: string;
}

export interface UpdateDisposisiPayload {
  status: DisposisiStatus;
  instruksi?: string;
}

export interface AssignableUser {
  id: string;
  username: string;
  full_name: string;
  roles: string[];
}

export interface Komentar {
  id: string;
  surat_id: string;
  user_id: string;
  user_name: string;
  body: string;
  created_at: string;
}

export const suratApi = {
  list(params: ListSuratParams = {}): Promise<SuratListResponse> {
    const qs = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== "") {
        qs.set(k, String(v));
      }
    }
    const path = qs.toString() ? `/api/surat?${qs}` : "/api/surat";
    return apiClient.get<SuratListResponse>(path);
  },

  get(id: string): Promise<SuratDetail> {
    return apiClient.get<SuratDetail>(`/api/surat/${id}`);
  },

  create(p: CreateSuratPayload): Promise<{ id: string }> {
    return apiClient.post<{ id: string }>("/api/surat", p);
  },

  update(id: string, p: UpdateSuratPayload): Promise<{ status: string }> {
    return apiClient.patch<{ status: string }>(`/api/surat/${id}`, p);
  },

  remove(id: string): Promise<{ status: string }> {
    return apiClient.delete<{ status: string }>(`/api/surat/${id}`);
  },

  // Upload attachments via multipart. Field name "primary" untuk PDF utama,
  // selain itu di-treat sebagai "lampiran".
  async uploadAttachments(
    id: string,
    files: { file: File; role: "primary" | "lampiran" }[],
  ): Promise<{ uploaded: { id: string; file_name: string }[] }> {
    const fd = new FormData();
    for (const { file, role } of files) {
      fd.append(role, file, file.name);
    }
    const authRaw = localStorage.getItem("surat-kec-auth");
    const headers: Record<string, string> = {};
    if (authRaw) {
      try {
        const auth = JSON.parse(authRaw);
        if (auth.accessToken) headers["Authorization"] = `Bearer ${auth.accessToken}`;
      } catch { /* ignore */ }
    }
    const resp = await fetch(`/api/surat/${id}/attachments`, {
      method: "POST",
      body: fd,
      headers,
    });
    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(`Upload gagal (${resp.status}): ${text}`);
    }
    return resp.json();
  },

  attachmentDownloadURL(suratID: string, attID: string): string {
    return `/api/surat/${suratID}/attachments/${attID}`;
  },

  attachmentPreviewURL(suratID: string, attID: string): string {
    return `/api/surat/${suratID}/attachments/${attID}/preview`;
  },

  // Fetch preview blob dengan auth header — return blob URL untuk embed di iframe.
  // Caller bertanggungjawab URL.revokeObjectURL setelah selesai.
  async fetchAttachmentPreviewBlobURL(suratID: string, attID: string): Promise<string> {
    const authRaw = localStorage.getItem("surat-kec-auth");
    let token = "";
    if (authRaw) {
      try {
        token = JSON.parse(authRaw).accessToken ?? "";
      } catch { /* ignore */ }
    }
    const resp = await fetch(`/api/surat/${suratID}/attachments/${attID}/preview`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!resp.ok) {
      throw new Error(`Preview fetch gagal: ${resp.status}`);
    }
    const blob = await resp.blob();
    return URL.createObjectURL(blob);
  },

  addReference(id: string, p: AddReferencePayload): Promise<{ id: string }> {
    return apiClient.post<{ id: string }>(`/api/surat/${id}/references`, p);
  },

  removeReference(suratID: string, refID: string): Promise<{ status: string }> {
    return apiClient.delete<{ status: string }>(`/api/surat/${suratID}/references/${refID}`);
  },

  addTembusan(id: string, p: AddTembusanPayload): Promise<{ id: string }> {
    return apiClient.post<{ id: string }>(`/api/surat/${id}/tembusan`, p);
  },

  removeTembusan(suratID: string, tembusanID: string): Promise<{ status: string }> {
    return apiClient.delete<{ status: string }>(`/api/surat/${suratID}/tembusan/${tembusanID}`);
  },
};

export const disposisiApi = {
  create(p: CreateDisposisiPayload): Promise<{ id: string }> {
    return apiClient.post<{ id: string }>("/api/disposisi", p);
  },

  update(id: string, p: UpdateDisposisiPayload): Promise<{ status: string }> {
    return apiClient.patch<{ status: string }>(`/api/disposisi/${id}`, p);
  },

  list(params: {
    surat_id?: string;
    assigned_to?: string;
    created_by?: string;
    status?: DisposisiStatus;
    mine?: boolean;
  } = {}): Promise<{ items: Disposisi[] }> {
    const qs = new URLSearchParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== "") {
        qs.set(k, String(v));
      }
    }
    const path = qs.toString() ? `/api/disposisi?${qs}` : "/api/disposisi";
    return apiClient.get<{ items: Disposisi[] }>(path);
  },

  listAssignableUsers(): Promise<{ items: AssignableUser[] }> {
    return apiClient.get<{ items: AssignableUser[] }>("/api/users/assignable");
  },
};

export interface DashboardCamatStats {
  surat_masuk_hari_ini: number;
  disposisi_belum_assign: number;
  disposisi_overdue: number;
  disposisi_assigned_to_me: number;
}

export interface NotificationItem {
  id: string;
  type: "disposisi_baru" | "komentar_baru";
  payload: Record<string, unknown>;
  read_at?: string;
  created_at: string;
}

export interface NotificationListResponse {
  items: NotificationItem[];
  unread: number;
}

export const notificationApi = {
  list(unreadOnly = false): Promise<NotificationListResponse> {
    const path = unreadOnly ? "/api/notifications?unread=true" : "/api/notifications";
    return apiClient.get<NotificationListResponse>(path);
  },

  markRead(id: string): Promise<{ status: string }> {
    return apiClient.patch<{ status: string }>(`/api/notifications/${id}/read`, {});
  },

  markAllRead(): Promise<{ status: string }> {
    return apiClient.post<{ status: string }>("/api/notifications/read-all", {});
  },
};

export const dashboardApi = {
  camat(): Promise<DashboardCamatStats> {
    return apiClient.get<DashboardCamatStats>("/api/dashboard/camat");
  },
};

export const komentarApi = {
  list(suratID: string): Promise<{ items: Komentar[] }> {
    return apiClient.get<{ items: Komentar[] }>(`/api/surat/${suratID}/komentar`);
  },

  append(suratID: string, body: string): Promise<{ id: string }> {
    return apiClient.post<{ id: string }>(`/api/surat/${suratID}/komentar`, { body });
  },
};

export interface InstansiItem {
  id: string;
  nama_kanonik: string;
  aliases: string[];
  alamat?: string;
  kontak?: string;
}

export interface LookupItem {
  id: string;
  kode: string;
  nama: string;
  deskripsi?: string;
}

export const direktoriApi = {
  searchInstansi(q: string, limit = 20): Promise<{ items: InstansiItem[] }> {
    const qs = new URLSearchParams({ limit: String(limit) });
    if (q) qs.set("q", q);
    return apiClient.get<{ items: InstansiItem[] }>(`/api/instansi?${qs}`);
  },

  createInstansi(p: { nama_kanonik: string; aliases?: string[]; alamat?: string; kontak?: string }): Promise<{ id: string }> {
    return apiClient.post<{ id: string }>("/api/instansi", { aliases: [], ...p });
  },

  listKlasifikasi(): Promise<{ items: LookupItem[] }> {
    return apiClient.get<{ items: LookupItem[] }>("/api/klasifikasi");
  },

  listSifat(): Promise<{ items: LookupItem[] }> {
    return apiClient.get<{ items: LookupItem[] }>("/api/sifat");
  },
};
