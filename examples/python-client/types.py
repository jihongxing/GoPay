"""
类型定义
"""

from typing import TypedDict, Optional, Dict, Any
from datetime import datetime


class CreateOrderRequest(TypedDict, total=False):
    """创建订单请求"""
    app_id: str
    out_trade_no: str
    amount: int
    currency: str
    subject: str
    body: Optional[str]
    channel: str
    notify_url: str
    extra_data: Optional[Dict[str, Any]]


class CreateOrderResponse(TypedDict, total=False):
    """创建订单响应"""
    order_no: str
    pay_url: Optional[str]
    qr_code: Optional[str]
    prepay_id: Optional[str]
    pay_info: Optional[Dict[str, Any]]


class Order(TypedDict, total=False):
    """订单信息"""
    order_no: str
    app_id: str
    out_trade_no: str
    amount: int
    currency: str
    subject: str
    body: str
    channel: str
    status: str
    paid_at: Optional[str]
    created_at: str
    updated_at: str


class CallbackData(TypedDict):
    """回调数据"""
    order_no: str
    out_trade_no: str
    amount: int
    currency: str
    channel: str
    status: str
    paid_at: Optional[str]
