"""
GoPay Python 客户端

提供简单易用的 API 来接入 GoPay 支付网关。
所有请求自动携带 HMAC-SHA256 签名。
"""

import hmac
import hashlib
import json
import time
import uuid
import requests
from typing import Dict, Any, Optional


class GopayClient:
    """GoPay 客户端"""

    def __init__(self, base_url: str, app_id: str, app_secret: str, timeout: int = 30):
        """
        初始化客户端

        Args:
            base_url: GoPay 服务地址
            app_id: 应用 ID
            app_secret: 应用密钥（用于签名）
            timeout: 请求超时时间（秒）
        """
        self.base_url = base_url.rstrip('/')
        self.app_id = app_id
        self.app_secret = app_secret
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({
            'Content-Type': 'application/json',
            'User-Agent': 'GoPay-Python-Client/2.0'
        })

    def _sign(self, body: str) -> Dict[str, str]:
        """生成签名头"""
        timestamp = str(int(time.time()))
        nonce = str(uuid.uuid4())
        message = body + "\n" + timestamp + "\n" + nonce
        signature = hmac.new(
            self.app_secret.encode(),
            message.encode(),
            hashlib.sha256
        ).hexdigest()

        return {
            'X-App-ID': self.app_id,
            'X-Timestamp': timestamp,
            'X-Nonce': nonce,
            'X-Signature': signature,
        }

    def create_order(
        self,
        out_trade_no: str,
        amount: int,
        subject: str,
        channel: str,
        notify_url: str = '',
        body: Optional[str] = None,
        extra_data: Optional[Dict[str, Any]] = None
    ) -> Dict[str, Any]:
        """
        创建支付订单

        Args:
            out_trade_no: 商户订单号（同一 app_id 下唯一）
            amount: 支付金额（单位：分）
            subject: 订单标题
            channel: 支付渠道
            notify_url: 自定义回调地址（可选）
            body: 订单描述
            extra_data: 渠道特定参数

        Returns:
            dict: {"order_no": "...", "pay_url": "...", "qr_code": "..."}

        Raises:
            GopayError: 创建订单失败
        """
        data = {
            'app_id': self.app_id,
            'out_trade_no': out_trade_no,
            'amount': amount,
            'subject': subject,
            'channel': channel,
        }
        if notify_url:
            data['notify_url'] = notify_url
        if body:
            data['body'] = body
        if extra_data:
            data['extra_data'] = extra_data

        return self._post('/api/v1/checkout', data)

    def query_order(self, order_no: str) -> Dict[str, Any]:
        """
        查询订单状态

        Args:
            order_no: GoPay 订单号

        Returns:
            dict: 订单信息

        Raises:
            GopayError: 查询失败
        """
        return self._get(f'/api/v1/orders/{order_no}')

    def _post(self, path: str, data: Dict[str, Any]) -> Dict[str, Any]:
        """发送带签名的 POST 请求"""
        url = f'{self.base_url}{path}'
        body = json.dumps(data, ensure_ascii=False)
        headers = self._sign(body)

        try:
            resp = self.session.post(url, data=body, headers=headers, timeout=self.timeout)
            return self._handle_response(resp)
        except requests.RequestException as e:
            raise GopayError(f'Request failed: {str(e)}')

    def _get(self, path: str) -> Dict[str, Any]:
        """发送带签名的 GET 请求"""
        url = f'{self.base_url}{path}'
        headers = self._sign('')

        try:
            resp = self.session.get(url, headers=headers, timeout=self.timeout)
            return self._handle_response(resp)
        except requests.RequestException as e:
            raise GopayError(f'Request failed: {str(e)}')

    def _handle_response(self, resp: requests.Response) -> Dict[str, Any]:
        """处理响应"""
        result = resp.json()

        if result.get('code') != 'SUCCESS':
            code = result.get('code', 'UNKNOWN')
            message = result.get('message', 'Unknown error')
            details = result.get('details', {})
            raise GopayError(f'[{code}] {message}', code=code, details=details)

        return result.get('data', {})


class GopayError(Exception):
    """GoPay 错误"""

    def __init__(self, message: str, code: str = '', details: dict = None):
        super().__init__(message)
        self.code = code
        self.details = details or {}
