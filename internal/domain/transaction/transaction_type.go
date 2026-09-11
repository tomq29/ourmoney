package transaction

type TransactionType int

const (
	Income TransactionType = iota
	Expense
	Transfer
)

var TransactionTypeName = map[TransactionType]string{
	Income:   "income",
	Expense:  "expense",
	Transfer: "transfer",
}
