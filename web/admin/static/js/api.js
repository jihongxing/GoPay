// API 基础配置
const API_BASE = '/admin';

// API 封装
const API = {
    // 通用请求方法
    async request(url, options = {}) {
        const defaultOptions = {
            headers: {
                'Content-Type': 'application/json',
            },
        };

        const response = await fetch(API_BASE + url, {
            ...defaultOptions,
            ...options,
        });

        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }

        const data = await response.json();

        if (data.code !== 'SUCCESS') {
            throw new Error(data.message || '请求失败');
        }

        return data.data;
    },

    // 统计数据
    async getStats() {
        return this.request('/stats');
    },

    // 订单相关
    async getFailedOrders(params) {
        const query = new URLSearchParams(params).toString();
        return this.request(`/orders/failed?${query}`);
    },

    async searchOrder(outTradeNo) {
        return this.request(`/orders/search?out_trade_no=${encodeURIComponent(outTradeNo)}`);
    },

    async getOrderDetail(orderNo) {
        return this.request(`/orders/${orderNo}`);
    },

    async retryOrder(orderNo) {
        return this.request(`/orders/${orderNo}/retry`, {
            method: 'POST',
        });
    },

    async batchRetry(orderNos) {
        return this.request('/orders/batch-retry', {
            method: 'POST',
            body: JSON.stringify({ order_nos: orderNos }),
        });
    },

    // 对账相关
    async getReconciliationReports(params) {
        const query = new URLSearchParams(params).toString();
        return this.request(`/reconciliation/reports?${query}`);
    },

    async getReconciliationDetail(reportId) {
        return this.request(`/reconciliation/${reportId}`);
    },

    // 统计相关
    async getOrderStats(params) {
        const query = new URLSearchParams(params).toString();
        return this.request(`/stats/orders?${query}`);
    },

    async getNotificationStats(params) {
        const query = new URLSearchParams(params).toString();
        return this.request(`/stats/notifications?${query}`);
    },
};
