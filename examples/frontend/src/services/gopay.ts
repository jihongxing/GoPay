import axios, { AxiosInstance } from 'axios';

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

/**
 * GoPay 前端服务
 *
 * 注意：前端不应直接持有 app_secret。
 * 推荐架构：前端 → 你的后端 → GoPay（签名在你的后端完成）
 *
 * 如果你的场景允许前端直签（如内网管理工具），可以传入 appSecret。
 */
export class GopayService {
  private client: AxiosInstance;
  private appId: string;
  private appSecret: string;

  constructor(baseURL: string, appId: string, appSecret: string = '') {
    this.appId = appId;
    this.appSecret = appSecret;
    this.client = axios.create({
      baseURL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  /**
   * 生成 HMAC-SHA256 签名（需要浏览器支持 SubtleCrypto）
   */
  private async sign(body: string): Promise<Record<string, string>> {
    const timestamp = Math.floor(Date.now() / 1000).toString();
    const nonce = crypto.randomUUID();
    const message = body + '\n' + timestamp + '\n' + nonce;

    const encoder = new TextEncoder();
    const key = await crypto.subtle.importKey(
      'raw',
      encoder.encode(this.appSecret),
      { name: 'HMAC', hash: 'SHA-256' },
      false,
      ['sign']
    );
    const sig = await crypto.subtle.sign('HMAC', key, encoder.encode(message));
    const signature = Array.from(new Uint8Array(sig))
      .map(b => b.toString(16).padStart(2, '0'))
      .join('');

    return {
      'X-App-ID': this.appId,
      'X-Timestamp': timestamp,
      'X-Nonce': nonce,
      'X-Signature': signature,
    };
  }

  async createOrder(req: CreateOrderRequest): Promise<CreateOrderResponse> {
    const body = JSON.stringify({ ...req, app_id: this.appId });
    const headers = this.appSecret ? await this.sign(body) : {};

    const response = await this.client.post('/api/v1/checkout', body, { headers });

    if (response.data.code !== 'SUCCESS') {
      throw new Error(`[${response.data.code}] ${response.data.message}`);
    }

    return response.data.data;
  }

  async queryOrder(orderNo: string): Promise<Order> {
    const headers = this.appSecret ? await this.sign('') : {};

    const response = await this.client.get(`/api/v1/orders/${orderNo}`, { headers });

    if (response.data.code !== 'SUCCESS') {
      throw new Error(`[${response.data.code}] ${response.data.message}`);
    }

    return response.data.data;
  }
}
