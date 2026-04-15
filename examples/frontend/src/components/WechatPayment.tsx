import React, { useState, useEffect } from 'react';
import QRCode from 'qrcode.react';
import { GopayService } from '../services/gopay';

interface WechatPaymentProps {
  appId: string;
  amount: number;
  subject: string;
  body?: string;
  notifyUrl?: string;
  onSuccess?: (orderNo: string) => void;
  onError?: (error: Error) => void;
}

export const WechatPayment: React.FC<WechatPaymentProps> = ({
  appId,
  amount,
  subject,
  body,
  notifyUrl = window.location.origin + '/callback',
  onSuccess,
  onError,
}) => {
  const [qrCode, setQrCode] = useState<string>('');
  const [orderNo, setOrderNo] = useState<string>('');
  const [status, setStatus] = useState<'idle' | 'loading' | 'waiting' | 'success' | 'error'>('idle');
  const [error, setError] = useState<string>('');

  const gopay = new GopayService(
    import.meta.env.VITE_GOPAY_URL || 'http://localhost:8080',
    appId
  );

  const createOrder = async () => {
    setStatus('loading');
    setError('');

    try {
      const response = await gopay.createOrder({
        out_trade_no: `WX_${Date.now()}`,
        amount,
        subject,
        body,
        channel: 'wechat_native',
        notify_url: notifyUrl,
      });

      setQrCode(response.qr_code || '');
      setOrderNo(response.order_no);
      setStatus('waiting');

      // 开始轮询订单状态
      startPolling(response.order_no);
    } catch (err) {
      const error = err as Error;
      setError(error.message);
      setStatus('error');
      onError?.(error);
    }
  };

  const startPolling = (orderNo: string) => {
    const interval = setInterval(async () => {
      try {
        const order = await gopay.queryOrder(orderNo);

        if (order.status === 'paid') {
          clearInterval(interval);
          setStatus('success');
          onSuccess?.(orderNo);
        }
      } catch (err) {
        console.error('轮询订单状态失败:', err);
      }
    }, 2000); // 每2秒查询一次

    // 5分钟后停止轮询
    setTimeout(() => clearInterval(interval), 5 * 60 * 1000);
  };

  return (
    <div className="wechat-payment">
      <h2>微信支付</h2>

      {status === 'idle' && (
        <div>
          <p>订单金额: ¥{(amount / 100).toFixed(2)}</p>
          <p>商品名称: {subject}</p>
          <button onClick={createOrder}>立即支付</button>
        </div>
      )}

      {status === 'loading' && (
        <div className="loading">
          <p>正在创建订单...</p>
        </div>
      )}

      {status === 'waiting' && qrCode && (
        <div className="qr-code">
          <p>请使用微信扫描二维码完成支付</p>
          <QRCode value={qrCode} size={256} />
          <p className="order-no">订单号: {orderNo}</p>
          <p className="amount">金额: ¥{(amount / 100).toFixed(2)}</p>
        </div>
      )}

      {status === 'success' && (
        <div className="success">
          <h3>✅ 支付成功！</h3>
          <p>订单号: {orderNo}</p>
        </div>
      )}

      {status === 'error' && (
        <div className="error">
          <h3>❌ 支付失败</h3>
          <p>{error}</p>
          <button onClick={createOrder}>重试</button>
        </div>
      )}
    </div>
  );
};
