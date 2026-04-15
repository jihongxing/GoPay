"""
创建支付订单示例
"""

import os
from dotenv import load_dotenv
from gopay_client import GopayClient

# 加载环境变量
load_dotenv()


def main():
    # 创建客户端
    client = GopayClient(
        base_url=os.getenv('GOPAY_URL', 'http://localhost:8080'),
        app_id=os.getenv('APP_ID', '')
    )

    if not client.app_id:
        print('错误: APP_ID 未配置')
        return

    # 创建订单
    print('创建支付订单...')

    try:
        response = client.create_order(
            out_trade_no=f'PY_ORDER_{os.getpid()}',
            amount=100,  # 1元 = 100分
            subject='测试商品',
            body='这是一个 Python 客户端测试订单',
            channel=os.getenv('CHANNEL', 'wechat_native'),
            notify_url=os.getenv('NOTIFY_URL', 'http://localhost:8081/callback')
        )

        print('✅ 订单创建成功!')
        print(f'订单号: {response["order_no"]}')

        if 'pay_url' in response:
            print(f'支付链接: {response["pay_url"]}')

        if 'qr_code' in response:
            print(f'二维码: {response["qr_code"]}')
            print('请使用微信扫描二维码完成支付')

        if 'prepay_id' in response:
            print(f'预支付ID: {response["prepay_id"]}')

    except Exception as e:
        print(f'❌ 创建订单失败: {e}')


if __name__ == '__main__':
    main()
