import React, { useState } from 'react';
import QRCode from 'qrcode.react';
import { GopayService } from '../services/gopay';

interface AlipayPaymentProps {
  appId: string;
  amount: number;
  subject: string;
  body?: string;
  channel?: 'alipay_qr' | 'alipay_wap';
  notifyUrl?: string;
  onSuccess?: (orderNo: string) => void;
  onError?: (error: Error) => void;
}

export const AlipayPayment: React.FC<AlipayPaymentProps> = ({
  appId,
  amount,
  subject,
  body,
  channel = 'alipay_qr',
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
        out_trade_no: `ALI_${Date.now()}`,
        amount,
        subject,
        body,
        channel,
        notify_url: notifyUrl,
      });

      setOrderNo(response.order_no);

      if (channel === 'alipay_qr' && response.qr_code) {
        // 扫码支付
        setQrCode(response.qr_code);
        setStatus('waiting');
        startPolling(response.order_no);
      } else if (channel === 'alipay_wap' && response.pay_url) {
        // 手机网站支付，直接跳转
        window.location.href = response.pay_url;
      }
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
    }, 2000);

    setTimeout(() => clearInterval(interval), 5 * 60 * 1000);
  };

  return (
    <div className="alipay-payment">
      <h2>支付宝支付</h2>

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
          <p>请使用支付宝扫描二维码完成支付</p>
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
