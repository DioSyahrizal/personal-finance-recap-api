package recap

import "strings"

// Category is the canonical vocabulary used when classifying transactions.
type Category string

const (
	CategoryFood          Category = "Food"
	CategoryGroceries     Category = "Groceries"
	CategoryBills         Category = "Bills"
	CategoryTransport     Category = "Transport"
	CategoryEWallet       Category = "E-Wallet"
	CategoryShopping      Category = "Shopping"
	CategoryIncome        Category = "Income"
	CategoryFees          Category = "Fees"
	CategoryTransfer      Category = "Transfer"
	CategoryUncategorized Category = "Uncategorized"
)

// Categories returns the values that may be stored in recap_items.category.
// Returning a new slice prevents callers from changing the package vocabulary.
func Categories() []string {
	return []string{
		string(CategoryFood),
		string(CategoryGroceries),
		string(CategoryBills),
		string(CategoryTransport),
		string(CategoryEWallet),
		string(CategoryShopping),
		string(CategoryIncome),
		string(CategoryFees),
		string(CategoryTransfer),
		string(CategoryUncategorized),
	}
}

// NormalizeCategory keeps old labels compatible with the canonical vocabulary.
func NormalizeCategory(value string) Category {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "f&b", "food & beverage", "food":
		return CategoryFood
	case "groceries":
		return CategoryGroceries
	case "bills":
		return CategoryBills
	case "transport":
		return CategoryTransport
	case "e-wallet", "ewallet", "e wallet":
		return CategoryEWallet
	case "shopping":
		return CategoryShopping
	case "income":
		return CategoryIncome
	case "fees", "fee":
		return CategoryFees
	case "transfer":
		return CategoryTransfer
	default:
		return CategoryUncategorized
	}
}
