package production

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/shopspring/decimal"

	"order-stock/backend/internal/database"
)

// ReportingCurrency is what every production figure is reported in. Amounts
// recorded in another currency are converted with the tenant's rate, so a
// report never adds denominations together.
const ReportingCurrency = "IDR"

// Rates maps a currency code to its value in Rupiah.
type Rates map[string]decimal.Decimal

func loadRates(ctx context.Context, tx database.TenantTx, a Actor) (Rates, error) {
	rates := Rates{}
	var raw []byte
	e := tx.QueryRow(ctx, `SELECT COALESCE(currency_rates,'{}'::jsonb) FROM tenant_settings WHERE tenant_id=$1`, a.TenantID).Scan(&raw)
	if e != nil {
		// A tenant without a settings row simply has no foreign currency.
		if strings.Contains(e.Error(), "no rows") {
			return rates, nil
		}
		return nil, e
	}
	values := map[string]decimal.Decimal{}
	if e = json.Unmarshal(raw, &values); e != nil {
		return nil, e
	}
	for code, rate := range values {
		if rate.IsPositive() {
			rates[strings.ToUpper(strings.TrimSpace(code))] = rate
		}
	}
	return rates, nil
}

// rate returns how many Rupiah one unit of the currency is worth.
func (r Rates) rate(currency string) (decimal.Decimal, bool) {
	code := strings.ToUpper(strings.TrimSpace(currency))
	if code == "" || code == ReportingCurrency {
		return decimal.NewFromInt(1), true
	}
	value, ok := r[code]
	return value, ok
}

// convert restates an amount in Rupiah, refusing to guess a missing rate.
func (r Rates) convert(amount decimal.Decimal, currency string) (decimal.Decimal, error) {
	value, ok := r.rate(currency)
	if !ok {
		return decimal.Zero, invalid("Set a conversion rate for %s before reporting production in that currency", strings.ToUpper(currency))
	}
	if value.Equal(decimal.NewFromInt(1)) {
		return amount, nil
	}
	return amount.Mul(value).Round(6), nil
}

// mustConvert is for aggregation paths that already validated their currencies;
// an unknown rate leaves the amount out rather than inventing a number.
func (r Rates) mustConvert(amount decimal.Decimal, currency string) decimal.Decimal {
	converted, e := r.convert(amount, currency)
	if e != nil {
		return decimal.Zero
	}
	return converted
}
