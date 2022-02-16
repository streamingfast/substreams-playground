package exchange

import "math/big"

func orDie(err error) {
	if err != nil {
		panic("error: " + err.Error())
	}
}

func floatToStr(f *big.Float) string {
	return f.Text('g', -1)
}
