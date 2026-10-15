package model

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestAmountReservationSurvivesOrderDeletionAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.db")
	if err := Init(path, ""); err != nil {
		t.Fatal(err)
	}
	defer Close()
	wallet, err := NewWallet("0x1111111111111111111111111111111111111111", UsdtBep20)
	if err != nil {
		t.Fatal(err)
	}
	p := OrderParams{Money: decimal.NewFromInt(10), TradeType: UsdtBep20, Fiat: "CNY", OrderId: "old", ApiType: OrderApiTypeEpusdt}
	w, first, err := CalcTradeAmount([]Wallet{wallet}, decimal.NewFromInt(1), p)
	if err != nil {
		t.Fatal(err)
	}
	order, err := BuildOrder(p, Trade{Wallet: w, Crypto: USDT, Rate: decimal.NewFromInt(1), Amount: first})
	if err != nil {
		t.Fatal(err)
	}
	order.ExpiredAt = time.Now().Add(-time.Minute)
	order.SetExpired()
	_, second, err := CalcTradeAmount([]Wallet{wallet}, decimal.NewFromInt(1), p)
	if err != nil || first == second {
		t.Fatalf("过期金额被复用：%s %s %v", first, second, err)
	}
	if err := Db.Delete(&order).Error; err != nil {
		t.Fatal(err)
	}
	Close()
	if err := Init(path, ""); err != nil {
		t.Fatal(err)
	}
	_, third, err := CalcTradeAmount([]Wallet{wallet}, decimal.NewFromInt(1), p)
	if err != nil || third == first || third == second {
		t.Fatalf("删除或重启后复用了金额：%s %v", third, err)
	}

	// 同一数据库上的并发分配必须得到不同金额，写入失败则拒绝分配。
	var wg sync.WaitGroup
	results := make(chan string, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, value, err := CalcTradeAmount([]Wallet{wallet}, decimal.NewFromInt(1), p)
			if err != nil {
				results <- "error:" + err.Error()
				return
			}
			results <- value
		}()
	}
	wg.Wait()
	close(results)
	seen := map[string]bool{first: true, second: true, third: true}
	for value := range results {
		if len(value) > 6 && value[:6] == "error:" {
			t.Fatal(value)
		}
		if seen[value] {
			t.Fatal("并发分配重复金额：" + value)
		}
		seen[value] = true
	}

	// 额度耗尽时明确失败，不能绕回已经使用的金额。
	rows := make([]AmountReservation, 0, amountReservationMaxSteps+1)
	for i := 0; i <= amountReservationMaxSteps; i++ {
		amount := decimal.NewFromInt(20).Add(decimal.NewFromInt(int64(i)).Mul(decimal.RequireFromString("0.01"))).String()
		rows = append(rows, AmountReservation{TradeType: UsdtBep20, Address: wallet.GetMatchAddr(), Amount: amount})
	}
	if err := Db.CreateInBatches(rows, 200).Error; err != nil {
		t.Fatal(err)
	}
	p.Money = decimal.NewFromInt(20)
	if _, _, err := CalcTradeAmount([]Wallet{wallet}, decimal.NewFromInt(1), p); err == nil {
		t.Fatal("金额耗尽后没有拒绝订单")
	}
	if err := Db.Migrator().DropTable(&AmountReservation{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := CalcTradeAmount([]Wallet{wallet}, decimal.NewFromInt(1), p); err == nil {
		t.Fatal("保留表损坏时没有拒绝订单")
	}
}
