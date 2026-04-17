import axios, { AxiosInstance } from 'axios';
import { createHmac, randomUUID } from 'crypto';
import {
  GopayClientConfig,
  CreateOrderRequest,
  CreateOrderResponse,
  Order,
  ApiResponse,
} from './types';

export class GopayClient {
  private client: AxiosInstance;
  private appId: string;
  private appSecret: string;

  constructor(config: GopayClientConfig) {
    this.appId = config.appId;
    this.appSecret = config.appSecret;
    this.client = axios.create({
      baseURL: config.baseURL,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  /**
   * 生成签名头
   */
  private sign(body: string): Record<string, string> {
    const timestamp = Math.floor(Date.now() / 1000).toString();
    const nonce = randomUUID();
    const message = body + '\n' + timestamp + '\n' + nonce;
    const signature = createHmac('sha256', this.appSecret)
      .update(message)
      .digest('hex');

    return {
      'X-App-ID': this.appId,
      'X-Timestamp': timestamp,
      'X-Nonce': nonce,
      'X-Signature': signature,
    };
  }

  /**
   * 创建支付订单
   */
  async createOrder(req: CreateOrderRequest): Promise<CreateOrderResponse> {
    const body = JSON.stringify({ ...req, app_id: this.appId });
    const headers = this.sign(body);

    const response = await this.client.post<ApiResponse<CreateOrderResponse>>(
      '/api/v1/checkout',
      body,
      { headers }
    );

    if (response.data.code !== 'SUCCESS') {
      throw new Error(`[${response.data.code}] ${response.data.message}`);
    }

    return response.data.data;
  }

  /**
   * 查询订单状态
   */
  async queryOrder(orderNo: string): Promise<Order> {
    const headers = this.sign('');

    const response = await this.client.get<ApiResponse<Order>>(
      `/api/v1/orders/${orderNo}`,
      { headers }
    );

    if (response.data.code !== 'SUCCESS') {
      throw new Error(`[${response.data.code}] ${response.data.message}`);
    }

    return response.data.data;
  }
}
