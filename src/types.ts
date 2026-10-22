export type Page = "dashboard" | "urls" | "users";

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
}

export interface UserItem {
  id: string;
  email: string;
  role: string;
  status: string;
  created_at: string;
}
