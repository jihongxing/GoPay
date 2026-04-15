"""
GoPay Python 客户端

提供简单易用的 API 来接入 GoPay 支付网关
"""

import requests
from typing import Dict, Any, Optional
from .types import CreateOrderRequest, CreateOrderResponse, Order


class GopayClient:
    """GoPay 客户端"""

    def __init__(self, base_url: str, app_id: str, timeout: int = 30):
        """
        初始化客户端

        Args:
            base_url: GoPay 服务地址
            app_id: 应用ID
            timeout: 请求超时时间（秒）
        """
        self.base_url = base_url.rstrip('/')
        self.app_id = app_id
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json',
            'User-Agent': 'GoPay-Python-Client/1.0'
        })

    def create_order(
        self,
        out_trade_no: str,
        amount: int,
        subject: str,
        channel: str,
        notify_url: str,
        body: Optional[str] = None,
        currency: str = 'CNY',
        extra_data: Optional[Dict[str, Any]] = None
    ) -> CreateOrderResponse:
        """
        创建支付订单

        Args:
            out_trade_no: 商户订单号
            amount: 支付金额（单位：分）
            subject: 订单标题
            channel: 支付渠道
            notify_url: 回调地址
            body: 订单描述
            currency: 货币类型
            extra_data: 额外数据

        Returns:
            CreateOrderResponse: 订单创建响应

        Raises:
            GopayError: 创建订单失败
        """
        data = {
            'app_id': self.app_id,
            'out_trade_no': out_trade_no,
            'amount': amount,
            'subject': subject,
            'channel': channel,
            'notify_url': notify_url,
            'currency': currency,
        }

        if body:
            data['body'] = body
        if extra_data:
            data['extra_data'] = extra_data

        response = self._post('/api/v1/checkout', data)
        return response['data']

    def query_order(self, order_no: str) -> Order:
        """
        查询订单状态

        Args:
            order_no: GoPay 订单号

        Returns:
            Order: 订单信息

        Raises:
            GopayError: 查询订单失败
        """
        response = self._get(f'/api/v1/orders/{order_no}')
        return response['data']

    def _post(self, path: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """发送 POST 请求"""
        url = f'{self.base_url}{path}'

        try:
            resp = self.session.post(url, json=data, timeout=self.timeout)
            resp.raise_for_status()
            result = resp.json()

            if result.get('code') != 0:
                raise GopayError(result.get('message', 'Unknown error'))

            return result
        except requests.RequestException as e:
            raise GopayError(f'Request failed: {str(e)}')

    def _get(self, path: str) -> Dict[str, Any]:
        """发送 GET 请求"""
        url = f'{self.base_url}{path}'

        try:
            resp = self.session.get(url, timeout=self.timeout)
            resp.raise_for_status()
            result = resp.json()

            if result.get('code') != 0:
                raise GopayError(result.get('message', 'Unknown error'))

            return result
        except requests.RequestException as e:
            raise GopayError(f'Request failed: {str(e)}')


class GopayError(Exception):
    """GoPay 错误"""
    pass
