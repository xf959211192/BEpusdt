package task

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/model"
)

func TestLatePaymentCannotMatchNewInvoice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "late.db")
	if err := model.Init(path, ""); err != nil {
		t.Fatal(err)
	}
	defer model.Close()
	wallet, err := model.NewWallet("0x1111111111111111111111111111111111111111", model.UsdtBep20)
	if err != nil {
		t.Fatal(err)
	}
	p := model.OrderParams{Money: decimal.NewFromInt(10), TradeType: model.UsdtBep20, Fiat: "CNY", OrderId: "old", ApiType: model.OrderApiTypeEpusdt}
	w, oldAmount, err := model.CalcTradeAmount([]model.Wallet{wallet}, decimal.NewFromInt(1), p)
	if err != nil {
		t.Fatal(err)
	}
	old, err := model.BuildOrder(p, model.Trade{Wallet: w, Crypto: model.USDT, Rate: decimal.NewFromInt(1), Amount: oldAmount})
	if err != nil {
		t.Fatal(err)
	}
	old.SetExpired()
	if err := model.Db.Delete(&old).Error; err != nil {
		t.Fatal(err)
	}
	model.Close()
	if err := model.Init(path, ""); err != nil {
		t.Fatal(err)
	}
	p.OrderId = "new"
	w, newAmount, err := model.CalcTradeAmount([]model.Wallet{wallet}, decimal.NewFromInt(1), p)
	if err != nil {
		t.Fatal(err)
	}
	current, err := model.BuildOrder(p, model.Trade{Wallet: w, Crypto: model.USDT, Rate: decimal.NewFromInt(1), Amount: newAmount})
	if err != nil {
		t.Fatal(err)
	}
	event := transfer{TradeType: model.UsdtBep20, RecvAddress: wallet.GetMatchAddr(), Timestamp: time.Now().Add(time.Second), Amount: decimal.RequireFromString(oldAmount)}
	if orderTransferMatch(current, event) {
		t.Fatal("旧付款错误匹配了新订单")
	}
	event.Amount = decimal.RequireFromString(newAmount)
	if !orderTransferMatch(current, event) {
		t.Fatal("有效付款未匹配")
	}
	event.Amount = event.Amount.Sub(decimal.RequireFromString("0.001"))
	if orderTransferMatch(current, event) {
		t.Fatal("少付金额被匹配")
	}
	event.Amount = decimal.RequireFromString(newAmount)
	event.TradeType = model.UsdcBep20
	if orderTransferMatch(current, event) {
		t.Fatal("错误代币被匹配")
	}
	event.TradeType = model.UsdtBep20
	event.Timestamp = current.ExpiredAt.Add(time.Second)
	if orderTransferMatch(current, event) {
		t.Fatal("超时付款被匹配")
	}
}
