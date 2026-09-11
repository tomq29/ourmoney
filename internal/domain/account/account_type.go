package account

type AccountType int

const (
	Cash AccountType = iota
	BankCard
	Investment
	Deposit
)

var AccountTypeName = map[AccountType]string{
	Cash:       "cash",
	BankCard:   "bank_card",
	Investment: "investment",
	Deposit:    "deposit",
}

var AccountTypeConverterFromString = map[string]AccountType{
	"cash":       Cash,
	"bank_card":  BankCard,
	"investment": Investment,
	"deposit":    Deposit,
}
