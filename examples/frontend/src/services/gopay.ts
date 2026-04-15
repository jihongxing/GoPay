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

export class GopayService {
  private client: AxiosInstance;
  private appId: string;

  constructor(baseURL: string, appId: string) {
    this.appId = appId;
    this.client = axios.create({
      baseURL,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  async createOrder(req: CreateOrderRequest): Promise<CreateOrderResponse> {
    const response = await this.client.post('/api/v1/checkout', {
      ...req,
      app_id: this.appId,
    });

    if (response.data.code !== 0) {
      throw new Error(response.data.message || 'Create order failed');
    }

    return response.data.data;
  }

  async queryOrder(orderNo: string): Promise<Order> {
    const response = await this.client.get(`/api/v1/orders/${orderNo}`);

    if (response.data.code !== 0) {
      throw new Error(response.data.message || 'Query order failed');
    }

    return response.data.data;
  }
}
