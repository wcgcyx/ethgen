package api

type API struct {
	Upcheck  func() bool
	Generate func(number uint, tokenWeight uint, txWeight uint) ([]string, error)
}
