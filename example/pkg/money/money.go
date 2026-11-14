// Package money handles amounts of money in cents.
package money

import "fmt"

// Amount is a sum of money in cents.
type Amount int64

// String formats a as dollars, such as "$12.50".
func (a Amount) String() string {
	sign := ""
	if a < 0 {
		sign, a = "-", -a
	}
	return fmt.Sprintf("%s$%d.%02d", sign, a/100, a%100)
}

// Sum adds amounts.
func Sum(amounts ...Amount) Amount {
	var total Amount
	for _, a := range amounts {
		total += a
	}
	return total
}
