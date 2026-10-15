package model

import "gorm.io/gorm/clause"

const amountReservationMaxSteps = 10000

// AmountReservation 独立于订单保留，不随订单删除、取消或过期而释放。
// 此补丁用于全新网关；已有网关迁移必须先回填全部历史金额。
type AmountReservation struct {
	TradeType TradeType `gorm:"primaryKey;type:varchar(20)"`
	Address   string    `gorm:"primaryKey;type:varchar(128)"`
	Amount    string    `gorm:"primaryKey;type:varchar(32)"`
}

func (AmountReservation) TableName() string { return "bep_amount_reservation" }

func reserveAmount(wallet Wallet, tradeType TradeType, amount string) (bool, error) {
	row := AmountReservation{TradeType: tradeType, Address: wallet.GetMatchAddr(), Amount: amount}
	result := Db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	return result.RowsAffected == 1, result.Error
}
