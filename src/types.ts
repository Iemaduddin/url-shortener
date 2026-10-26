export type Page = "dashboard" | "urls" | "url-detail" | "users" | "audit";

export interface APIError {
  message?: string;
}

export interface APIResponse {
  short_url?: string;
  role?: string;
  error?: APIError;
}

export interface URLItem {
  id: string;
  short_url: string;
  original_url: string;
  created_at: string;
  status: string;
  total_clicks: number;
  last_clicked_at?: string;
}

export interface QRConfig {
  foreground: string;
  background: string;
  frame_text: string;
  has_logo: boolean;
  updated_at: string;
}

export interface URLListResponse {
  items: URLItem[];
  page: number;
  page_size: number;
  total_items: number;
}

export interface UserItem {
  id: string;
  email: string;
  role: string;
  status: string;
  created_at: string;
}

export interface UserListResponse {
  items: UserItem[];
  page: number;
  page_size: number;
  total_items: number;
}

export interface AuditLog {
  id: string;
  actor_user_id?: string;
  action: string;
  resource_type: string;
  resource_id?: string;
  metadata: string;
  created_at: string;
}

export interface AuditLogResponse {
  items: AuditLog[];
  page: number;
  page_size: number;
  total_items: number;
}
