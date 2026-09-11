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

var transactionTypeFromString = map[string]TransactionType{
	"income":   Income,
	"expense":  Expense,
	"transfer": Transfer,
}

func TypeFromString(s string) TransactionType {
	return transactionTypeFromString[s]
}
