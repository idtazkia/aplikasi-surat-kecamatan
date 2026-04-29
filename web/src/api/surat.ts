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

export interface SuratDetail extends SuratListItem {
  deskripsi_klasifikasi?: string;
  nama_sifat?: string;
  attachments: SuratAttachment[];
  predecessors: SuratReference[];
  successors: SuratReference[];
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
};
