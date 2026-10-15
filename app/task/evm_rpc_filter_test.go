package task

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/v03413/bepusdt/app/conf"
	"github.com/v03413/bepusdt/app/model"
)

// 模拟拒绝全链事件请求的 RPC，验证过滤后仍能解析真实格式的 USDT 转账。
func TestEVMLogsUseNetworkContractFilter(t *testing.T) {
	if err := model.Init(filepath.Join(t.TempDir(), "rpc.db"), ""); err != nil {
		t.Fatal(err)
	}
	defer model.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string `json:"method"`
			Params []struct {
				Address []string `json:"address"`
				From    string   `json:"fromBlock"`
				To      string   `json:"toBlock"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if request.Method != "eth_getLogs" || len(request.Params) != 1 || len(request.Params[0].Address) == 0 {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32701,"message":"address required"}}`))
			return
		}
		filter := request.Params[0]
		contracts := make(map[string]bool)
		for _, address := range filter.Address {
			contracts[address] = true
		}
		if !contracts[strings.ToLower(conf.UsdtBep20)] || !contracts[strings.ToLower(conf.UsdcBep20)] || len(contracts) != 2 || filter.From != "0x2a" || filter.To != "0x2a" {
			t.Errorf("合约或区块过滤错误：%+v", filter)
		}
		response := map[string]interface{}{"jsonrpc": "2.0", "id": 1, "result": []interface{}{map[string]interface{}{
			"address": strings.ToLower(conf.UsdtBep20), "blockNumber": "0x2a", "transactionHash": "0x1234", "data": "0xde0b6b3a7640000",
			"topics": []string{evmTransferEvent, "0x" + strings.Repeat("0", 24) + strings.Repeat("1", 40), "0x" + strings.Repeat("0", 24) + strings.Repeat("2", 40)},
		}}}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	if err := model.Db.Model(&model.Conf{}).Where("k = ?", model.RpcEndpointBsc).Update("v", server.URL).Error; err != nil {
		t.Fatal(err)
	}
	model.RefreshC()
	scanner := evm{Network: conf.Bsc, Client: server.Client()}
	transfers, err := scanner.parseEventTransfer(evmBlock{From: 42, To: 42}, map[string]time.Time{"0x2a": time.Unix(1000, 0)})
	if err != nil {
		t.Fatal(err)
	}
	if len(transfers) != 1 || transfers[0].TradeType != model.UsdtBep20 || !transfers[0].Amount.Equal(decimal.NewFromInt(1)) || transfers[0].RecvAddress != "0x"+strings.Repeat("2", 40) {
		t.Fatalf("转账解析结果错误：%+v", transfers)
	}
	unknown := evm{Network: "unsupported", Client: server.Client()}
	if transfers, err := unknown.parseEventTransfer(evmBlock{From: 42, To: 42}, nil); err != nil || len(transfers) != 0 {
		t.Fatalf("未知网络应跳过事件查询：%v", err)
	}
}
