-- 为存量环境补齐 app_id + out_trade_no 的数据库级幂等约束
CREATE UNIQUE INDEX IF NOT EXISTS uk_orders_app_out_trade_no
ON orders(app_id, out_trade_no);
