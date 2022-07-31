package api

type API struct {
	Upcheck  func() bool
	Generate func(number uint) ([]string, error)
}
