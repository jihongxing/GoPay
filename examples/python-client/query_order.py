"""
查询订单状态示例
"""

import os
import sys
from dotenv import load_dotenv
from gopay_client import GopayClient

# 加载环境变量
load_dotenv()


def main():
    if len(sys.argv) < 2:
        print('用法: python query_order.py <order_no>')
        sys.exit(1)

    order_no = sys.argv[1]

    # 创建客户端
    client = GopayClient(
        base_url=os.getenv('GOPAY_URL', 'http://localhost:8080'),
        app_id=os.getenv('APP_ID', ''),
        app_secret=os.getenv('APP_SECRET', '')
    )

    if not client.app_id:
        print('错误: APP_ID 未配置')
        return

    if not client.app_secret:
        print('错误: APP_SECRET 未配置')
        return

    # 查询订单
    print(f'查询订单: {order_no}')

    try:
        order = client.query_order(order_no)

        print('✅ 订单查询成功!')
        print(f'订单号: {order["order_no"]}')
        print(f'商户订单号: {order["out_trade_no"]}')
        print(f'订单状态: {order["status"]}')
        print(f'支付金额: {order["amount"]} 分 ({order["amount"]/100:.2f} 元)')
        print(f'支付渠道: {order["channel"]}')
        print(f'创建时间: {order["created_at"]}')

        if order.get('paid_at'):
            print(f'支付时间: {order["paid_at"]}')

    except Exception as e:
        print(f'❌ 查询订单失败: {e}')


if __name__ == '__main__':
    main()
