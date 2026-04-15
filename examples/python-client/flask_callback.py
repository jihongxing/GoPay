"""
Flask 回调服务器示例
"""

import os
from flask import Flask, request, jsonify
from dotenv import load_dotenv

# 加载环境变量
load_dotenv()

app = Flask(__name__)


@app.route('/callback', methods=['POST'])
def callback():
    """处理支付回调"""
    try:
        data = request.get_json()

        print('📨 收到支付回调:')
        print(f'  订单号: {data["order_no"]}')
        print(f'  商户订单号: {data["out_trade_no"]}')
        print(f'  订单状态: {data["status"]}')
        print(f'  支付金额: {data["amount"]} 分')
        print(f'  支付渠道: {data["channel"]}')

        # 处理业务逻辑
        if data['status'] == 'paid':
            print('✅ 订单支付成功，可以进行后续业务处理')
            # TODO: 实现你的业务逻辑
            # 例如：更新订单状态、发货、发送通知等

        # 返回成功响应
        return jsonify({
            'code': 'SUCCESS',
            'message': 'OK'
        })

    except Exception as e:
        print(f'❌ 处理回调失败: {e}')
        return jsonify({
            'code': 'ERROR',
            'message': str(e)
        }), 500


@app.route('/health', methods=['GET'])
def health():
    """健康检查"""
    return jsonify({'status': 'ok'})


if __name__ == '__main__':
    port = int(os.getenv('CALLBACK_PORT', 8081))
    print(f'🚀 Flask 回调服务器启动在端口 {port}')
    print(f'回调地址: http://localhost:{port}/callback')
    app.run(host='0.0.0.0', port=port, debug=True)
