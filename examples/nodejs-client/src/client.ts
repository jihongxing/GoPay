import axios, { AxiosInstance } from 'axios';
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

  constructor(config: GopayClientConfig) {
    this.appId = config.appId;
    this.client = axios.create({
      baseURL: config.baseURL,
      timeout: config.timeout || 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });
  }

  /**
   * 创建支付订单
   */
  async createOrder(req: CreateOrderRequest): Promise<CreateOrderResponse> {
    const response = await this.client.post<ApiResponse<CreateOrderResponse>>(
      '/api/v1/checkout',
      {
        ...req,
        app_id: this.appId,
      }
    );

    if (response.data.code !== 0) {
      throw new Error(response.data.message || 'Create order failed');
    }

    return response.data.data;
  }

  /**
   * 查询订单状态
   */
  async queryOrder(orderNo: string): Promise<Order> {
    const response = await this.client.get<ApiResponse<Order>>(
      `/api/v1/orders/${orderNo}`
    );

    if (response.data.code !== 0) {
      throw new Error(response.data.message || 'Query order failed');
    }

    return response.data.data;
  }
}
