export interface GopayClientConfig {
  baseURL: string;
  appId: string;
  appSecret: string;
  timeout?: number;
}

export interface CreateOrderRequest {
  out_trade_no: string;
  amount: number;
  subject: string;
  body?: string;
  channel: string;
  notify_url: string;
  extra_data?: Record<string, any>;
}

export interface CreateOrderResponse {
  order_no: string;
  pay_url?: string;
  qr_code?: string;
  prepay_id?: string;
  pay_info?: Record<string, any>;
}

export interface Order {
  order_no: string;
  app_id: string;
  out_trade_no: string;
  amount: number;
  currency: string;
  subject: string;
  body: string;
  channel: string;
  status: string;
  paid_at?: string;
  created_at: string;
  updated_at: string;
}

export interface CallbackData {
  order_no: string;
  out_trade_no: string;
  amount: number;
  status: string;
  channel: string;
  paid_at?: string;
  // 退款通知额外字段
  refund_no?: string;
  refund_amount?: number;
  refunded_at?: string;
}

export interface ApiResponse<T> {
  code: string;
  message: string;
  data: T;
}
